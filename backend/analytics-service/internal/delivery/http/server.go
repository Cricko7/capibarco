package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"github.com/petmatch/petmatch/internal/observability"
)

type Server struct {
	mux      *http.ServeMux
	service  *application.Service
	logger   *slog.Logger
	validate *validator.Validate
	metrics  *observability.Metrics
}

type ingestRequest struct {
	EventID    string            `json:"event_id" validate:"required,uuid4"`
	ProfileID  string            `json:"profile_id" validate:"required"`
	ActorID    string            `json:"actor_id" validate:"required"`
	Type       string            `json:"type" validate:"required"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata"`
}

func NewServer(logger *slog.Logger, service *application.Service, reg *prometheus.Registry, metrics *observability.Metrics, limiter *rate.Limiter) http.Handler {
	s := &Server{
		mux:      http.NewServeMux(),
		service:  service,
		logger:   logger,
		validate: validator.New(validator.WithRequiredStructEnabled()),
		metrics:  metrics,
	}
	s.routes(reg)

	return s.withMiddlewares(s.mux, limiter)
}

func (s *Server) routes(reg *prometheus.Registry) {
	s.mux.Handle("GET /healthz", http.HandlerFunc(s.healthz))
	s.mux.Handle("GET /readyz", http.HandlerFunc(s.readyz))
	s.mux.Handle("POST /v1/events", http.HandlerFunc(s.ingest))
	s.mux.Handle("GET /v1/profiles/{profileID}/metrics", http.HandlerFunc(s.metricsByBucket))
	s.mux.Handle("GET /v1/profiles/{profileID}/extended", http.HandlerFunc(s.extendedStats))
	s.mux.Handle("GET /v1/ranking-feedback", http.HandlerFunc(s.rankingFeedback))
	s.mux.Handle("GET /metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.service.Ready(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("not ready: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) ingest(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("decode request: %w", err))
		return
	}
	if req.EventID == "" {
		req.EventID = uuid.NewString()
	}
	if err := s.validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("validate request: %w", err))
		return
	}
	if req.OccurredAt.IsZero() {
		req.OccurredAt = time.Now().UTC()
	}
	event := domain.Event{
		EventID: req.EventID, ProfileID: req.ProfileID, ActorID: req.ActorID,
		Type: domain.EventType(req.Type), OccurredAt: req.OccurredAt, Metadata: req.Metadata,
	}
	if err := s.service.IngestEvent(r.Context(), event); err != nil {
		if errors.Is(err, application.ErrDuplicateEvent) {
			s.metrics.EventsDuplicate.Inc()
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "duplicate_ignored"})
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.metrics.EventsIngestedTotal.Inc()
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (s *Server) metricsByBucket(w http.ResponseWriter, r *http.Request) {
	profileID := r.PathValue("profileID")
	from, to, bucket, err := parseQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	items, err := s.service.MetricsByBucket(r.Context(), profileID, from, to, bucket)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) extendedStats(w http.ResponseWriter, r *http.Request) {
	profileID := r.PathValue("profileID")
	from, to, _, err := parseQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	role := domain.ExtendedStatsRole(r.URL.Query().Get("role"))
	stats, err := s.service.ExtendedStats(r.Context(), role, profileID, from, to)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, domain.ErrForbidden) {
			status = http.StatusForbidden
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) rankingFeedback(w http.ResponseWriter, r *http.Request) {
	from, to, _, err := parseQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid limit"))
			return
		}
		limit = parsed
	}
	items, err := s.service.RankingFeedback(r.Context(), from, to, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func parseQuery(r *http.Request) (time.Time, time.Time, domain.BucketSize, error) {
	from, err := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("invalid from")
	}
	to, err := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, "", fmt.Errorf("invalid to")
	}
	bucket := domain.BucketSize(r.URL.Query().Get("bucket"))
	if bucket == "" {
		bucket = domain.BucketHour
	}
	return from, to, bucket, nil
}

func (s *Server) withMiddlewares(next http.Handler, limiter *rate.Limiter) http.Handler {
	return s.recovery(s.metricsMiddleware(s.requestID(s.security(s.cors(s.rateLimit(next, limiter))))))
}

func (s *Server) rateLimit(next http.Handler, limiter *rate.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			writeError(w, http.StatusTooManyRequests, fmt.Errorf("rate limit exceeded"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, requestID)))
	})
}

func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.ErrorContext(r.Context(), "panic recovered", slog.Any("panic", rec))
				writeError(w, http.StatusInternalServerError, fmt.Errorf("internal error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		route := r.URL.Path
		if strings.HasPrefix(route, "/v1/profiles/") {
			route = strings.Replace(route, r.PathValue("profileID"), "{profileID}", 1)
		}
		s.metrics.HTTPRequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(ww.status)).Inc()
		s.metrics.HTTPRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

type requestIDKey struct{}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
