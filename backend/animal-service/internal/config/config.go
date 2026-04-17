// Package config loads and validates service configuration.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config contains animal-service runtime settings.
type Config struct {
	Service  ServiceConfig  `mapstructure:"service" validate:"required"`
	GRPC     GRPCConfig     `mapstructure:"grpc" validate:"required"`
	HTTP     HTTPConfig     `mapstructure:"http" validate:"required"`
	Postgres PostgresConfig `mapstructure:"postgres" validate:"required"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Shutdown ShutdownConfig `mapstructure:"shutdown"`
}

// ServiceConfig contains service metadata.
type ServiceConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	Version     string `mapstructure:"version" validate:"required,semver"`
}

// GRPCConfig contains gRPC listener settings.
type GRPCConfig struct {
	Addr string `mapstructure:"addr" validate:"required"`
}

// HTTPConfig contains operational HTTP settings.
type HTTPConfig struct {
	Addr           string   `mapstructure:"addr" validate:"required"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// PostgresConfig contains PostgreSQL settings.
type PostgresConfig struct {
	URL      string `mapstructure:"url" validate:"required"`
	MaxConns int32  `mapstructure:"max_conns" validate:"gte=1,lte=200"`
}

// KafkaConfig contains Kafka settings.
type KafkaConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	Brokers []string `mapstructure:"brokers" validate:"required_if=Enabled true,dive,hostname_port"`
	GroupID string   `mapstructure:"group_id" validate:"required_if=Enabled true"`
}

// LoggingConfig contains structured logging settings.
type LoggingConfig struct {
	Level string `mapstructure:"level" validate:"oneof=debug info warn error"`
}

// ShutdownConfig contains graceful shutdown settings.
type ShutdownConfig struct {
	Timeout time.Duration `mapstructure:"timeout" validate:"required"`
}

// Load reads configuration from file and environment.
func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("ANIMAL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.Kafka.Brokers = v.GetStringSlice("kafka.brokers")
	cfg.HTTP.AllowedOrigins = v.GetStringSlice("http.allowed_origins")
	cfg.Shutdown.Timeout = v.GetDuration("shutdown.timeout")
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service.name", "animal-service")
	v.SetDefault("service.environment", "local")
	v.SetDefault("service.version", "0.1.0")
	v.SetDefault("grpc.addr", "0.0.0.0:9090")
	v.SetDefault("http.addr", "0.0.0.0:8080")
	v.SetDefault("postgres.url", "postgres://animal:animal@localhost:5432/animal?sslmode=disable")
	v.SetDefault("postgres.max_conns", 10)
	v.SetDefault("kafka.enabled", false)
	v.SetDefault("kafka.group_id", "animal-service")
	v.SetDefault("logging.level", "info")
	v.SetDefault("shutdown.timeout", "15s")
}
