package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	Service ServiceConfig `mapstructure:"service" validate:"required"`
	HTTP    HTTPConfig    `mapstructure:"http" validate:"required"`
	GRPC    GRPCConfig    `mapstructure:"grpc" validate:"required"`
	DB      DBConfig      `mapstructure:"db" validate:"required"`
	Kafka   KafkaConfig   `mapstructure:"kafka" validate:"required"`
}

type ServiceConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	LogLevel    string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
	Version     string `mapstructure:"version" validate:"required"`
}

type HTTPConfig struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"required"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
}

type GRPCConfig struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	RateLimitPerSec float64       `mapstructure:"rate_limit_per_sec" validate:"gte=1"`
	RateLimitBurst  int           `mapstructure:"rate_limit_burst" validate:"gte=1"`
	RequestTimeout  time.Duration `mapstructure:"request_timeout" validate:"required"`
}

type DBConfig struct {
	DSN             string        `mapstructure:"dsn" validate:"required"`
	MaxConns        int32         `mapstructure:"max_conns" validate:"gte=1,lte=200"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime" validate:"required"`
}

type KafkaConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	Brokers              []string      `mapstructure:"brokers" validate:"min=1,dive,required"`
	ClientID             string        `mapstructure:"client_id" validate:"required"`
	GroupID              string        `mapstructure:"group_id" validate:"required"`
	TopicPrefix          string        `mapstructure:"topic_prefix"`
	RetryCount           int           `mapstructure:"retry_count" validate:"gte=1,lte=10"`
	RetryBackoff         time.Duration `mapstructure:"retry_backoff" validate:"required"`
	BreakerFailThreshold uint32        `mapstructure:"breaker_fail_threshold" validate:"gte=1"`
}

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
	v.SetEnvPrefix("NOTIFICATION")
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
	if !cfg.Kafka.Enabled {
		cfg.Kafka.Brokers = []string{"localhost:19093"}
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service.name", "notification-service")
	v.SetDefault("service.environment", "local")
	v.SetDefault("service.log_level", "info")
	v.SetDefault("service.version", "0.1.0")
	v.SetDefault("http.addr", "0.0.0.0:8080")
	v.SetDefault("http.read_timeout", "5s")
	v.SetDefault("http.write_timeout", "10s")
	v.SetDefault("http.idle_timeout", "60s")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("http.cors_origins", []string{"http://localhost:3000"})
	v.SetDefault("grpc.addr", "0.0.0.0:9090")
	v.SetDefault("grpc.rate_limit_per_sec", 100.0)
	v.SetDefault("grpc.rate_limit_burst", 200)
	v.SetDefault("grpc.request_timeout", "5s")
	v.SetDefault("db.max_conns", 25)
	v.SetDefault("db.max_conn_lifetime", "30m")
	v.SetDefault("kafka.enabled", true)
	v.SetDefault("kafka.brokers", []string{"localhost:19093"})
	v.SetDefault("kafka.client_id", "notification-service")
	v.SetDefault("kafka.group_id", "notification-service")
	v.SetDefault("kafka.topic_prefix", "notification")
	v.SetDefault("kafka.retry_count", 3)
	v.SetDefault("kafka.retry_backoff", "200ms")
	v.SetDefault("kafka.breaker_fail_threshold", 5)
}
