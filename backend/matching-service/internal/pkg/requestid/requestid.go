// Package requestid propagates request identifiers through context.
package requestid

import (
	"context"

	"github.com/google/uuid"
)

type contextKey struct{}

// New returns a fresh request id.
func New() string {
	return uuid.NewString()
}

// With stores request id in context.
func With(ctx context.Context, id string) context.Context {
	if id == "" {
		id = New()
	}
	return context.WithValue(ctx, contextKey{}, id)
}

// From returns request id from context.
func From(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}
