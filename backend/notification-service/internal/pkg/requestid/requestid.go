package requestid

import (
	"context"
	"github.com/google/uuid"
)

type key struct{}

func New() string                                         { return uuid.NewString() }
func With(ctx context.Context, id string) context.Context { return context.WithValue(ctx, key{}, id) }
func From(ctx context.Context) string                     { v, _ := ctx.Value(key{}).(string); return v }
