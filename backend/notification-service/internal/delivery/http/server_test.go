package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestServerHealthEndpoints(t *testing.T) {
	t.Parallel()

	server := NewServer(
		"127.0.0.1:0",
		time.Second,
		time.Second,
		time.Second,
		[]string{"http://localhost:3000"},
		readinessCheckerFunc(func(context.Context) error { return nil }),
		metrics.New(prometheus.NewRegistry()),
		nil,
	)

	t.Run("healthz", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
		if got := rec.Header().Get("X-Request-ID"); got == "" {
			t.Fatal("expected X-Request-ID header")
		}
	})

	t.Run("readyz", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		server.server.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", rec.Code)
		}
	})
}

type readinessCheckerFunc func(context.Context) error

func (f readinessCheckerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}
