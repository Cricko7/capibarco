package platform

import (
	"context"
	"fmt"
	"log/slog"
)

func Go(ctx context.Context, logger *slog.Logger, name string, fn func() error, errCh chan<- error) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in %s: %v", name, r)
				logger.ErrorContext(ctx, "panic recovered", slog.String("name", name), slog.Any("panic", r))
				errCh <- err
			}
		}()
		if err := fn(); err != nil {
			errCh <- err
		}
	}()
}
