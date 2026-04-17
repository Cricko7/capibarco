package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/automaxprocs/maxprocs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"

	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/config"
	grpcdelivery "github.com/petmatch/petmatch/internal/delivery/grpc"
	httpdelivery "github.com/petmatch/petmatch/internal/delivery/http"
	"github.com/petmatch/petmatch/internal/infrastructure/events"
	"github.com/petmatch/petmatch/internal/infrastructure/payment"
	"github.com/petmatch/petmatch/internal/infrastructure/postgres"
	"github.com/petmatch/petmatch/internal/observability"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger := observability.NewLogger(cfg.App)
	if _, err := maxprocs.Set(maxprocs.Logger(func(format string, args ...any) {
		logger.Debug("automaxprocs", slog.String("message", format))
	})); err != nil {
		logger.Warn("set automaxprocs", slog.String("error", err.Error()))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		logger.ErrorContext(ctx, "init postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer store.Close()

	reg := prometheus.NewRegistry()
	metrics := observability.NewMetrics(reg)
	eventPublisher, closeEvents, err := newEventPublisher(cfg.Events, logger)
	if err != nil {
		logger.ErrorContext(ctx, "init event publisher", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer closeEvents()
	service := application.NewService(application.Dependencies{
		Store:    store,
		Payments: payment.NewMockProvider(cfg.Payment.MockBaseURL, cfg.Payment.MockSecretKey),
		Events:   eventPublisher,
		Clock:    systemClock{},
		IDGen:    uuidGenerator{},
		Retry:    application.RetryPolicy{Attempts: 3, Backoff: 100 * time.Millisecond},
		Breaker:  application.NewCircuitBreaker(5, 30*time.Second),
	})

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		grpcdelivery.RecoveryInterceptor(logger),
		grpcdelivery.RequestIDInterceptor(),
		metrics.UnaryServerInterceptor(),
		grpcdelivery.LoggingInterceptor(logger),
	))
	billingv1.RegisterBillingServiceServer(grpcServer, grpcdelivery.NewServer(service))
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)

	httpServer := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           httpdelivery.NewServer(logger, store, reg).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 2)
	go runGRPC(ctx, logger, cfg.GRPC.Addr, grpcServer, errCh)
	go runHTTP(ctx, logger, httpServer, errCh)

	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "shutdown requested")
	case err := <-errCh:
		logger.ErrorContext(ctx, "server failed", slog.String("error", err.Error()))
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.WarnContext(shutdownCtx, "http shutdown failed", slog.String("error", err.Error()))
	}
	grpcServer.GracefulStop()
	logger.Info("billing-service stopped")
}

func runGRPC(ctx context.Context, logger *slog.Logger, addr string, server *grpc.Server, errCh chan<- error) {
	defer recoverPanic(ctx, logger, "grpc-server", errCh)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		errCh <- err
		return
	}
	logger.InfoContext(ctx, "grpc server listening", slog.String("addr", addr))
	if err := server.Serve(listener); err != nil {
		errCh <- err
	}
}

func runHTTP(ctx context.Context, logger *slog.Logger, server *http.Server, errCh chan<- error) {
	defer recoverPanic(ctx, logger, "http-server", errCh)
	logger.InfoContext(ctx, "http server listening", slog.String("addr", server.Addr))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errCh <- err
	}
}

func recoverPanic(ctx context.Context, logger *slog.Logger, name string, errCh chan<- error) {
	if recovered := recover(); recovered != nil {
		logger.ErrorContext(ctx, "goroutine panic recovered", slog.String("name", name), slog.Any("panic", recovered))
		errCh <- os.ErrInvalid
	}
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

type uuidGenerator struct{}

func (uuidGenerator) NewID(prefix string) string {
	return prefix + "_" + uuid.NewString()
}

func newEventPublisher(cfg config.EventsConfig, logger *slog.Logger) (application.EventPublisher, func(), error) {
	if cfg.Publisher == "log" {
		return events.NewLoggingPublisher(logger), func() {}, nil
	}
	writer, err := events.NewKafkaGoWriter(events.KafkaGoConfig{
		Brokers:      cfg.Brokers,
		ClientID:     cfg.ClientID,
		RequiredAcks: cfg.RequiredAcks,
		BatchTimeout: cfg.BatchTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	if err != nil {
		return nil, nil, err
	}
	publisher := events.NewKafkaPublisher(writer, events.KafkaPublisherConfig{
		ProducerName:  cfg.ClientID,
		SchemaVersion: cfg.SchemaVersion,
	})
	return publisher, func() {
		if err := writer.Close(); err != nil {
			logger.Warn("close kafka writer", slog.String("error", err.Error()))
		}
	}, nil
}
