package grpc

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		requestID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("x-request-id"); len(values) > 0 {
				requestID = values[0]
			}
		}
		if requestID == "" {
			requestID = uuid.NewString()
		}
		ctx = context.WithValue(ctx, requestIDKey{}, requestID)
		return handler(ctx, req)
	}
}

func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			logger.ErrorContext(ctx, "grpc request failed", slog.String("method", info.FullMethod), slog.String("error", err.Error()))
			return nil, err
		}
		logger.InfoContext(ctx, "grpc request completed", slog.String("method", info.FullMethod))
		return resp, nil
	}
}

func RecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(ctx, "grpc panic recovered", slog.String("method", info.FullMethod), slog.Any("panic", rec))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

type requestIDKey struct{}
