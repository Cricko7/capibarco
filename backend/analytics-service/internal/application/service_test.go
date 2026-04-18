package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/petmatch/petmatch/internal/domain"
)

type repoMock struct{ mock.Mock }

type publisherMock struct{ mock.Mock }

type clockMock struct{}

func (clockMock) Now() time.Time { return time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC) }

func (m *repoMock) IngestRawEvent(ctx context.Context, event domain.Event) (bool, error) {
	args := m.Called(ctx, event)
	return args.Bool(0), args.Error(1)
}
func (m *repoMock) AggregateEvent(ctx context.Context, event domain.Event, bucket domain.BucketSize) error {
	return m.Called(ctx, event, bucket).Error(0)
}
func (m *repoMock) MetricsByBucket(ctx context.Context, profileID string, from, to time.Time, bucket domain.BucketSize) ([]domain.ProfileMetric, error) {
	args := m.Called(ctx, profileID, from, to, bucket)
	return args.Get(0).([]domain.ProfileMetric), args.Error(1)
}
func (m *repoMock) ExtendedStats(ctx context.Context, profileID string, from, to time.Time) (domain.ExtendedStats, error) {
	args := m.Called(ctx, profileID, from, to)
	return args.Get(0).(domain.ExtendedStats), args.Error(1)
}
func (m *repoMock) RankingFeedback(ctx context.Context, from, to time.Time, limit int) ([]domain.RankingFeedback, error) {
	args := m.Called(ctx, from, to, limit)
	return args.Get(0).([]domain.RankingFeedback), args.Error(1)
}
func (m *repoMock) Ping(ctx context.Context) error { return m.Called(ctx).Error(0) }

func (m *publisherMock) PublishRankingFeedback(ctx context.Context, items []domain.RankingFeedback) error {
	return m.Called(ctx, items).Error(0)
}

func TestServiceIngestEvent_Duplicate(t *testing.T) {
	repo := new(repoMock)
	pub := new(publisherMock)
	svc := NewService(repo, clockMock{}, pub, ServiceConfig{})
	e := domain.Event{EventID: "e1", ProfileID: "p1", ActorID: "a1", Type: domain.EventView}

	repo.On("IngestRawEvent", mock.Anything, mock.Anything).Return(false, nil).Once()

	err := svc.IngestEvent(context.Background(), e)
	require.ErrorIs(t, err, ErrDuplicateEvent)
}

func TestServiceExtendedStats_Forbidden(t *testing.T) {
	svc := NewService(new(repoMock), clockMock{}, new(publisherMock), ServiceConfig{})
	_, err := svc.ExtendedStats(context.Background(), "guest", "p1", time.Now(), time.Now())
	require.Error(t, err)
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestServiceRankingFeedback_Retry(t *testing.T) {
	repo := new(repoMock)
	pub := new(publisherMock)
	svc := NewService(repo, clockMock{}, pub, ServiceConfig{Retries: 2, Backoff: time.Millisecond})
	from := time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	items := []domain.RankingFeedback{{ProfileID: "p1"}}

	repo.On("RankingFeedback", mock.Anything, from, to, 10).Return(items, nil).Once()
	pub.On("PublishRankingFeedback", mock.Anything, items).Return(errors.New("downstream")).Once()
	pub.On("PublishRankingFeedback", mock.Anything, items).Return(nil).Once()

	result, err := svc.RankingFeedback(context.Background(), from, to, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
}
