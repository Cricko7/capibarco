package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	app "github.com/petmatch/petmatch/internal/app/user"
	"github.com/petmatch/petmatch/internal/config"
	deliverygrpc "github.com/petmatch/petmatch/internal/delivery/grpc"
	deliveryhttp "github.com/petmatch/petmatch/internal/delivery/http"
	"github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/infra/postgres"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	"github.com/petmatch/petmatch/internal/pkg/resilience"
)

func main() {
	if err := run(); err != nil {
		slog.Error("user-service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	logger := config.NewLogger(cfg.Service.LogLevel)
	slog.SetDefault(logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.OpenPool(ctx, cfg.DB.DSN, cfg.DB.MaxConns, cfg.DB.MaxConnLifetime)
	if err != nil {
		return err
	}
	defer pool.Close()
	repo := postgres.NewRepository(pool)

	var publisher app.EventPublisher = kafka.NoopPublisher{}
	var kafkaPublisher *kafka.Publisher
	if cfg.Kafka.Enabled {
		kafkaPublisher = kafka.NewPublisher(cfg.Kafka.Brokers, cfg.Kafka.ClientID, cfg.Kafka.RetryCount, cfg.Kafka.RetryBackoff, cfg.Kafka.BreakerFailThreshold, logger)
		publisher = kafkaPublisher
		defer func() {
			if err := kafkaPublisher.Close(); err != nil {
				logger.Warn("close kafka", "error", err)
			}
		}()
	}
	service := app.NewService(repo, publisher, cfg.Kafka.TopicPrefix, time.Now)
	m := metrics.New(prometheus.DefaultRegisterer)

	grpcLis, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(deliverygrpc.UnaryInterceptor(logger, m, resilience.NewRateLimiter(cfg.GRPC.RateLimitPerSec, cfg.GRPC.RateLimitBurst), cfg.GRPC.RequestTimeout)))
	userv1.RegisterUserServiceServer(grpcServer, deliverygrpc.NewServer(service))

	httpServer := deliveryhttp.NewServer(cfg.HTTP.Addr, cfg.HTTP.ReadTimeout, cfg.HTTP.WriteTimeout, cfg.HTTP.IdleTimeout, cfg.HTTP.CORSOrigins, service, m, logger)

	errCh := make(chan error, 2)
	safe.Go(ctx, logger, "grpc-server", func(context.Context) {
		logger.Info("starting grpc server", "addr", cfg.GRPC.Addr)
		if err := grpcServer.Serve(grpcLis); err != nil {
			errCh <- fmt.Errorf("serve grpc: %w", err)
		}
	})
	safe.Go(ctx, logger, "http-server", func(context.Context) {
		logger.Info("starting http server", "addr", cfg.HTTP.Addr)
		if err := httpServer.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("serve http: %w", err)
		}
	})
	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			stop()
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown http", "error", err)
	}
	stopped := make(chan struct{})
	go func() { grpcServer.GracefulStop(); close(stopped) }()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	return nil
}
