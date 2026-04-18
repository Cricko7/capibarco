package http

import "context"

type requestIDKey struct{}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func requestIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDKey{}).(string); ok {
		return value
	}
	return ""
}
