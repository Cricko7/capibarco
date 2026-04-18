package notification

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	domain "github.com/petmatch/petmatch/internal/domain/notification"
	"github.com/stretchr/testify/require"
)

func TestCreateNotificationAppliesQuietHours(t *testing.T) {
	repo := &memoryRepo{
		preferences: map[string]domain.Preference{
			"owner-1": {
				RecipientProfileID: "owner-1",
				PushEnabled:        true,
				InAppEnabled:       true,
				EmailEnabled:       true,
				QuietHoursEnabled:  true,
				QuietHoursStart:    "22:00",
				QuietHoursEnd:      "08:00",
			},
		},
	}
	pub := &memoryPublisher{}
	now := func() time.Time { return time.Date(2026, 4, 18, 23, 30, 0, 0, time.UTC) }
	service := NewService(repo, pub, "notification-service", "notification", now, nil)

	n, err := service.CreateNotification(context.Background(), domain.Notification{
		RecipientProfileID: "owner-1",
		Type:               domain.TypeMatchCreated,
		Channels:           []domain.Channel{domain.ChannelPush, domain.ChannelInApp, domain.ChannelEmail},
		Title:              "title",
		Body:               "body",
	})

	require.NoError(t, err)
	require.Equal(t, []domain.Channel{domain.ChannelInApp}, n.Channels)
	require.Equal(t, domain.StatusDelivered, n.Status)
	require.Len(t, pub.topics, 2)
}

func TestCreateNotificationMarksFailedWhenRecipientMuted(t *testing.T) {
	repo := &memoryRepo{
		preferences: map[string]domain.Preference{
			"owner-1": {RecipientProfileID: "owner-1", Muted: true},
		},
	}
	pub := &memoryPublisher{}
	service := NewService(repo, pub, "notification-service", "notification", time.Now, nil)

	n, err := service.CreateNotification(context.Background(), domain.Notification{
		RecipientProfileID: "owner-1",
		Type:               domain.TypeReviewCreated,
		Channels:           []domain.Channel{domain.ChannelPush},
		Title:              "title",
		Body:               "body",
	})

	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, n.Status)
	require.Len(t, pub.topics, 2)
}

func TestEnvelopeKeyFallsBackToEventID(t *testing.T) {
	require.Equal(t, "event-1", envelopeKey(&commonv1.EventEnvelope{EventId: "event-1"}))
}

func TestProtoEnumAlignment(t *testing.T) {
	require.Equal(t, notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_PUSH, notificationv1.NotificationChannel(domain.ChannelPush))
}

type memoryRepo struct {
	preferences   map[string]domain.Preference
	notifications map[string]domain.Notification
	byKey         map[string]string
}

func (m *memoryRepo) Ping(context.Context) error { return nil }

func (m *memoryRepo) RegisterDevice(context.Context, domain.DeviceToken) (domain.DeviceToken, error) {
	return domain.DeviceToken{}, nil
}

func (m *memoryRepo) UnregisterDevice(context.Context, string) (bool, error) { return true, nil }

func (m *memoryRepo) FindNotificationByIdempotencyKey(_ context.Context, recipientProfileID, idempotencyKey string) (domain.Notification, error) {
	if m.byKey == nil {
		return domain.Notification{}, domain.ErrNotFound
	}
	id, ok := m.byKey[recipientProfileID+":"+idempotencyKey]
	if !ok {
		return domain.Notification{}, domain.ErrNotFound
	}
	return m.notifications[id], nil
}

func (m *memoryRepo) CreateNotification(_ context.Context, notification domain.Notification) (domain.Notification, error) {
	if m.notifications == nil {
		m.notifications = map[string]domain.Notification{}
	}
	if m.byKey == nil {
		m.byKey = map[string]string{}
	}
	m.notifications[notification.ID] = notification
	if notification.IdempotencyKey != "" {
		m.byKey[notification.RecipientProfileID+":"+notification.IdempotencyKey] = notification.ID
	}
	return notification, nil
}

func (m *memoryRepo) UpdateNotificationStatus(_ context.Context, notificationID string, status domain.Status) error {
	notification := m.notifications[notificationID]
	notification.Status = status
	m.notifications[notificationID] = notification
	return nil
}

func (m *memoryRepo) ListNotifications(context.Context, string, []domain.Status, domain.PageRequest) ([]domain.Notification, string, error) {
	return nil, "", nil
}

func (m *memoryRepo) MarkNotificationRead(_ context.Context, notificationID, _ string, readAt time.Time) (domain.Notification, error) {
	notification := m.notifications[notificationID]
	notification.Status = domain.StatusRead
	notification.ReadAt = &readAt
	m.notifications[notificationID] = notification
	return notification, nil
}

func (m *memoryRepo) GetPreference(_ context.Context, recipientProfileID string) (domain.Preference, error) {
	if preference, ok := m.preferences[recipientProfileID]; ok {
		return preference, nil
	}
	return domain.DefaultPreference(recipientProfileID), nil
}

type memoryPublisher struct {
	topics []string
}

func (m *memoryPublisher) Publish(_ context.Context, topic, _ string, _ []byte) error {
	m.topics = append(m.topics, topic)
	return nil
}
