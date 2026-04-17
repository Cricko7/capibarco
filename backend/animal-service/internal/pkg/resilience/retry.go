// Package resilience provides retry and circuit breaker helpers.
package resilience

import (
	"context"
	"fmt"
	"time"
)

// Retry retries fn with exponential backoff until attempts are exhausted.
func Retry(ctx context.Context, attempts int, baseDelay time.Duration, fn func(context.Context) error) error {
	if attempts <= 0 {
		attempts = 1
	}
	if baseDelay <= 0 {
		baseDelay = 50 * time.Millisecond
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(ctx); err != nil {
			lastErr = err
			if attempt == attempts {
				break
			}
			timer := time.NewTimer(baseDelay * time.Duration(1<<(attempt-1)))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("retry exhausted after %d attempts: %w", attempts, lastErr)
}
