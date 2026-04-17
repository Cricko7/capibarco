package application

import (
	"context"
	"time"
)

type RetryPolicy struct {
	Attempts int
	Backoff  time.Duration
}

func (p RetryPolicy) Do(ctx context.Context, fn func(context.Context) error) error {
	attempts := p.Attempts
	if attempts <= 0 {
		attempts = 1
	}
	backoff := p.Backoff
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
		if backoff <= 0 {
			continue
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}
