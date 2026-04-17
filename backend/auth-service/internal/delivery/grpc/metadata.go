package grpc

import (
	"context"

	"github.com/hackathon/authsvc/internal/domain"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// EventMetadataUnaryInterceptor extracts event metadata from gRPC metadata.
func EventMetadataUnaryInterceptor() gogrpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *gogrpc.UnaryServerInfo, handler gogrpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}
		meta := domain.EventMeta{
			TraceID:        firstMetadata(md, "x-trace-id", "trace-id"),
			CorrelationID:  firstMetadata(md, "x-correlation-id", "correlation-id"),
			IdempotencyKey: firstMetadata(md, "x-idempotency-key", "idempotency-key"),
		}
		return handler(domain.WithEventMeta(ctx, meta), req)
	}
}

func firstMetadata(md metadata.MD, keys ...string) string {
	for _, key := range keys {
		values := md.Get(key)
		if len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
