package application

import (
	"context"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
)

type Repository interface {
	IngestRawEvent(ctx context.Context, event domain.Event) (bool, error)
	AggregateEvent(ctx context.Context, event domain.Event, bucket domain.BucketSize) error
	MetricsByBucket(ctx context.Context, profileID string, from, to time.Time, bucket domain.BucketSize) ([]domain.ProfileMetric, error)
	ExtendedStats(ctx context.Context, profileID string, from, to time.Time) (domain.ExtendedStats, error)
	RankingFeedback(ctx context.Context, from, to time.Time, limit int) ([]domain.RankingFeedback, error)
	Ping(ctx context.Context) error
}

type FeedbackPublisher interface {
	PublishRankingFeedback(ctx context.Context, items []domain.RankingFeedback) error
}

type Clock interface {
	Now() time.Time
}
