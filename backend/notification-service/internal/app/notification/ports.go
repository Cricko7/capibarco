package notification

import (
	"context"
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/notification"
)

type Repository interface {
	Ping(ctx context.Context) error
	RegisterDevice(ctx context.Context, token domain.DeviceToken) (domain.DeviceToken, error)
	UnregisterDevice(ctx context.Context, deviceTokenID string) (bool, error)
	FindNotificationByIdempotencyKey(ctx context.Context, recipientProfileID, idempotencyKey string) (domain.Notification, error)
	CreateNotification(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	UpdateNotificationStatus(ctx context.Context, notificationID string, status domain.Status) error
	ListNotifications(ctx context.Context, recipientProfileID string, statuses []domain.Status, page domain.PageRequest) ([]domain.Notification, string, error)
	MarkNotificationRead(ctx context.Context, notificationID, recipientProfileID string, readAt time.Time) (domain.Notification, error)
	GetPreference(ctx context.Context, recipientProfileID string) (domain.Preference, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
}
