package grpc

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/petmatch/chat-service/internal/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const requestIDHeader = "x-request-id"

// UnaryInterceptors returns production middleware for unary RPCs.
func UnaryInterceptors(logger *slog.Logger, metrics *observability.Metrics) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		requestIDUnaryInterceptor(),
		recoveryUnaryInterceptor(logger),
		loggingUnaryInterceptor(logger),
		metricsUnaryInterceptor(metrics),
	)
}

// StreamInterceptors returns production middleware for streaming RPCs.
func StreamInterceptors(logger *slog.Logger, metrics *observability.Metrics) grpc.ServerOption {
	return grpc.ChainStreamInterceptor(
		requestIDStreamInterceptor(),
		recoveryStreamInterceptor(logger),
		loggingStreamInterceptor(logger),
		metricsStreamInterceptor(metrics),
	)
}

func requestIDUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(contextWithRequestID(ctx), req)
	}
}

func requestIDStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &wrappedServerStream{ServerStream: stream, ctx: contextWithRequestID(stream.Context())})
	}
}

func recoveryUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx, "grpc panic recovered", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal chat service error")
			}
		}()
		return handler(ctx, req)
	}
}

func recoveryStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(stream.Context(), "grpc stream panic recovered", "method", info.FullMethod, "panic", recovered, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal chat service error")
			}
		}()
		return handler(srv, stream)
	}
}

func loggingUnaryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		logger.InfoContext(ctx, "grpc unary request",
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", observability.RequestIDFromContext(ctx),
		)
		return resp, err
	}
}

func loggingStreamInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, stream)
		logger.InfoContext(stream.Context(), "grpc stream request",
			"method", info.FullMethod,
			"code", status.Code(err).String(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", observability.RequestIDFromContext(stream.Context()),
		)
		return err
	}
}

func metricsUnaryInterceptor(metrics *observability.Metrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		if metrics != nil {
			metrics.GRPCRequests.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
			metrics.GRPCDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		}
		return resp, err
	}
}

func metricsStreamInterceptor(metrics *observability.Metrics) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, stream)
		if metrics != nil {
			metrics.GRPCRequests.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()
			metrics.GRPCDuration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		}
		return err
	}
}

func contextWithRequestID(ctx context.Context) context.Context {
	requestID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(requestIDHeader)
		if len(values) > 0 {
			requestID = values[0]
		}
	}
	if requestID == "" {
		requestID = uuid.NewString()
	}
	return observability.ContextWithRequestID(ctx, requestID)
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
