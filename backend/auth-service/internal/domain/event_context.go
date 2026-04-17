package domain

import "context"

type eventMetaContextKey struct{}

// WithEventMeta attaches event metadata to a context.
func WithEventMeta(ctx context.Context, meta EventMeta) context.Context {
	return context.WithValue(ctx, eventMetaContextKey{}, meta)
}

// EventMetaFromContext returns event metadata from a context.
func EventMetaFromContext(ctx context.Context) EventMeta {
	meta, ok := ctx.Value(eventMetaContextKey{}).(EventMeta)
	if !ok {
		return EventMeta{}
	}
	return meta
}
