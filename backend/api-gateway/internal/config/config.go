// Package config loads and validates api-gateway configuration.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config contains api-gateway runtime settings.
type Config struct {
	Service ServiceConfig `mapstructure:"service" validate:"required"`
	HTTP    HTTPConfig    `mapstructure:"http" validate:"required"`
	GRPC    GRPCConfig    `mapstructure:"grpc" validate:"required"`
	Redis   RedisConfig   `mapstructure:"redis" validate:"required"`
	Kafka   KafkaConfig   `mapstructure:"kafka"`
	S3      S3Config      `mapstructure:"s3"`
	Rate    RateConfig    `mapstructure:"rate" validate:"required"`
	Auth    AuthConfig    `mapstructure:"auth" validate:"required"`
}

type ServiceConfig struct {
	Name        string `mapstructure:"name" validate:"required"`
	Environment string `mapstructure:"environment" validate:"required"`
	Version     string `mapstructure:"version" validate:"required"`
	LogLevel    string `mapstructure:"log_level" validate:"required,oneof=debug info warn error"`
}

type HTTPConfig struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"required"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout" validate:"required"`
	MaxBodyBytes    int64         `mapstructure:"max_body_bytes" validate:"gte=1024"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
}

type GRPCConfig struct {
	AuthAddr            string        `mapstructure:"auth_addr" validate:"required"`
	AnimalAddr          string        `mapstructure:"animal_addr" validate:"required"`
	FeedAddr            string        `mapstructure:"feed_addr" validate:"required"`
	MatchingAddr        string        `mapstructure:"matching_addr" validate:"required"`
	ChatAddr            string        `mapstructure:"chat_addr" validate:"required"`
	BillingAddr         string        `mapstructure:"billing_addr" validate:"required"`
	UserAddr            string        `mapstructure:"user_addr" validate:"required"`
	AnalyticsAddr       string        `mapstructure:"analytics_addr" validate:"required"`
	NotificationEnabled bool          `mapstructure:"notification_enabled"`
	NotificationAddr    string        `mapstructure:"notification_addr"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout" validate:"required"`
	RetryCount          int           `mapstructure:"retry_count" validate:"gte=0,lte=5"`
	RetryBackoff        time.Duration `mapstructure:"retry_backoff" validate:"required"`
	BreakerName         string        `mapstructure:"breaker_name" validate:"required"`
}

type RedisConfig struct {
	Addr     string        `mapstructure:"addr" validate:"required"`
	Password string        `mapstructure:"password"`
	DB       int           `mapstructure:"db" validate:"gte=0"`
	Timeout  time.Duration `mapstructure:"timeout" validate:"required"`
}

type KafkaConfig struct {
	Enabled  bool     `mapstructure:"enabled"`
	Brokers  []string `mapstructure:"brokers" validate:"required_if=Enabled true,dive,required"`
	ClientID string   `mapstructure:"client_id" validate:"required_if=Enabled true"`
}

type S3Config struct {
	Enabled   bool   `mapstructure:"enabled"`
	Endpoint  string `mapstructure:"endpoint" validate:"required_if=Enabled true"`
	AccessKey string `mapstructure:"access_key" validate:"required_if=Enabled true"`
	SecretKey string `mapstructure:"secret_key" validate:"required_if=Enabled true"`
	Bucket    string `mapstructure:"bucket" validate:"required_if=Enabled true"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	PublicURL string `mapstructure:"public_url"`
}

type RateConfig struct {
	Window         time.Duration `mapstructure:"window" validate:"required"`
	IPLimit        int64         `mapstructure:"ip_limit" validate:"gte=1"`
	ActorLimit     int64         `mapstructure:"actor_limit" validate:"gte=1"`
	RoleLimit      int64         `mapstructure:"role_limit" validate:"gte=1"`
	GuestLimit     int64         `mapstructure:"guest_limit" validate:"gte=1"`
	AdminRoleLimit int64         `mapstructure:"admin_role_limit" validate:"gte=1"`
}

type AuthConfig struct {
	TenantID     string        `mapstructure:"tenant_id" validate:"required"`
	GuestSecret  string        `mapstructure:"guest_secret" validate:"required,min=16"`
	GuestTTL     time.Duration `mapstructure:"guest_ttl" validate:"required"`
	FeedPrefetch int32         `mapstructure:"feed_prefetch" validate:"gte=1,lte=50"`
	MaxPageSize  int32         `mapstructure:"max_page_size" validate:"gte=1,lte=100"`
}

// Load reads config from YAML and environment variables.
func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("API_GATEWAY")
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
		if path != "" || !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	cfg.HTTP.CORSOrigins = v.GetStringSlice("http.cors_origins")
	cfg.Kafka.Brokers = v.GetStringSlice("kafka.brokers")
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("service.name", "api-gateway")
	v.SetDefault("service.environment", "local")
	v.SetDefault("service.version", "0.1.0")
	v.SetDefault("service.log_level", "info")
	v.SetDefault("http.addr", "0.0.0.0:8080")
	v.SetDefault("http.read_timeout", "10s")
	v.SetDefault("http.write_timeout", "30s")
	v.SetDefault("http.idle_timeout", "60s")
	v.SetDefault("http.shutdown_timeout", "15s")
	v.SetDefault("http.max_body_bytes", 10485760)
	v.SetDefault("http.cors_origins", []string{"http://localhost:3000"})
	v.SetDefault("grpc.auth_addr", "localhost:15051")
	v.SetDefault("grpc.animal_addr", "localhost:19090")
	v.SetDefault("grpc.feed_addr", "localhost:18085")
	v.SetDefault("grpc.matching_addr", "localhost:19094")
	v.SetDefault("grpc.chat_addr", "localhost:19092")
	v.SetDefault("grpc.billing_addr", "localhost:19091")
	v.SetDefault("grpc.user_addr", "localhost:19095")
	v.SetDefault("grpc.analytics_addr", "localhost:19096")
	v.SetDefault("grpc.notification_enabled", false)
	v.SetDefault("grpc.notification_addr", "localhost:19097")
	v.SetDefault("grpc.request_timeout", "3s")
	v.SetDefault("grpc.retry_count", 2)
	v.SetDefault("grpc.retry_backoff", "100ms")
	v.SetDefault("grpc.breaker_name", "api-gateway-downstream")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.timeout", "500ms")
	v.SetDefault("kafka.enabled", false)
	v.SetDefault("kafka.brokers", []string{"localhost:19093"})
	v.SetDefault("kafka.client_id", "api-gateway")
	v.SetDefault("s3.enabled", false)
	v.SetDefault("s3.endpoint", "localhost:9000")
	v.SetDefault("s3.bucket", "petmatch-photos")
	v.SetDefault("rate.window", "1m")
	v.SetDefault("rate.ip_limit", 600)
	v.SetDefault("rate.actor_limit", 300)
	v.SetDefault("rate.role_limit", 1000)
	v.SetDefault("rate.guest_limit", 120)
	v.SetDefault("rate.admin_role_limit", 5000)
	v.SetDefault("auth.tenant_id", "default")
	v.SetDefault("auth.guest_secret", "local-development-guest-secret")
	v.SetDefault("auth.guest_ttl", "24h")
	v.SetDefault("auth.feed_prefetch", 10)
	v.SetDefault("auth.max_page_size", 50)
}

// NewLogger creates a JSON slog logger from configuration.
func NewLogger(cfg Config) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.Service.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
