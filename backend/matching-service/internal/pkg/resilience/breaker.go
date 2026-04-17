package resilience

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker open")

type breakerState int

const (
	breakerClosed breakerState = iota
	breakerOpen
	breakerHalfOpen
)

// CircuitBreaker is a small thread-safe failure threshold breaker.
type CircuitBreaker struct {
	mu          sync.Mutex
	state       breakerState
	failures    int
	threshold   int
	openedAt    time.Time
	openTimeout time.Duration
	now         func() time.Time
}

// NewCircuitBreaker creates a circuit breaker.
func NewCircuitBreaker(threshold int, openTimeout time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if openTimeout <= 0 {
		openTimeout = 30 * time.Second
	}
	return &CircuitBreaker{threshold: threshold, openTimeout: openTimeout, now: time.Now}
}

// Execute runs fn when the breaker allows it and updates breaker state.
func (b *CircuitBreaker) Execute(fn func() error) error {
	if err := b.before(); err != nil {
		return err
	}
	if err := fn(); err != nil {
		b.afterFailure()
		return err
	}
	b.afterSuccess()
	return nil
}

func (b *CircuitBreaker) before() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == breakerOpen {
		if b.now().Sub(b.openedAt) < b.openTimeout {
			return ErrCircuitOpen
		}
		b.state = breakerHalfOpen
	}
	return nil
}

func (b *CircuitBreaker) afterFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold || b.state == breakerHalfOpen {
		b.state = breakerOpen
		b.openedAt = b.now()
	}
}

func (b *CircuitBreaker) afterSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = breakerClosed
}
