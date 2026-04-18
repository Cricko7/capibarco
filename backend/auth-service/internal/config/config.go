package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config contains service runtime configuration.
type Config struct {
	GRPCAddr                    string        `mapstructure:"grpc_addr" validate:"required"`
	HTTPAddr                    string        `mapstructure:"http_addr" validate:"required"`
	DatabaseURL                 string        `mapstructure:"database_url" validate:"required,url"`
	JWTIssuer                   string        `mapstructure:"jwt_issuer" validate:"required"`
	JWTAudience                 string        `mapstructure:"jwt_audience" validate:"required"`
	JWTKeyID                    string        `mapstructure:"jwt_key_id" validate:"required"`
	AccessTTL                   time.Duration `mapstructure:"access_ttl" validate:"required,gt=0"`
	RefreshTTL                  time.Duration `mapstructure:"refresh_ttl" validate:"required,gt=0"`
	ResetTTL                    time.Duration `mapstructure:"reset_ttl" validate:"required,gt=0"`
	ArgonMemoryKiB              uint32        `mapstructure:"argon2_memory_kib" validate:"required,gt=0"`
	ArgonIterations             uint32        `mapstructure:"argon2_iterations" validate:"required,gt=0"`
	ArgonParallelism            uint8         `mapstructure:"argon2_parallelism" validate:"required,gt=0"`
	KafkaEnabled                bool          `mapstructure:"kafka_enabled"`
	KafkaBrokers                []string      `mapstructure:"kafka_brokers" validate:"required,min=1,dive,hostname_port"`
	KafkaClientID               string        `mapstructure:"kafka_client_id" validate:"required"`
	KafkaAllowAutoTopicCreation bool          `mapstructure:"kafka_allow_auto_topic_creation"`
	RateLimitRPS                float64       `mapstructure:"rate_limit_rps" validate:"required,gt=0"`
	RateLimitBurst              int           `mapstructure:"rate_limit_burst" validate:"required,gt=0"`
	MailRetryAttempts           uint          `mapstructure:"mail_retry_attempts" validate:"required,gt=0"`
	MailRetryDelay              time.Duration `mapstructure:"mail_retry_delay" validate:"required,gt=0"`

	Ed25519Private ed25519.PrivateKey
	Ed25519Public  ed25519.PublicKey
}

// Load reads configuration from file + environment variables.
func Load() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/auth-service")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	pub, priv, err := loadEd25519Keys(v)
	if err != nil {
		return Config{}, err
	}
	cfg.Ed25519Public = pub
	cfg.Ed25519Private = priv

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("grpc_addr", ":50051")
	v.SetDefault("http_addr", ":8080")
	v.SetDefault("database_url", "postgres://auth:auth@localhost:5432/auth?sslmode=disable")
	v.SetDefault("jwt_issuer", "authsvc")
	v.SetDefault("jwt_audience", "internal-services")
	v.SetDefault("jwt_key_id", "local-dev")
	v.SetDefault("access_ttl", "15m")
	v.SetDefault("refresh_ttl", "720h")
	v.SetDefault("reset_ttl", "15m")
	v.SetDefault("argon2_memory_kib", 128*1024)
	v.SetDefault("argon2_iterations", 3)
	v.SetDefault("argon2_parallelism", 4)
	v.SetDefault("kafka_enabled", false)
	v.SetDefault("kafka_brokers", []string{"localhost:9092"})
	v.SetDefault("kafka_client_id", "auth-service")
	v.SetDefault("kafka_allow_auto_topic_creation", false)
	v.SetDefault("rate_limit_rps", 20)
	v.SetDefault("rate_limit_burst", 60)
	v.SetDefault("mail_retry_attempts", 3)
	v.SetDefault("mail_retry_delay", "200ms")
}

func loadEd25519Keys(v *viper.Viper) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privateRaw := strings.TrimSpace(v.GetString("jwt_ed25519_private_key_b64"))
	publicRaw := strings.TrimSpace(v.GetString("jwt_ed25519_public_key_b64"))
	if privateRaw == "" || publicRaw == "" {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("generate development ed25519 key: %w", err)
		}
		return pub, priv, nil
	}
	privBytes, err := base64.StdEncoding.DecodeString(privateRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("decode JWT_ED25519_PRIVATE_KEY_B64: %w", err)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(publicRaw)
	if err != nil {
		return nil, nil, fmt.Errorf("decode JWT_ED25519_PUBLIC_KEY_B64: %w", err)
	}
	if l := len(privBytes); l != ed25519.PrivateKeySize {
		return nil, nil, fmt.Errorf("invalid private key length: %d", l)
	}
	if l := len(pubBytes); l != ed25519.PublicKeySize {
		return nil, nil, fmt.Errorf("invalid public key length: %d", l)
	}
	return ed25519.PublicKey(pubBytes), ed25519.PrivateKey(privBytes), nil
}
