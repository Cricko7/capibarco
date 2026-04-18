// Package requestid stores request IDs in contexts.
package requestid

import "context"

type key struct{}

// With returns a context carrying requestID.
func With(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, key{}, requestID)
}

// From returns the request ID from ctx.
func From(ctx context.Context) string {
	value, _ := ctx.Value(key{}).(string)
	return value
}
