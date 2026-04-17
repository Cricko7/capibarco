package platform

import (
	"context"
	"log/slog"
)

// Go starts fn in a goroutine and recovers panics with structured logging.
func Go(ctx context.Context, logger *slog.Logger, name string, fn func(context.Context)) {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx, "goroutine panic recovered", "goroutine", name, "panic", recovered)
			}
		}()
		fn(ctx)
	}()
}
