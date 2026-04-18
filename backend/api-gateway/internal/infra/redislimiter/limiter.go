// Package redislimiter implements Redis-backed fixed-window rate limiting.
package redislimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Decision is a rate-limit decision for one bucket.
type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Limiter stores counters in Redis with per-window expiration.
type Limiter struct {
	client *redis.Client
	prefix string
}

// New creates a Redis-backed limiter.
func New(client *redis.Client, prefix string) *Limiter {
	return &Limiter{client: client, prefix: prefix}
}

// Allow increments a bucket and reports whether it is still under limit.
func (l *Limiter) Allow(ctx context.Context, key string, limit int64, window time.Duration) (Decision, error) {
	if key == "" {
		return Decision{Allowed: true}, nil
	}
	now := time.Now().UTC()
	windowStart := now.Truncate(window).Unix()
	redisKey := fmt.Sprintf("%s:%s:%d", l.prefix, key, windowStart)
	count, err := l.client.Incr(ctx, redisKey).Result()
	if err != nil {
		return Decision{}, fmt.Errorf("redis incr rate limit bucket: %w", err)
	}
	if count == 1 {
		if err := l.client.Expire(ctx, redisKey, window*2).Err(); err != nil {
			return Decision{}, fmt.Errorf("redis expire rate limit bucket: %w", err)
		}
	}
	if count > limit {
		return Decision{Allowed: false, RetryAfter: window - now.Sub(now.Truncate(window))}, nil
	}
	return Decision{Allowed: true}, nil
}

// Ping verifies Redis availability.
func (l *Limiter) Ping(ctx context.Context) error {
	if err := l.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}
