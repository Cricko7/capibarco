package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/hackathon/authsvc/internal/adapters/audit"
	"github.com/hackathon/authsvc/internal/adapters/hasher"
	jwtadapter "github.com/hackathon/authsvc/internal/adapters/jwt"
	kafkaadapter "github.com/hackathon/authsvc/internal/adapters/kafka"
	"github.com/hackathon/authsvc/internal/adapters/postgres"
	"github.com/hackathon/authsvc/internal/config"
	deliverygrpc "github.com/hackathon/authsvc/internal/delivery/grpc"
	deliveryhttp "github.com/hackathon/authsvc/internal/delivery/http"
	"github.com/hackathon/authsvc/internal/ports"
	"github.com/hackathon/authsvc/internal/usecase"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	gogrpc "google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	repo, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("open postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			logger.Error("close postgres", slog.Any("error", err))
		}
	}()

	limiter := rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst)

	registry := prometheus.NewRegistry()
	metrics := deliveryhttp.NewMetrics(registry)
	passwordHasher := hasher.NewArgon2id(hasher.Argon2idParams{MemoryKiB: cfg.ArgonMemoryKiB, Iterations: cfg.ArgonIterations, Parallelism: cfg.ArgonParallelism, SaltLength: 16, KeyLength: 32})
	issuer := jwtadapter.NewEd25519Issuer(jwtadapter.Ed25519Config{PrivateKey: cfg.Ed25519Private, PublicKey: cfg.Ed25519Public, Issuer: cfg.JWTIssuer, Audience: cfg.JWTAudience, AccessTTL: cfg.AccessTTL, KeyID: cfg.JWTKeyID})
	tokenService := usecase.NewTokenService(repo, usecase.SystemClock{}, usecase.TokenConfig{RefreshTTL: cfg.RefreshTTL})
	var eventPublisher ports.EventPublisher = kafkaadapter.NewSlogPublisher(logger)
	if cfg.KafkaEnabled {
		kafkaPublisher, err := kafkaadapter.NewPublisher(kafkaadapter.PublisherConfig{Brokers: cfg.KafkaBrokers, ClientID: cfg.KafkaClientID, AllowAutoTopicCreation: cfg.KafkaAllowAutoTopicCreation})
		if err != nil {
			logger.Error("create kafka publisher", slog.Any("error", err))
			os.Exit(1)
		}
		defer func() {
			if err := kafkaPublisher.Close(); err != nil {
				logger.Error("close kafka publisher", slog.Any("error", err))
			}
		}()
		eventPublisher = kafkaPublisher
	}

	mailer := audit.NewResilientMailer(audit.NewLogMailer(logger), cfg.MailRetryAttempts, cfg.MailRetryDelay)
	authService := usecase.NewAuthService(usecase.AuthDependencies{Users: repo, Refresh: tokenService, Resets: repo, RBAC: repo, Hasher: passwordHasher, Issuer: issuer, Audit: repo, Mailer: mailer, Clock: usecase.SystemClock{}, Metrics: metrics, Publisher: eventPublisher, Config: usecase.AuthConfig{ResetTTL: cfg.ResetTTL}})

	httpServer := deliveryhttp.NewServer(cfg.HTTPAddr, repo.DB(), logger, registry, limiter)
	grpcServer := gogrpc.NewServer(
		gogrpc.ForceServerCodec(deliverygrpc.Codec()),
		gogrpc.ChainUnaryInterceptor(
			deliverygrpc.RecoveryUnaryInterceptor(logger),
			deliverygrpc.RequestIDUnaryInterceptor(),
			deliverygrpc.EventMetadataUnaryInterceptor(),
			deliverygrpc.RateLimitUnaryInterceptor(limiter),
			deliverygrpc.LoggingUnaryInterceptor(logger),
		),
	)
	deliverygrpc.RegisterAuthServiceServer(grpcServer, deliverygrpc.NewServer(authService))
	healthServer := healthgrpc.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("auth.v1.AuthService", healthpb.HealthCheckResponse_SERVING)

	grpcListener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("listen grpc", slog.Any("error", err))
		os.Exit(1)
	}

	errCh := make(chan error, 2)
	safeGo(logger, func() error {
		logger.Info("grpc server listening", slog.String("addr", cfg.GRPCAddr))
		return grpcServer.Serve(grpcListener)
	}, errCh)
	safeGo(logger, func() error {
		logger.Info("http server listening", slog.String("addr", cfg.HTTPAddr))
		return httpServer.ListenAndServe()
	}, errCh)

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("server stopped", slog.Any("error", err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	healthServer.SetServingStatus("auth.v1.AuthService", healthpb.HealthCheckResponse_NOT_SERVING)
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown", slog.Any("error", err))
	}
	done := make(chan struct{})
	safeGo(logger, func() error {
		grpcServer.GracefulStop()
		close(done)
		return nil
	}, make(chan error, 1))
	select {
	case <-done:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
	}
	logger.Info("shutdown complete")
}

func safeGo(logger *slog.Logger, fn func() error, errCh chan<- error) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				errCh <- &panicError{value: rec, stack: string(debug.Stack())}
			}
		}()
		if err := fn(); err != nil {
			errCh <- err
		}
	}()
}

type panicError struct {
	value any
	stack string
}

func (e *panicError) Error() string {
	return "panic: recovered"
}
