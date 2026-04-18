package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/petmatch/petmatch/internal/adapters/kafka"
	"github.com/petmatch/petmatch/internal/adapters/memory"
	pgadapter "github.com/petmatch/petmatch/internal/adapters/postgres"
	"github.com/petmatch/petmatch/internal/config"
	"github.com/petmatch/petmatch/internal/feed"
	"github.com/petmatch/petmatch/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	deps, workers, cleanup, err := buildRuntime(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("init runtime", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	service := feed.NewService(deps)
	app := server.New(cfg, service, logger, workers...)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("feed-service stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("feed-service stopped")
}

func buildRuntime(ctx context.Context, cfg config.Config, logger *slog.Logger) (feed.Dependencies, []server.Worker, func(), error) {
	deps := feed.Dependencies{}
	workers := make([]server.Worker, 0, 1)
	cleanupFns := make([]func(), 0, 2)

	switch cfg.Storage {
	case config.StoragePostgres:
		db, err := sql.Open("pgx", cfg.Postgres.DSN)
		if err != nil {
			return feed.Dependencies{}, nil, func() {}, err
		}
		db.SetMaxOpenConns(cfg.Postgres.MaxOpenConns)
		db.SetMaxIdleConns(cfg.Postgres.MaxIdleConns)
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return feed.Dependencies{}, nil, func() {}, err
		}
		pgStore := pgadapter.NewStore(db)
		if err := pgStore.EnsureSchema(ctx); err != nil {
			_ = db.Close()
			return feed.Dependencies{}, nil, func() {}, err
		}
		deps.Store = pgStore
		deps.Swipes = pgStore
		deps.Entitlements = pgStore
		cleanupFns = append(cleanupFns, func() { _ = db.Close() })
		if cfg.Kafka.Enabled {
			workers = append(workers, kafka.NewConsumer(kafka.NewKafkaGoReader(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, cfg.Kafka.SubscribeTopics), kafka.NewInboundHandler(pgStore, logger)))
		}
	default:
		memStore := memory.NewStore(nil)
		deps.Store = memStore
		deps.Swipes = memStore
		deps.Entitlements = memStore
		if cfg.Kafka.Enabled {
			workers = append(workers, kafka.NewConsumer(kafka.NewKafkaGoReader(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, cfg.Kafka.SubscribeTopics), kafka.NewInboundHandler(memStore, logger)))
		}
	}

	if cfg.Kafka.Enabled {
		writer := kafka.NewKafkaGoWriter(cfg.Kafka.Brokers)
		deps.Publisher = kafka.NewPublisher(writer)
		cleanupFns = append(cleanupFns, func() {
			if err := writer.Close(); err != nil {
				logger.Warn("close kafka writer", "error", err)
			}
		})
	}

	cleanup := func() {
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			cleanupFns[i]()
		}
	}
	return deps, workers, cleanup, nil
}
