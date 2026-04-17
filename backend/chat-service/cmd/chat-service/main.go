package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chatv1 "github.com/petmatch/chat-service/gen/go/petmatch/chat/v1"
	appchat "github.com/petmatch/chat-service/internal/application/chat"
	"github.com/petmatch/chat-service/internal/config"
	grpcdelivery "github.com/petmatch/chat-service/internal/delivery/grpc"
	httpdelivery "github.com/petmatch/chat-service/internal/delivery/http"
	"github.com/petmatch/chat-service/internal/infrastructure/authgrpc"
	"github.com/petmatch/chat-service/internal/infrastructure/breaker"
	"github.com/petmatch/chat-service/internal/infrastructure/postgres"
	"github.com/petmatch/chat-service/internal/infrastructure/realtime"
	"github.com/petmatch/chat-service/internal/observability"
	"github.com/petmatch/chat-service/internal/platform"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Observability.LogLevel).With(
		"service", cfg.App.Name,
		"version", cfg.App.Version,
		"environment", cfg.App.Environment,
	)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.Postgres.DSN, cfg.Postgres.MaxOpenConns, cfg.Postgres.MaxIdleConns, cfg.Postgres.ConnMaxLifetime)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	repo := postgres.NewRepository(pool)
	hub := realtime.NewHub()
	metrics := observability.NewMetrics()
	service := appchat.NewService(repo, repo, repo, hub, platform.SystemClock{}, platform.UUIDGenerator{})
	authExecutor := breaker.NewExecutor(breaker.Config{
		Name:        "auth-service",
		MaxElapsed:  cfg.Resilience.RetryMaxElapsedTime,
		Timeout:     cfg.Resilience.CircuitBreakerTimeout,
		MaxRequests: cfg.Resilience.CircuitBreakerMaxRequests,
	})
	authClient, err := authgrpc.NewClient(ctx, cfg.Auth.Address, authExecutor)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create auth client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := authClient.Close(); closeErr != nil {
			logger.WarnContext(context.Background(), "auth client close failed", "error", closeErr)
		}
	}()

	grpcServer := grpc.NewServer(
		grpcdelivery.UnaryInterceptors(logger, metrics),
		grpcdelivery.StreamInterceptors(logger, metrics),
	)
	chatv1.RegisterChatServiceServer(grpcServer, grpcdelivery.NewChatServer(service, hub, metrics, authClient))
	reflection.Register(grpcServer)

	grpcListener, err := net.Listen("tcp", cfg.GRPC.Address)
	if err != nil {
		logger.ErrorContext(ctx, "failed to listen grpc", "address", cfg.GRPC.Address, "error", err)
		os.Exit(1)
	}

	httpServer := httpdelivery.NewServer(cfg.HTTP, logger, metrics, repo)
	errCh := make(chan error, 2)

	platform.Go(ctx, logger, "grpc-server", func(context.Context) {
		logger.InfoContext(ctx, "starting grpc server", "address", cfg.GRPC.Address)
		if serveErr := grpcServer.Serve(grpcListener); serveErr != nil {
			errCh <- serveErr
		}
	})
	platform.Go(ctx, logger, "http-server", func(context.Context) {
		logger.InfoContext(ctx, "starting http server", "address", cfg.HTTP.Address)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, nethttp.ErrServerClosed) {
			errCh <- serveErr
		}
	})

	select {
	case <-ctx.Done():
		logger.InfoContext(ctx, "shutdown signal received")
	case serveErr := <-errCh:
		logger.ErrorContext(ctx, "server stopped unexpectedly", "error", serveErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.WarnContext(shutdownCtx, "http shutdown failed", "error", err)
	}
	grpcServer.GracefulStop()
	logger.InfoContext(shutdownCtx, "chat-service stopped")
}
