package breaker

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Executor wraps calls with retry and a circuit breaker.
type Executor struct {
	breaker         *gobreaker.CircuitBreaker[any]
	maxElapsed      time.Duration
	initialInterval time.Duration
}

// Config controls retry/circuit-breaker behavior.
type Config struct {
	Name        string
	MaxElapsed  time.Duration
	Timeout     time.Duration
	MaxRequests uint32
}

// NewExecutor creates a resilient executor.
func NewExecutor(cfg Config) *Executor {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Timeout:     cfg.Timeout,
	}
	return &Executor{
		breaker:         gobreaker.NewCircuitBreaker[any](settings),
		maxElapsed:      cfg.MaxElapsed,
		initialInterval: 50 * time.Millisecond,
	}
}

// Do executes fn with bounded retry under circuit breaker protection.
func (e *Executor) Do(ctx context.Context, fn func(context.Context) error) error {
	deadline := time.Now().Add(e.maxElapsed)
	delay := e.initialInterval
	var lastErr error

	for {
		_, err := e.breaker.Execute(func() (any, error) {
			if callErr := fn(ctx); callErr != nil {
				return nil, callErr
			}
			return nil, nil
		})
		if err == nil {
			return nil
		}
		lastErr = err
		if time.Now().Add(delay).After(deadline) {
			return fmt.Errorf("resilient call failed: %w", lastErr)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("resilient call cancelled: %w", ctx.Err())
		case <-timer.C:
		}
		delay *= 2
		if delay > 500*time.Millisecond {
			delay = 500 * time.Millisecond
		}
	}
}
