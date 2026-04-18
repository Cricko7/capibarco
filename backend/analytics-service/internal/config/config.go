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
	GRPC     GRPCConfig     `mapstructure:"grpc" validate:"required"`
	Postgres PostgresConfig `mapstructure:"postgres" validate:"required"`
	Kafka    KafkaConfig    `mapstructure:"kafka" validate:"required"`
}

type AppConfig struct {
	Name    string `mapstructure:"name" validate:"required"`
	Env     string `mapstructure:"env" validate:"required,oneof=local dev stage prod"`
	Version string `mapstructure:"version" validate:"required"`
}

type GRPCConfig struct {
	Addr            string        `mapstructure:"addr" validate:"required"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"required"`
}

type PostgresConfig struct {
	DSN string `mapstructure:"dsn" validate:"required"`
}

type KafkaConfig struct {
	Brokers             []string `mapstructure:"brokers" validate:"required,min=1,dive,required"`
	IngestTopic         string   `mapstructure:"ingest_topic" validate:"required"`
	RankingFeedbackTopic string  `mapstructure:"ranking_feedback_topic" validate:"required"`
	ConsumerGroup       string   `mapstructure:"consumer_group" validate:"required"`
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
	v.SetDefault("app.version", "0.2.0")
	v.SetDefault("grpc.addr", ":19096")
	v.SetDefault("grpc.shutdown_timeout", "10s")
	v.SetDefault("postgres.dsn", "postgres://analytics:analytics@localhost:5432/analytics?sslmode=disable")
	v.SetDefault("kafka.brokers", []string{"localhost:19093"})
	v.SetDefault("kafka.ingest_topic", "analytics.events.raw")
	v.SetDefault("kafka.ranking_feedback_topic", "analytics.ranking.feedback")
	v.SetDefault("kafka.consumer_group", "analytics-service")
}
