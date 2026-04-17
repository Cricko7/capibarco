// Package config loads and validates matching-service configuration.
package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config is the service runtime configuration.
type Config struct {
	Service ServiceConfig `mapstructure:"service" validate:"required"`
	HTTP    HTTPConfig    `mapstructure:"http" validate:"required"`
	GRPC    GRPCConfig    `mapstructure:"grpc" validate:"required"`
	DB      DBConfig      `mapstructure:"db" validate:"required"`
	Kafka   KafkaConfig   `mapstructure:"kafka" validate:"required"`
	Chat    ChatConfig    `mapstructure:"chat" validate:"required"`
}

type ServiceConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	LogLevel    string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
}

type HTTPConfig struct {
	Addr         string        `mapstructure:"addr" validate:"required"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"required"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout" validate:"required"`
	CORSOrigins  []string      `mapstructure:"cors_origins"`
}

type GRPCConfig struct {
	Addr string `mapstructure:"addr" validate:"required"`
}

type DBConfig struct {
	DSN             string        `mapstructure:"dsn" validate:"required"`
	MaxConns        int32         `mapstructure:"max_conns" validate:"gte=1,lte=200"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime" validate:"required"`
}

type KafkaConfig struct {
	Brokers        []string      `mapstructure:"brokers" validate:"required,min=1,dive,required"`
	ClientID       string        `mapstructure:"client_id" validate:"required"`
	GroupID        string        `mapstructure:"group_id" validate:"required"`
	OutboxInterval time.Duration `mapstructure:"outbox_interval" validate:"required"`
	OutboxBatch    int           `mapstructure:"outbox_batch" validate:"gte=1,lte=100"`
}

type ChatConfig struct {
	Address string        `mapstructure:"address" validate:"required"`
	Timeout time.Duration `mapstructure:"timeout" validate:"required"`
	Retries int           `mapstructure:"retries" validate:"gte=1,lte=10"`
}

// Load reads configuration from defaults, optional file, and environment variables.
func Load(configPath string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	setDefaults(v)
	v.SetEnvPrefix("MATCHING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if configPath != "" || !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := validator.New().Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service.name", "matching-service")
	v.SetDefault("service.environment", "local")
	v.SetDefault("service.log_level", "info")
	v.SetDefault("http.addr", "0.0.0.0:8080")
	v.SetDefault("http.read_timeout", "5s")
	v.SetDefault("http.write_timeout", "10s")
	v.SetDefault("http.idle_timeout", "60s")
	v.SetDefault("http.cors_origins", []string{"http://localhost:3000"})
	v.SetDefault("grpc.addr", "0.0.0.0:9090")
	v.SetDefault("db.max_conns", 25)
	v.SetDefault("db.max_conn_lifetime", "30m")
	v.SetDefault("kafka.brokers", []string{"localhost:19093"})
	v.SetDefault("kafka.client_id", "matching-service")
	v.SetDefault("kafka.group_id", "matching-service")
	v.SetDefault("kafka.outbox_interval", "1s")
	v.SetDefault("kafka.outbox_batch", 50)
	v.SetDefault("chat.address", "localhost:19092")
	v.SetDefault("chat.timeout", "2s")
	v.SetDefault("chat.retries", 3)
}
