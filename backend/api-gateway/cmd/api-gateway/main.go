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

	"github.com/petmatch/petmatch/internal/app/gateway"
	"github.com/petmatch/petmatch/internal/config"
	httpserver "github.com/petmatch/petmatch/internal/delivery/http"
	"github.com/petmatch/petmatch/internal/domain"
	"github.com/petmatch/petmatch/internal/infra/grpcclient"
	kafkaevents "github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/infra/redislimiter"
	"github.com/petmatch/petmatch/internal/infra/storage"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/resilience"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
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

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	gatewayMetrics := metrics.New(registry)

	conns, err := grpcclient.DialAll(ctx, cfg.GRPC, gatewayMetrics)
	if err != nil {
		logger.ErrorContext(ctx, "dial downstream services", "error", err)
		return 1
	}
	defer closeWithLog(logger, "grpc connections", conns.Close)

	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  cfg.Redis.Timeout,
		ReadTimeout:  cfg.Redis.Timeout,
		WriteTimeout: cfg.Redis.Timeout,
	})
	defer closeWithLog(logger, "redis", redisClient.Close)
	limiter := redislimiter.New(redisClient, "api-gateway")

	var publisher kafkaevents.Publisher = kafkaevents.NoopPublisher{}
	if cfg.Kafka.Enabled {
		publisher = kafkaevents.New(cfg.Kafka)
	}
	defer closeWithLog(logger, "kafka publisher", publisher.Close)

	var uploader storage.Uploader = storage.NoopUploader{}
	if cfg.S3.Enabled {
		s3Uploader, err := storage.NewS3Uploader(cfg.S3)
		if err != nil {
			logger.ErrorContext(ctx, "create object storage uploader", "error", err)
			return 1
		}
		uploader = s3Uploader
	}

	res := func(name string) *resilience.Client {
		return resilience.New(name, cfg.GRPC.RetryCount, cfg.GRPC.RetryBackoff)
	}
	authClient := grpcclient.NewAuthClient(conns.Auth, cfg.GRPC.RequestTimeout, res("auth-service"))
	animalClient := grpcclient.NewAnimalClient(conns.Animal, cfg.GRPC.RequestTimeout, res("animal-service"))
	feedClient := grpcclient.NewFeedClient(conns.Feed, cfg.GRPC.RequestTimeout, res("feed-service"))
	matchingClient := grpcclient.NewMatchingClient(conns.Matching, cfg.GRPC.RequestTimeout, res("matching-service"))
	chatClient := grpcclient.NewChatClient(conns.Chat, cfg.GRPC.RequestTimeout, res("chat-service"))
	billingClient := grpcclient.NewBillingClient(conns.Billing, cfg.GRPC.RequestTimeout, res("billing-service"))
	userClient := grpcclient.NewUserClient(conns.User, cfg.GRPC.RequestTimeout, res("user-service"))
	analyticsClient := grpcclient.NewAnalyticsClient(conns.Analytics, cfg.GRPC.RequestTimeout, res("analytics-service"))
	var notificationClient *grpcclient.NotificationClient
	var notificationDependency gateway.NotificationClient
	if conns.Notification != nil {
		notificationClient = grpcclient.NewNotificationClient(conns.Notification, cfg.GRPC.RequestTimeout, res("notification-service"))
		notificationDependency = notificationClient
	}

	app := gateway.NewService(gateway.Dependencies{
		Auth:          authClient,
		Feed:          feedClient,
		Animal:        animalClient,
		Matching:      matchingClient,
		Chat:          chatClient,
		Billing:       billingClient,
		User:          userClient,
		Analytics:     analyticsClient,
		Notification:  notificationDependency,
		GuestSessions: domain.NewGuestSessionCodec([]byte(cfg.Auth.GuestSecret), cfg.Auth.GuestTTL),
		Defaults: gateway.Defaults{
			TenantID:         cfg.Auth.TenantID,
			FeedPrefetchSize: cfg.Auth.FeedPrefetch,
			MaxPageSize:      cfg.Auth.MaxPageSize,
		},
	})

	var httpSrv *httpserver.Server
	if notificationClient != nil {
		httpSrv = httpserver.New(cfg, app, chatClient, notificationClient, limiter, publisher, uploader, registry, gatewayMetrics, logger)
	} else {
		httpSrv = httpserver.New(cfg, app, chatClient, nil, limiter, publisher, uploader, registry, gatewayMetrics, logger)
	}
	errCh := make(chan error, 1)
	safe.Go(ctx, logger, "http-server", errCh, func(context.Context) error {
		logger.Info("starting api-gateway", "addr", cfg.HTTP.Addr)
		return httpSrv.ListenAndServe()
	})

	exitCode := 0
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			logger.Error("api-gateway runtime error", "error", err)
			exitCode = 1
		}
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.ErrorContext(shutdownCtx, "shutdown http server", "error", err)
		exitCode = 1
	}
	logger.Info("api-gateway stopped")
	return exitCode
}

func closeWithLog(logger *slog.Logger, name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		logger.Error("close resource", "name", name, "error", err)
	}
}
