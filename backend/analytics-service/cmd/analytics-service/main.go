package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/config"
	grpcdelivery "github.com/petmatch/petmatch/internal/delivery/grpc"
	"github.com/petmatch/petmatch/internal/delivery/grpc/pb"
	"github.com/petmatch/petmatch/internal/infrastructure/kafka"
	"github.com/petmatch/petmatch/internal/infrastructure/postgres"
	"github.com/petmatch/petmatch/internal/infrastructure/resilience"
	"github.com/petmatch/petmatch/internal/observability"
	"github.com/petmatch/petmatch/internal/platform"
)

func main() {
	configPath := flag.String("config", "", "path to config")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger := observability.NewLogger(cfg.App)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo, err := postgres.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.ErrorContext(ctx, "connect postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer repo.Close()

	rankingPublisher := kafka.NewRankingPublisher(cfg.Kafka.Brokers, cfg.Kafka.RankingFeedbackTopic)
	defer func() { _ = rankingPublisher.Close() }()

	publisher := resilience.NewBreakerPublisher(rankingPublisher)
	service := application.NewService(repo, platform.Clock{}, publisher, application.ServiceConfig{Retries: 3, Backoff: 100 * time.Millisecond})
	consumer := kafka.NewEventConsumer(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, cfg.Kafka.IngestTopic, service)
	defer func() { _ = consumer.Close() }()

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		grpcdelivery.RecoveryInterceptor(logger),
		grpcdelivery.RequestIDInterceptor(),
		grpcdelivery.LoggingInterceptor(logger),
	))
	pb.RegisterAnalyticsServiceServer(grpcServer, grpcdelivery.NewServer(service))
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthSrv)

	errCh := make(chan error, 2)
	platform.Go(ctx, logger, "kafka-consumer", func() error {
		logger.InfoContext(ctx, "kafka consumer started", slog.String("topic", cfg.Kafka.IngestTopic))
		if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
			return err
		}
		return nil
	}, errCh)
	platform.Go(ctx, logger, "grpc-server", func() error {
		listener, err := net.Listen("tcp", cfg.GRPC.Addr)
		if err != nil {
			return err
		}
		logger.InfoContext(ctx, "grpc server listening", slog.String("addr", cfg.GRPC.Addr))
		if err := grpcServer.Serve(listener); err != nil {
			return err
		}
		return nil
	}, errCh)

	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "shutdown requested")
	case err := <-errCh:
		logger.ErrorContext(ctx, "runtime error", slog.String("error", err.Error()))
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.GRPC.ShutdownTimeout)
	defer cancel()
	shutdownDone := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(shutdownDone)
	}()
	select {
	case <-shutdownDone:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	logger.Info("analytics-service stopped")
}
