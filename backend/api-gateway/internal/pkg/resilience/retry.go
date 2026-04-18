// Package resilience contains retry and circuit-breaker helpers.
package resilience

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client wraps downstream calls with retry and circuit breaker.
type Client struct {
	breaker *gobreaker.CircuitBreaker
	retries int
	backoff time.Duration
}

// New creates a resilience client.
func New(name string, retries int, backoff time.Duration) *Client {
	return &Client{
		breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        name,
			MaxRequests: 5,
			Interval:    time.Minute,
			Timeout:     10 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5
			},
		}),
		retries: retries,
		backoff: backoff,
	}
}

// Do executes fn with retry and circuit breaking.
func Do[T any](ctx context.Context, client *Client, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error
	attempts := client.retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(client.backoff * time.Duration(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return zero, fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-timer.C:
			}
		}
		result, err := client.breaker.Execute(func() (any, error) {
			return fn(ctx)
		})
		if err == nil {
			value, ok := result.(T)
			if !ok {
				return zero, fmt.Errorf("unexpected downstream result type")
			}
			return value, nil
		}
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return zero, lastErr
}

func retryable(err error) bool {
	code := status.Code(err)
	return code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.ResourceExhausted || code == codes.Aborted || code == codes.Unknown
}
