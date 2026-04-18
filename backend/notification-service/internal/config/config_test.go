package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	cfg, err := Load(filepath.Join("..", "..", "configs", "config.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Service.Name != "notification-service" {
		t.Fatalf("unexpected service name: %q", cfg.Service.Name)
	}
	if cfg.HTTP.ShutdownTimeout != 10*time.Second {
		t.Fatalf("unexpected shutdown timeout: %s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.GRPC.Addr != "0.0.0.0:9090" {
		t.Fatalf("unexpected grpc addr: %q", cfg.GRPC.Addr)
	}
}
