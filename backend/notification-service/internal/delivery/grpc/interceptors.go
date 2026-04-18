package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
	"github.com/petmatch/petmatch/internal/pkg/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func UnaryInterceptor(logger *slog.Logger, m *metrics.Metrics, limiter *resilience.RateLimiter, timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		requestID := requestid.New()
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("x-request-id"); len(values) > 0 && values[0] != "" {
				requestID = values[0]
			}
		}
		ctx = requestid.With(ctx, requestID)
		if limiter != nil && !limiter.Allow() {
			if m != nil {
				m.GRPCRequests.WithLabelValues(info.FullMethod, codes.ResourceExhausted.String()).Inc()
			}
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err)
		if m != nil {
			m.GRPCRequests.WithLabelValues(info.FullMethod, code.String()).Inc()
		}
		if logger != nil {
			logger.Info("grpc request", "method", info.FullMethod, "code", code.String(), "duration", time.Since(start), "request_id", requestID)
		}
		return resp, err
	}
}
