package config

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains service runtime configuration.
type Config struct {
	GRPCAddr                    string
	HTTPAddr                    string
	DatabaseURL                 string
	JWTIssuer                   string
	JWTAudience                 string
	JWTKeyID                    string
	AccessTTL                   time.Duration
	RefreshTTL                  time.Duration
	ResetTTL                    time.Duration
	Ed25519Private              ed25519.PrivateKey
	Ed25519Public               ed25519.PublicKey
	ArgonMemoryKiB              uint32
	ArgonIterations             uint32
	ArgonParallelism            uint8
	KafkaEnabled                bool
	KafkaBrokers                []string
	KafkaClientID               string
	KafkaAllowAutoTopicCreation bool
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	pub, priv, err := loadEd25519Keys()
	if err != nil {
		return Config{}, err
	}
	return Config{
		GRPCAddr:                    getEnv("GRPC_ADDR", ":50051"),
		HTTPAddr:                    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable"),
		JWTIssuer:                   getEnv("JWT_ISSUER", "authsvc"),
		JWTAudience:                 getEnv("JWT_AUDIENCE", "internal-services"),
		JWTKeyID:                    getEnv("JWT_KEY_ID", "local-dev"),
		AccessTTL:                   durationEnv("ACCESS_TTL", 15*time.Minute),
		RefreshTTL:                  durationEnv("REFRESH_TTL", 30*24*time.Hour),
		ResetTTL:                    durationEnv("RESET_TTL", 15*time.Minute),
		Ed25519Private:              priv,
		Ed25519Public:               pub,
		ArgonMemoryKiB:              uint32(intEnv("ARGON2_MEMORY_KIB", 128*1024)),
		ArgonIterations:             uint32(intEnv("ARGON2_ITERATIONS", 3)),
		ArgonParallelism:            uint8(intEnv("ARGON2_PARALLELISM", 4)),
		KafkaEnabled:                boolEnv("KAFKA_ENABLED", false),
		KafkaBrokers:                csvEnv("KAFKA_BROKERS", []string{"localhost:9092"}),
		KafkaClientID:               getEnv("KAFKA_CLIENT_ID", "auth-service"),
		KafkaAllowAutoTopicCreation: boolEnv("KAFKA_ALLOW_AUTO_TOPIC_CREATION", false),
	}, nil
}

func loadEd25519Keys() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privateRaw := strings.TrimSpace(os.Getenv("JWT_ED25519_PRIVATE_KEY_B64"))
	publicRaw := strings.TrimSpace(os.Getenv("JWT_ED25519_PUBLIC_KEY_B64"))
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

func getEnv(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func csvEnv(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}
