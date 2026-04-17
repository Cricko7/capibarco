package grpc

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryInterceptor adds request id, structured logs, panic recovery, and metrics.
func UnaryInterceptor(logger *slog.Logger, m *metrics.Metrics) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		started := time.Now()
		ctx = requestid.With(ctx, requestIDFromMetadata(ctx))
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("grpc panic recovered", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal matching-service error")
			}
			code := status.Code(err).String()
			m.ObserveGRPC(info.FullMethod, code, started)
			logger.Info("grpc request", "method", info.FullMethod, "code", code, "duration", time.Since(started), "request_id", requestid.From(ctx))
		}()
		return handler(ctx, req)
	}
}

func requestIDFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return requestid.New()
	}
	values := md.Get("x-request-id")
	if len(values) == 0 || values[0] == "" {
		return requestid.New()
	}
	return values[0]
}
