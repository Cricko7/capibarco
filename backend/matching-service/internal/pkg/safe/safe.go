// Package safe provides panic-safe goroutine helpers.
package safe

import (
	"context"
	"log/slog"
	"runtime/debug"
)

// Go starts fn in a goroutine and logs panics with stack traces.
func Go(parent context.Context, logger *slog.Logger, name string, fn func(context.Context)) {
	if logger == nil {
		logger = slog.Default()
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("goroutine panic recovered", "goroutine", name, "panic", recovered, "stack", string(debug.Stack()))
			}
		}()
		fn(parent)
	}()
}
