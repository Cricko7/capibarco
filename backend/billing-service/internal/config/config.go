package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app" validate:"required"`
	GRPC     ServerConfig   `mapstructure:"grpc" validate:"required"`
	HTTP     ServerConfig   `mapstructure:"http" validate:"required"`
	Postgres PostgresConfig `mapstructure:"postgres" validate:"required"`
	Payment  PaymentConfig  `mapstructure:"payment" validate:"required"`
	Events   EventsConfig   `mapstructure:"events" validate:"required"`
}

type AppConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Env         string `mapstructure:"env" validate:"required,oneof=local dev staging prod"`
	Version     string `mapstructure:"version" validate:"required,semver"`
	LogLevel    string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
	Development bool   `mapstructure:"development"`
}

type ServerConfig struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
}

type PostgresConfig struct {
	DSN           string        `mapstructure:"dsn" validate:"required"`
	MaxConns      int32         `mapstructure:"max_conns" validate:"gte=1,lte=100"`
	MinConns      int32         `mapstructure:"min_conns" validate:"gte=0,lte=100"`
	HealthTimeout time.Duration `mapstructure:"health_timeout" validate:"required"`
}

type PaymentConfig struct {
	Provider      string `mapstructure:"provider" validate:"required,eq=mock"`
	MockBaseURL   string `mapstructure:"mock_base_url" validate:"required,url"`
	MockSecretKey string `mapstructure:"mock_secret_key" validate:"required,min=16"`
}

type EventsConfig struct {
	Publisher     string        `mapstructure:"publisher" validate:"required,oneof=kafka log"`
	Brokers       []string      `mapstructure:"brokers" validate:"required_if=Publisher kafka,dive,required"`
	ClientID      string        `mapstructure:"client_id" validate:"required"`
	SchemaVersion string        `mapstructure:"schema_version" validate:"required,semver"`
	BatchTimeout  time.Duration `mapstructure:"batch_timeout" validate:"required"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout" validate:"required"`
	RequiredAcks  int           `mapstructure:"required_acks" validate:"oneof=-1 0 1"`
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("BILLING")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigName("config")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		if err := v.ReadInConfig(); err != nil {
			_, notFound := err.(viper.ConfigFileNotFoundError)
			if !notFound {
				return Config{}, fmt.Errorf("read config: %w", err)
			}
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := validator.New().Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "billing-service")
	v.SetDefault("app.env", "local")
	v.SetDefault("app.version", "0.1.0")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.development", true)
	v.SetDefault("grpc.addr", ":9090")
	v.SetDefault("grpc.shutdown_timeout", "10s")
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("postgres.dsn", "postgres://billing:billing@localhost:5432/billing?sslmode=disable")
	v.SetDefault("postgres.max_conns", 10)
	v.SetDefault("postgres.min_conns", 1)
	v.SetDefault("postgres.health_timeout", "2s")
	v.SetDefault("payment.provider", "mock")
	v.SetDefault("payment.mock_base_url", "https://mock-payments.petmatch.local/pay")
	v.SetDefault("payment.mock_secret_key", "local-development-secret")
	v.SetDefault("events.publisher", "kafka")
	v.SetDefault("events.brokers", []string{"localhost:29092"})
	v.SetDefault("events.client_id", "billing-service")
	v.SetDefault("events.schema_version", "1.0.0")
	v.SetDefault("events.batch_timeout", "10ms")
	v.SetDefault("events.write_timeout", "10s")
	v.SetDefault("events.required_acks", -1)
}
