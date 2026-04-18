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
	HTTP     HTTPConfig     `mapstructure:"http" validate:"required"`
	GRPC     GRPCConfig     `mapstructure:"grpc" validate:"required"`
	Postgres PostgresConfig `mapstructure:"postgres" validate:"required"`
	Kafka    KafkaConfig    `mapstructure:"kafka" validate:"required"`
	Rate     RateConfig     `mapstructure:"rate" validate:"required"`
}

type AppConfig struct {
	Name    string `mapstructure:"name" validate:"required"`
	Env     string `mapstructure:"env" validate:"required,oneof=local dev stage prod"`
	Version string `mapstructure:"version" validate:"required"`
}

type HTTPConfig struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" validate:"required"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout" validate:"required"`
}

type GRPCConfig struct {
	Addr string `mapstructure:"addr" validate:"required"`
}

type PostgresConfig struct {
	DSN string `mapstructure:"dsn" validate:"required"`
}

type KafkaConfig struct {
	Brokers      []string      `mapstructure:"brokers" validate:"required,min=1,dive,required"`
	TopicRanking string        `mapstructure:"topic_ranking" validate:"required"`
	ClientID     string        `mapstructure:"client_id" validate:"required"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" validate:"required"`
}

type RateConfig struct {
	RPS   float64 `mapstructure:"rps" validate:"required,gt=0"`
	Burst int     `mapstructure:"burst" validate:"required,gt=0"`
}

func Load(path string) (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}
	v.SetEnvPrefix("ANALYTICS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "analytics-service")
	v.SetDefault("app.env", "local")
	v.SetDefault("app.version", "0.1.0")
	v.SetDefault("http.addr", ":18088")
	v.SetDefault("grpc.addr", ":9090")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("http.read_timeout", "10s")
	v.SetDefault("http.write_timeout", "15s")
	v.SetDefault("http.idle_timeout", "30s")
	v.SetDefault("postgres.dsn", "postgres://analytics:analytics@localhost:5432/analytics?sslmode=disable")
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topic_ranking", "analytics.ranking_feedback.v1")
	v.SetDefault("kafka.client_id", "analytics-service")
	v.SetDefault("kafka.write_timeout", "5s")
	v.SetDefault("rate.rps", 100.0)
	v.SetDefault("rate.burst", 200)
}
