package grpcserver

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Metrics contains Prometheus collectors for gRPC requests.
type Metrics struct {
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
}

// NewMetrics registers gRPC Prometheus collectors.
func NewMetrics(registry *prometheus.Registry) *Metrics {
	metrics := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "animal_grpc_requests_total",
			Help: "Total gRPC requests handled by animal-service.",
		}, []string{"method", "code"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "animal_grpc_request_duration_seconds",
			Help:    "gRPC request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method"}),
	}
	registry.MustRegister(metrics.requests, metrics.latency)
	return metrics
}

// UnaryInterceptor returns logging, request-id, recovery, and metrics middleware.
func UnaryInterceptor(logger *slog.Logger, metrics *Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		started := time.Now()
		requestID := requestIDFromContext(ctx)
		if requestID == "" {
			requestID = uuid.NewString()
			ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(headerRequestID, requestID))
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx, "panic recovered in grpc handler", "method", info.FullMethod, "request_id", requestID, "panic", recovered)
				err = status.Error(codes.Internal, "internal server error")
			}
			code := status.Code(err).String()
			metrics.requests.WithLabelValues(info.FullMethod, code).Inc()
			metrics.latency.WithLabelValues(info.FullMethod).Observe(time.Since(started).Seconds())
			logger.InfoContext(ctx, "grpc request completed", "method", info.FullMethod, "code", code, "request_id", requestID, "duration_ms", time.Since(started).Milliseconds())
		}()
		return handler(ctx, req)
	}
}
