package notification

import (
	"context"
	"testing"
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/notification"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateNotificationReturnsExistingByIdempotencyKey(t *testing.T) {
	t.Parallel()

	existing := domain.Notification{
		ID:                 "notification-1",
		RecipientProfileID: "profile-1",
		Type:               domain.TypeChatMessage,
		Channels:           []domain.Channel{domain.ChannelPush},
		Title:              "existing",
		Body:               "existing body",
		Status:             domain.StatusDelivered,
		IdempotencyKey:     "idem-1",
		CreatedAt:          time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC),
	}

	repo := newRepositoryMock(t)
	publisher := newPublisherMock(t)
	repo.On("FindNotificationByIdempotencyKey", mock.Anything, "profile-1", "idem-1").Return(existing, nil).Once()

	service := NewService(repo, publisher, "notification-service", "notification", time.Now, nil)

	got, err := service.CreateNotification(context.Background(), domain.Notification{
		RecipientProfileID: "profile-1",
		Type:               domain.TypeChatMessage,
		Channels:           []domain.Channel{domain.ChannelPush},
		Title:              "new",
		Body:               "body",
		IdempotencyKey:     "idem-1",
	})

	require.NoError(t, err)
	require.Equal(t, existing, got)
}

func TestMarkNotificationReadPublishesReadEvent(t *testing.T) {
	t.Parallel()

	readAt := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	repo := newRepositoryMock(t)
	publisher := newPublisherMock(t)
	service := NewService(repo, publisher, "notification-service", "notification", func() time.Time { return readAt }, nil)

	notification := domain.Notification{
		ID:                 "notification-1",
		RecipientProfileID: "profile-1",
		IdempotencyKey:     "idem-1",
		Status:             domain.StatusRead,
		ReadAt:             &readAt,
	}

	repo.On("MarkNotificationRead", mock.Anything, "notification-1", "profile-1", readAt).Return(notification, nil).Once()
	publisher.On("Publish", mock.Anything, "notification.read", "profile-1", mock.AnythingOfType("[]uint8")).Return(nil).Once()

	got, err := service.MarkNotificationRead(context.Background(), "notification-1", "profile-1")

	require.NoError(t, err)
	require.Equal(t, notification, got)
}

type repositoryMock struct {
	mock.Mock
}

func newRepositoryMock(t *testing.T) *repositoryMock {
	t.Helper()

	repo := &repositoryMock{}
	t.Cleanup(func() { repo.AssertExpectations(t) })
	return repo
}

func (m *repositoryMock) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *repositoryMock) RegisterDevice(ctx context.Context, token domain.DeviceToken) (domain.DeviceToken, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(domain.DeviceToken), args.Error(1)
}

func (m *repositoryMock) UnregisterDevice(ctx context.Context, deviceTokenID string) (bool, error) {
	args := m.Called(ctx, deviceTokenID)
	return args.Bool(0), args.Error(1)
}

func (m *repositoryMock) FindNotificationByIdempotencyKey(ctx context.Context, recipientProfileID, idempotencyKey string) (domain.Notification, error) {
	args := m.Called(ctx, recipientProfileID, idempotencyKey)
	return args.Get(0).(domain.Notification), args.Error(1)
}

func (m *repositoryMock) CreateNotification(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	args := m.Called(ctx, notification)
	return args.Get(0).(domain.Notification), args.Error(1)
}

func (m *repositoryMock) UpdateNotificationStatus(ctx context.Context, notificationID string, status domain.Status) error {
	args := m.Called(ctx, notificationID, status)
	return args.Error(0)
}

func (m *repositoryMock) ListNotifications(ctx context.Context, recipientProfileID string, statuses []domain.Status, page domain.PageRequest) ([]domain.Notification, string, error) {
	args := m.Called(ctx, recipientProfileID, statuses, page)
	return args.Get(0).([]domain.Notification), args.String(1), args.Error(2)
}

func (m *repositoryMock) MarkNotificationRead(ctx context.Context, notificationID, recipientProfileID string, readAt time.Time) (domain.Notification, error) {
	args := m.Called(ctx, notificationID, recipientProfileID, readAt)
	return args.Get(0).(domain.Notification), args.Error(1)
}

func (m *repositoryMock) GetPreference(ctx context.Context, recipientProfileID string) (domain.Preference, error) {
	args := m.Called(ctx, recipientProfileID)
	return args.Get(0).(domain.Preference), args.Error(1)
}

type publisherMock struct {
	mock.Mock
}

func newPublisherMock(t *testing.T) *publisherMock {
	t.Helper()

	publisher := &publisherMock{}
	t.Cleanup(func() { publisher.AssertExpectations(t) })
	return publisher
}

func (m *publisherMock) Publish(ctx context.Context, topic, key string, payload []byte) error {
	args := m.Called(ctx, topic, key, payload)
	return args.Error(0)
}
