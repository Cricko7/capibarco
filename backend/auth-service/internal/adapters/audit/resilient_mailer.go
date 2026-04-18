package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/hackathon/authsvc/internal/ports"
	"github.com/sony/gobreaker/v2"
)

// ResilientMailer wraps mail sending with retries and circuit breaker.
type ResilientMailer struct {
	base    ports.Mailer
	retries uint
	delay   time.Duration
	breaker *gobreaker.CircuitBreaker[struct{}]
}

func NewResilientMailer(base ports.Mailer, retries uint, delay time.Duration) *ResilientMailer {
	return &ResilientMailer{
		base:    base,
		retries: retries,
		delay:   delay,
		breaker: gobreaker.NewCircuitBreaker[struct{}](gobreaker.Settings{Timeout: 10 * time.Second, MaxRequests: 5}),
	}
}

func (m *ResilientMailer) SendPasswordReset(ctx context.Context, tenantID string, email string, resetToken string) error {
	if m.base == nil {
		return nil
	}
	var lastErr error
	for attempt := uint(1); attempt <= m.retries; attempt++ {
		_, err := m.breaker.Execute(func() (struct{}, error) {
			return struct{}{}, m.base.SendPasswordReset(ctx, tenantID, email, resetToken)
		})
		if err == nil {
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return fmt.Errorf("mail send interrupted: %w", ctx.Err())
		case <-time.After(m.delay):
		}
	}
	return fmt.Errorf("mail send failed after %d attempts: %w", m.retries, lastErr)
}
