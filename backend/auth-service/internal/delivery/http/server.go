package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	stdhttp "net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics exports Prometheus counters for auth operations.
type Metrics struct {
	register *prometheus.CounterVec
	login    *prometheus.CounterVec
	refresh  *prometheus.CounterVec
	errors   *prometheus.CounterVec
}

// NewMetrics registers auth metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		register: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "auth_register_total", Help: "Registration attempts."}, []string{"outcome"}),
		login:    prometheus.NewCounterVec(prometheus.CounterOpts{Name: "auth_login_total", Help: "Login attempts."}, []string{"outcome"}),
		refresh:  prometheus.NewCounterVec(prometheus.CounterOpts{Name: "auth_refresh_total", Help: "Refresh attempts."}, []string{"outcome"}),
		errors:   prometheus.NewCounterVec(prometheus.CounterOpts{Name: "auth_errors_total", Help: "Auth errors."}, []string{"operation"}),
	}
	reg.MustRegister(m.register, m.login, m.refresh, m.errors)
	return m
}

func (m *Metrics) IncRegister(outcome string) { m.register.WithLabelValues(outcome).Inc() }
func (m *Metrics) IncLogin(outcome string)    { m.login.WithLabelValues(outcome).Inc() }
func (m *Metrics) IncRefresh(outcome string)  { m.refresh.WithLabelValues(outcome).Inc() }
func (m *Metrics) IncError(operation string)  { m.errors.WithLabelValues(operation).Inc() }

// Server hosts health and metrics endpoints.
type Server struct {
	server *stdhttp.Server
	db     *sql.DB
	logger *slog.Logger
}

// NewServer creates an HTTP server.
func NewServer(addr string, db *sql.DB, logger *slog.Logger, reg *prometheus.Registry) *Server {
	mux := stdhttp.NewServeMux()
	s := &Server{db: db, logger: logger}
	mux.HandleFunc("/healthz", s.healthz)
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	s.server = &stdhttp.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	if err := s.server.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) healthz(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	status := "ok"
	if s.db != nil {
		if err := s.db.PingContext(ctx); err != nil {
			status = "degraded"
			w.WriteHeader(stdhttp.StatusServiceUnavailable)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": status}); err != nil {
		s.logger.ErrorContext(r.Context(), "write health response", slog.Any("error", err))
	}
}
