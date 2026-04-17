package runtime

import (
	"context"
	"log/slog"
)

func Go(ctx context.Context, logger *slog.Logger, name string, fn func(context.Context)) {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(ctx, "goroutine panic recovered", slog.String("name", name), slog.Any("panic", recovered))
			}
		}()
		fn(ctx)
	}()
}
