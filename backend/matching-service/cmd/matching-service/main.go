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

	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	app "github.com/petmatch/petmatch/internal/app/matching"
	"github.com/petmatch/petmatch/internal/config"
	deliverygrpc "github.com/petmatch/petmatch/internal/delivery/grpc"
	deliveryhttp "github.com/petmatch/petmatch/internal/delivery/http"
	"github.com/petmatch/petmatch/internal/infra/chatgrpc"
	"github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/infra/postgres"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("matching-service stopped", "error", err)
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

	clock := app.SystemClock{}
	store := postgres.NewStore(pool, clock)
	chatClient, err := chatgrpc.Dial(ctx, cfg.Chat.Address, cfg.Chat.Timeout, cfg.Chat.Retries)
	if err != nil {
		return err
	}
	defer func() {
		if err := chatClient.Close(); err != nil {
			logger.Warn("close chat client", "error", err)
		}
	}()

	service := app.NewService(store, chatClient, clock)
	serviceMetrics := metrics.New(prometheus.DefaultRegisterer)

	grpcListener, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(deliverygrpc.UnaryInterceptor(logger, serviceMetrics)))
	matchingv1.RegisterMatchingServiceServer(grpcServer, deliverygrpc.NewServer(service))

	httpServer := deliveryhttp.NewServer(
		cfg.HTTP.Addr,
		cfg.HTTP.ReadTimeout,
		cfg.HTTP.WriteTimeout,
		cfg.HTTP.IdleTimeout,
		cfg.HTTP.CORSOrigins,
		store,
		serviceMetrics,
		logger,
	)

	outboxPublisher := kafka.NewOutboxPublisher(store, cfg.Kafka.Brokers, logger, cfg.Kafka.ClientID, cfg.Kafka.OutboxInterval, cfg.Kafka.OutboxBatch)
	consumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, service, logger)

	errCh := make(chan error, 4)
	safe.Go(ctx, logger, "grpc-server", func(ctx context.Context) {
		logger.Info("starting grpc server", "addr", cfg.GRPC.Addr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			errCh <- fmt.Errorf("serve grpc: %w", err)
		}
	})
	safe.Go(ctx, logger, "http-server", func(ctx context.Context) {
		logger.Info("starting http server", "addr", cfg.HTTP.Addr)
		if err := httpServer.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("serve http: %w", err)
		}
	})
	safe.Go(ctx, logger, "kafka-outbox", func(ctx context.Context) {
		if err := outboxPublisher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("run kafka outbox: %w", err)
		}
	})
	safe.Go(ctx, logger, "kafka-consumer", func(ctx context.Context) {
		if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("run kafka consumer: %w", err)
		}
	})

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err := <-errCh:
		stop()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown http server", "error", err)
	}
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	return nil
}
