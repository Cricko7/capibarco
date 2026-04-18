package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
)

var ErrDuplicateEvent = errors.New("duplicate event")

type Service struct {
	repo      Repository
	clock     Clock
	publisher FeedbackPublisher
	retries   int
	backoff   time.Duration
}

type ServiceConfig struct {
	Retries int
	Backoff time.Duration
}

func NewService(repo Repository, clock Clock, publisher FeedbackPublisher, cfg ServiceConfig) *Service {
	if cfg.Retries <= 0 {
		cfg.Retries = 3
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 50 * time.Millisecond
	}
	return &Service{repo: repo, clock: clock, publisher: publisher, retries: cfg.Retries, backoff: cfg.Backoff}
}

func (s *Service) IngestEvent(ctx context.Context, event domain.Event) error {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = s.clock.Now()
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("validate event: %w", err)
	}

	inserted, err := s.repo.IngestRawEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("ingest event: %w", err)
	}
	if !inserted {
		return ErrDuplicateEvent
	}

	if err := s.repo.AggregateEvent(ctx, event, domain.BucketHour); err != nil {
		return fmt.Errorf("aggregate hour bucket: %w", err)
	}
	if err := s.repo.AggregateEvent(ctx, event, domain.BucketDay); err != nil {
		return fmt.Errorf("aggregate day bucket: %w", err)
	}
	return nil
}

func (s *Service) MetricsByBucket(ctx context.Context, profileID string, from, to time.Time, bucket domain.BucketSize) ([]domain.ProfileMetric, error) {
	if !bucket.IsValid() {
		return nil, fmt.Errorf("invalid bucket %s", bucket)
	}
	if from.After(to) {
		return nil, fmt.Errorf("invalid range: from after to")
	}
	items, err := s.repo.MetricsByBucket(ctx, profileID, from, to, bucket)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	return items, nil
}

func (s *Service) ExtendedStats(ctx context.Context, role domain.ExtendedStatsRole, profileID string, from, to time.Time) (domain.ExtendedStats, error) {
	if !role.IsEntitled() {
		return domain.ExtendedStats{}, fmt.Errorf("role %q: %w", role, domain.ErrForbidden)
	}
	stats, err := s.repo.ExtendedStats(ctx, profileID, from, to)
	if err != nil {
		return domain.ExtendedStats{}, fmt.Errorf("extended stats: %w", err)
	}
	return stats, nil
}

func (s *Service) RankingFeedback(ctx context.Context, from, to time.Time, limit int) ([]domain.RankingFeedback, error) {
	items, err := s.repo.RankingFeedback(ctx, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("ranking feedback query: %w", err)
	}

	for attempt := 1; attempt <= s.retries; attempt++ {
		if err := s.publisher.PublishRankingFeedback(ctx, items); err == nil {
			return items, nil
		}
		if attempt == s.retries {
			return nil, fmt.Errorf("publish ranking feedback: %w", err)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * s.backoff):
		}
	}
	return items, nil
}

func (s *Service) Ready(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
