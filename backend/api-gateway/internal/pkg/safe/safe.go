// Package safe runs goroutines with panic recovery.
package safe

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
)

// Go runs fn in a goroutine and sends returned errors or panics to errCh.
func Go(ctx context.Context, logger *slog.Logger, name string, errCh chan<- error, fn func(context.Context) error) {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err := fmt.Errorf("panic in goroutine %s: %v", name, recovered)
				logger.ErrorContext(ctx, "goroutine panic recovered", "name", name, "error", err, "stack", string(debug.Stack()))
				errCh <- err
			}
		}()
		if err := fn(ctx); err != nil {
			errCh <- fmt.Errorf("%s: %w", name, err)
		}
	}()
}
