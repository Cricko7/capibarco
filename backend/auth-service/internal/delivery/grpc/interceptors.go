package grpc

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RecoveryUnaryInterceptor(logger *slog.Logger) gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(ctx, "panic recovered", slog.String("method", info.FullMethod), slog.Any("panic", rec), slog.String("stack", string(debug.Stack())))
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

func RequestIDUnaryInterceptor() gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		requestID := firstMetadata(md, "x-request-id", "request-id")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		md = md.Copy()
		md.Set("x-request-id", requestID)
		return handler(metadata.NewIncomingContext(ctx, md), req)
	}
}

func LoggingUnaryInterceptor(logger *slog.Logger) gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		started := time.Now()
		resp, err := handler(ctx, req)
		logger.InfoContext(ctx, "grpc request", slog.String("method", info.FullMethod), slog.Duration("duration", time.Since(started)), slog.Any("error", err))
		return resp, err
	}
}

func RateLimitUnaryInterceptor(limiter *rate.Limiter) gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
