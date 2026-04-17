package grpcserver

import (
	"context"

	"google.golang.org/grpc/metadata"
)

const (
	headerActorID   = "x-actor-id"
	headerRequestID = "x-request-id"
)

func actorIDFromContext(ctx context.Context, fallback string) string {
	values := metadata.ValueFromIncomingContext(ctx, headerActorID)
	if len(values) > 0 && values[0] != "" {
		return values[0]
	}
	return fallback
}

func requestIDFromContext(ctx context.Context) string {
	values := metadata.ValueFromIncomingContext(ctx, headerRequestID)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}
