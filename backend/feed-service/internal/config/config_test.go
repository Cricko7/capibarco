package config

import "testing"

func TestLoadFromEnvIncludesKafkaSettings(t *testing.T) {
	t.Setenv("FEED_KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092")
	t.Setenv("FEED_KAFKA_CONSUMER_GROUP", "feed-service-test")
	t.Setenv("FEED_KAFKA_ENABLED", "true")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	assertStrings(t, cfg.Kafka.Brokers, []string{"kafka-1:9092", "kafka-2:9092"})
	if cfg.Kafka.ConsumerGroup != "feed-service-test" {
		t.Fatalf("consumer group = %q, want feed-service-test", cfg.Kafka.ConsumerGroup)
	}
	if !cfg.Kafka.Enabled {
		t.Fatal("kafka should be enabled")
	}
	assertStrings(t, cfg.Kafka.SubscribeTopics, []string{
		"animal.profile_published",
		"animal.profile_archived",
		"animal.status_changed",
		"matching.swipe_recorded",
		"billing.boost_activated",
		"billing.entitlement_granted",
		"analytics.animal_stats_aggregated",
	})
}

func TestLoadFromEnvIncludesPostgresSettings(t *testing.T) {
	t.Setenv("FEED_STORAGE", "postgres")
	t.Setenv("FEED_POSTGRES_DSN", "postgres://feed:feed@localhost:5432/feed?sslmode=disable")
	t.Setenv("FEED_POSTGRES_MAX_OPEN_CONNS", "12")
	t.Setenv("FEED_POSTGRES_MAX_IDLE_CONNS", "4")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv returned error: %v", err)
	}

	if cfg.Storage != StoragePostgres {
		t.Fatalf("storage = %q, want %q", cfg.Storage, StoragePostgres)
	}
	if cfg.Postgres.DSN != "postgres://feed:feed@localhost:5432/feed?sslmode=disable" {
		t.Fatalf("postgres dsn = %q", cfg.Postgres.DSN)
	}
	if cfg.Postgres.MaxOpenConns != 12 {
		t.Fatalf("max open conns = %d, want 12", cfg.Postgres.MaxOpenConns)
	}
	if cfg.Postgres.MaxIdleConns != 4 {
		t.Fatalf("max idle conns = %d, want 4", cfg.Postgres.MaxIdleConns)
	}
}

func TestLoadFromEnvRequiresPostgresDSN(t *testing.T) {
	t.Setenv("FEED_STORAGE", "postgres")

	_, err := LoadFromEnv()
	if err == nil {
		t.Fatal("LoadFromEnv returned nil error")
	}
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d = %q, want %q; full got %v", i, got[i], want[i], got)
		}
	}
}
