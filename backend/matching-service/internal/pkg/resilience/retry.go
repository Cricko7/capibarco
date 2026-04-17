// Package resilience contains small retry and circuit-breaker primitives.
package resilience

import (
	"context"
	"fmt"
	"time"
)

// Retry executes fn up to attempts times with exponential backoff.
func Retry(ctx context.Context, attempts int, baseDelay time.Duration, fn func(context.Context) error) error {
	if attempts <= 0 {
		attempts = 1
	}
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := fn(ctx); err != nil {
			lastErr = err
		} else {
			return nil
		}
		if attempt == attempts {
			break
		}
		delay := baseDelay * time.Duration(1<<(attempt-1))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return lastErr
}
