package resilience

import (
	"context"
	"fmt"

	"github.com/sony/gobreaker"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
)

type BreakerPublisher struct {
	next    application.FeedbackPublisher
	breaker *gobreaker.CircuitBreaker
}

func NewBreakerPublisher(next application.FeedbackPublisher) *BreakerPublisher {
	st := gobreaker.Settings{Name: "ranking-feedback", MaxRequests: 5}
	return &BreakerPublisher{next: next, breaker: gobreaker.NewCircuitBreaker(st)}
}

func (b *BreakerPublisher) PublishRankingFeedback(ctx context.Context, items []domain.RankingFeedback) error {
	_, err := b.breaker.Execute(func() (any, error) {
		if err := b.next.PublishRankingFeedback(ctx, items); err != nil {
			return nil, err
		}
		return struct{}{}, nil
	})
	if err != nil {
		return fmt.Errorf("breaker publish: %w", err)
	}
	return nil
}

type LogPublisher struct{}

func (LogPublisher) PublishRankingFeedback(_ context.Context, _ []domain.RankingFeedback) error { return nil }
