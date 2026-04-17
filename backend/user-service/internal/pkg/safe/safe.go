package safe

import (
	"context"
	"fmt"
	"log/slog"
)

func Go(ctx context.Context, logger *slog.Logger, name string, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if logger != nil {
					logger.Error("panic recovered", "goroutine", name, "panic", r)
				}
				panic(fmt.Sprintf("panic in %s: %v", name, r))
			}
		}()
		fn(ctx)
	}()
}
