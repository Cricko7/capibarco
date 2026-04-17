package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	appanimal "github.com/petmatch/petmatch/internal/app/animal"
	"github.com/petmatch/petmatch/internal/config"
	grpcserver "github.com/petmatch/petmatch/internal/delivery/grpc"
	httpserver "github.com/petmatch/petmatch/internal/delivery/http"
	"github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/infra/postgres"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "", "path to config yaml")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}
	logger := config.NewLogger(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.OpenPool(ctx, cfg.Postgres.URL, cfg.Postgres.MaxConns)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect postgres", "error", err)
		return 1
	}
	defer pool.Close()

	repository := postgres.NewRepository(pool)
	var publisher appanimal.EventPublisher = kafka.NoopPublisher{}
	var kafkaPublisher *kafka.Publisher
	if cfg.Kafka.Enabled {
		kafkaPublisher = kafka.NewPublisher(cfg.Kafka.Brokers)
		publisher = kafkaPublisher
		defer func() {
			if err := kafkaPublisher.Close(); err != nil {
				logger.Error("failed to close kafka publisher", "error", err)
			}
		}()
	}

	service := appanimal.NewService(repository, publisher, appanimal.UUIDGenerator{}, appanimal.SystemClock{})
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	grpcSrv := grpcserver.NewServer(cfg.GRPC.Addr, service, registry, logger)
	httpSrv := httpserver.NewServer(cfg.HTTP.Addr, repository, registry, logger, cfg.HTTP.AllowedOrigins)

	errCh := make(chan error, 3)
	safe.Go(ctx, logger, "grpc-server", errCh, func(context.Context) error {
		logger.Info("starting grpc server", "addr", cfg.GRPC.Addr)
		return grpcSrv.ListenAndServe()
	})
	safe.Go(ctx, logger, "http-server", errCh, func(context.Context) error {
		logger.Info("starting http server", "addr", cfg.HTTP.Addr)
		return httpSrv.ListenAndServe()
	})

	var consumers *kafka.ConsumerGroup
	if cfg.Kafka.Enabled {
		consumers = kafka.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.GroupID, service, logger)
		safe.Go(ctx, logger, "kafka-consumers", errCh, consumers.Run)
	}

	exitCode := 0
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			logger.Error("runtime error", "error", err)
			exitCode = 1
		}
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout(cfg, logger))
	defer cancel()
	if consumers != nil {
		if err := consumers.Close(); err != nil {
			logger.ErrorContext(shutdownCtx, "failed to close kafka consumers", "error", err)
			exitCode = 1
		}
	}
	grpcStopped := make(chan struct{})
	go func() {
		defer close(grpcStopped)
		grpcSrv.GracefulStop()
	}()
	select {
	case <-grpcStopped:
	case <-shutdownCtx.Done():
		logger.Warn("grpc graceful stop timed out; forcing stop")
		grpcSrv.Stop()
	}
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(shutdownCtx, "failed to shutdown http server", "error", err)
		exitCode = 1
	}
	logger.Info("animal-service stopped")
	return exitCode
}

func shutdownTimeout(cfg config.Config, logger *slog.Logger) time.Duration {
	if cfg.Shutdown.Timeout <= 0 {
		logger.Warn("invalid shutdown timeout; using fallback", "configured", cfg.Shutdown.Timeout)
		return 15 * time.Second
	}
	return cfg.Shutdown.Timeout
}
