package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config is the runtime configuration for chat-service.
type Config struct {
	App           AppConfig           `mapstructure:"app" validate:"required"`
	GRPC          GRPCConfig          `mapstructure:"grpc" validate:"required"`
	HTTP          HTTPConfig          `mapstructure:"http" validate:"required"`
	Postgres      PostgresConfig      `mapstructure:"postgres" validate:"required"`
	Auth          AuthConfig          `mapstructure:"auth" validate:"required"`
	Resilience    ResilienceConfig    `mapstructure:"resilience" validate:"required"`
	Observability ObservabilityConfig `mapstructure:"observability" validate:"required"`
}

type AppConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	Version     string `mapstructure:"version" validate:"required,semver"`
}

type GRPCConfig struct {
	Address string `mapstructure:"address" validate:"required,hostname_port"`
}

type HTTPConfig struct {
	Address      string        `mapstructure:"address" validate:"required,hostname_port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"required"`
}

type PostgresConfig struct {
	DSN             string        `mapstructure:"dsn" validate:"required,url"`
	MaxOpenConns    int32         `mapstructure:"max_open_conns" validate:"min=1"`
	MaxIdleConns    int32         `mapstructure:"max_idle_conns" validate:"min=1"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" validate:"required"`
}

type AuthConfig struct {
	Address             string `mapstructure:"address" validate:"required,hostname_port"`
	ValidateTokenMethod string `mapstructure:"validate_token_method" validate:"required"`
}

type ResilienceConfig struct {
	RetryMaxElapsedTime       time.Duration `mapstructure:"retry_max_elapsed_time" validate:"required"`
	CircuitBreakerTimeout     time.Duration `mapstructure:"circuit_breaker_timeout" validate:"required"`
	CircuitBreakerMaxRequests uint32        `mapstructure:"circuit_breaker_max_requests" validate:"min=1"`
}

type ObservabilityConfig struct {
	LogLevel string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
}

// Load reads configuration from file and CHAT_* environment variables.
func Load(configPath string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")
	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	setDefaults(v)
	v.SetEnvPrefix("CHAT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
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
	v.SetDefault("app.name", "chat-service")
	v.SetDefault("app.environment", "local")
	v.SetDefault("app.version", "0.1.0")
	v.SetDefault("grpc.address", "0.0.0.0:9090")
	v.SetDefault("http.address", "0.0.0.0:8080")
	v.SetDefault("http.read_timeout", "5s")
	v.SetDefault("http.write_timeout", "10s")
	v.SetDefault("postgres.max_open_conns", 20)
	v.SetDefault("postgres.max_idle_conns", 5)
	v.SetDefault("postgres.conn_max_lifetime", "30m")
	v.SetDefault("auth.address", "localhost:50051")
	v.SetDefault("auth.validate_token_method", "petmatch.auth.v1.AuthService/ValidateToken")
	v.SetDefault("resilience.retry_max_elapsed_time", "2s")
	v.SetDefault("resilience.circuit_breaker_timeout", "10s")
	v.SetDefault("resilience.circuit_breaker_max_requests", 3)
	v.SetDefault("observability.log_level", "info")
}
