package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/config"
	grpcdelivery "github.com/petmatch/petmatch/internal/delivery/grpc"
	httpdelivery "github.com/petmatch/petmatch/internal/delivery/http"
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

	kafkaPublisher := kafka.NewPublisher(cfg.Kafka.Brokers, cfg.Kafka.TopicRanking, cfg.Kafka.ClientID, cfg.Kafka.WriteTimeout)
	defer func() {
		if err := kafkaPublisher.Close(); err != nil {
			logger.Error("close kafka publisher", slog.String("error", err.Error()))
		}
	}()

	reg := prometheus.NewRegistry()
	metrics := observability.NewMetrics(reg)
	publisher := resilience.NewBreakerPublisher(kafkaPublisher)
	service := application.NewService(repo, platform.Clock{}, publisher, application.ServiceConfig{Retries: 3, Backoff: 100 * time.Millisecond})
	limiter := rate.NewLimiter(rate.Limit(cfg.Rate.RPS), cfg.Rate.Burst)
	handler := httpdelivery.NewServer(logger, service, reg, metrics, limiter)

	httpServer := &http.Server{Addr: cfg.HTTP.Addr, Handler: handler, ReadTimeout: cfg.HTTP.ReadTimeout, WriteTimeout: cfg.HTTP.WriteTimeout, IdleTimeout: cfg.HTTP.IdleTimeout, ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 2)
	platform.Go(ctx, logger, "http-server", func() error {
		logger.InfoContext(ctx, "http server listening", slog.String("addr", cfg.HTTP.Addr))
		err := httpServer.ListenAndServe()
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return err
	}, errCh)

	grpcServer, err := grpcdelivery.ListenAndServe(ctx, logger, cfg.GRPC.Addr, service, errCh)
	if err != nil {
		logger.ErrorContext(ctx, "start grpc server", slog.String("error", err.Error()))
		os.Exit(1)
	}

	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "shutdown requested")
	case err := <-errCh:
		logger.ErrorContext(ctx, "server error", slog.String("error", err.Error()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(shutdownCtx, "shutdown http server", slog.String("error", err.Error()))
	}
	logger.Info("analytics-service stopped")
}
