// Package config loads feed-service runtime configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultGRPCAddr        = ":8081"
	defaultHTTPAddr        = ":8080"
	defaultShutdownTimeout = 10 * time.Second
	defaultConsumerGroup   = "feed-service"
	defaultPostgresMaxOpen = 10
	defaultPostgresMaxIdle = 5
)

// StorageBackend selects the feed projection storage adapter.
type StorageBackend string

const (
	// StorageMemory keeps feed projection state in process memory.
	StorageMemory StorageBackend = "memory"
	// StoragePostgres keeps feed projection state in PostgreSQL.
	StoragePostgres StorageBackend = "postgres"
)

// Config contains feed-service process settings.
type Config struct {
	GRPCAddr        string
	HTTPAddr        string
	ShutdownTimeout time.Duration
	Storage         StorageBackend
	Postgres        PostgresConfig
	Kafka           KafkaConfig
}

// PostgresConfig contains PostgreSQL storage settings.
type PostgresConfig struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

// KafkaConfig contains Kafka producer and consumer settings.
type KafkaConfig struct {
	Enabled         bool
	Brokers         []string
	ConsumerGroup   string
	SubscribeTopics []string
}

// LoadFromEnv reads configuration from process environment variables.
func LoadFromEnv() (Config, error) {
	cfg := Config{
		GRPCAddr:        envString("FEED_GRPC_ADDR", defaultGRPCAddr),
		HTTPAddr:        envString("FEED_HTTP_ADDR", defaultHTTPAddr),
		ShutdownTimeout: envDuration("FEED_SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		Storage:         StorageBackend(envString("FEED_STORAGE", string(StorageMemory))),
		Postgres: PostgresConfig{
			DSN:          envString("FEED_POSTGRES_DSN", ""),
			MaxOpenConns: envInt("FEED_POSTGRES_MAX_OPEN_CONNS", defaultPostgresMaxOpen),
			MaxIdleConns: envInt("FEED_POSTGRES_MAX_IDLE_CONNS", defaultPostgresMaxIdle),
		},
		Kafka: KafkaConfig{
			Enabled:         envBool("FEED_KAFKA_ENABLED", false),
			Brokers:         envCSV("FEED_KAFKA_BROKERS"),
			ConsumerGroup:   envString("FEED_KAFKA_CONSUMER_GROUP", defaultConsumerGroup),
			SubscribeTopics: defaultKafkaSubscribeTopics(),
		},
	}
	if cfg.GRPCAddr == "" {
		return Config{}, fmt.Errorf("FEED_GRPC_ADDR must not be empty")
	}
	if cfg.HTTPAddr == "" {
		return Config{}, fmt.Errorf("FEED_HTTP_ADDR must not be empty")
	}
	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("FEED_SHUTDOWN_TIMEOUT must be positive")
	}
	switch cfg.Storage {
	case StorageMemory:
	case StoragePostgres:
		if cfg.Postgres.DSN == "" {
			return Config{}, fmt.Errorf("FEED_POSTGRES_DSN is required when FEED_STORAGE=postgres")
		}
	default:
		return Config{}, fmt.Errorf("FEED_STORAGE must be one of: %s, %s", StorageMemory, StoragePostgres)
	}
	if cfg.Postgres.MaxOpenConns <= 0 {
		return Config{}, fmt.Errorf("FEED_POSTGRES_MAX_OPEN_CONNS must be positive")
	}
	if cfg.Postgres.MaxIdleConns < 0 {
		return Config{}, fmt.Errorf("FEED_POSTGRES_MAX_IDLE_CONNS must be non-negative")
	}
	if cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) == 0 {
		return Config{}, fmt.Errorf("FEED_KAFKA_BROKERS is required when Kafka is enabled")
	}
	return cfg, nil
}

func envString(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}
	seconds, err := strconv.Atoi(value)
	if err == nil {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envCSV(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func defaultKafkaSubscribeTopics() []string {
	return []string{
		"animal.profile_published",
		"animal.profile_archived",
		"animal.status_changed",
		"matching.swipe_recorded",
		"billing.boost_activated",
		"billing.entitlement_granted",
		"analytics.animal_stats_aggregated",
	}
}
