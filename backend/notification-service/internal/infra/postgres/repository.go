package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	domain "github.com/petmatch/petmatch/internal/domain/notification"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) RegisterDevice(ctx context.Context, token domain.DeviceToken) (domain.DeviceToken, error) {
	const q = `
INSERT INTO device_tokens (device_token_id, profile_id, token, platform, locale, active, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (profile_id, token) DO UPDATE
SET platform = EXCLUDED.platform,
    locale = EXCLUDED.locale,
    active = TRUE,
    updated_at = EXCLUDED.updated_at
RETURNING device_token_id, profile_id, token, platform, locale, active, created_at, updated_at`
	if err := r.pool.QueryRow(ctx, q, token.ID, token.ProfileID, token.Token, token.Platform, token.Locale, token.Active, token.CreatedAt, token.UpdatedAt).
		Scan(&token.ID, &token.ProfileID, &token.Token, &token.Platform, &token.Locale, &token.Active, &token.CreatedAt, &token.UpdatedAt); err != nil {
		return domain.DeviceToken{}, fmt.Errorf("register device: %w", err)
	}
	return token, nil
}

func (r *Repository) UnregisterDevice(ctx context.Context, deviceTokenID string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE device_tokens SET active = FALSE, updated_at = now() WHERE device_token_id = $1`, deviceTokenID)
	if err != nil {
		return false, fmt.Errorf("unregister device: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) FindNotificationByIdempotencyKey(ctx context.Context, recipientProfileID, idempotencyKey string) (domain.Notification, error) {
	const q = `
SELECT notification_id, recipient_profile_id, type, channels, title, body, data, status, read_at, created_at, COALESCE(idempotency_key, '')
FROM notifications
WHERE recipient_profile_id = $1 AND idempotency_key = $2`
	return scanNotification(r.pool.QueryRow(ctx, q, recipientProfileID, idempotencyKey))
}

func (r *Repository) CreateNotification(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	payload, err := json.Marshal(notification.Data)
	if err != nil {
		return domain.Notification{}, fmt.Errorf("marshal notification data: %w", err)
	}
	const q = `
INSERT INTO notifications (notification_id, recipient_profile_id, type, channels, title, body, data, status, read_at, created_at, idempotency_key)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING notification_id, recipient_profile_id, type, channels, title, body, data, status, read_at, created_at, COALESCE(idempotency_key, '')`
	return scanNotification(r.pool.QueryRow(
		ctx,
		q,
		notification.ID,
		notification.RecipientProfileID,
		int16(notification.Type),
		pq.Array(channelsToInts(notification.Channels)),
		notification.Title,
		notification.Body,
		payload,
		int16(notification.Status),
		notification.ReadAt,
		notification.CreatedAt,
		nullable(notification.IdempotencyKey),
	))
}

func (r *Repository) UpdateNotificationStatus(ctx context.Context, notificationID string, status domain.Status) error {
	_, err := r.pool.Exec(ctx, `UPDATE notifications SET status = $2 WHERE notification_id = $1`, notificationID, int16(status))
	if err != nil {
		return fmt.Errorf("update notification status: %w", err)
	}
	return nil
}

func (r *Repository) ListNotifications(ctx context.Context, recipientProfileID string, statuses []domain.Status, page domain.PageRequest) ([]domain.Notification, string, error) {
	limit := 20
	if page.PageSize > 0 && page.PageSize <= 100 {
		limit = int(page.PageSize)
	}
	offset := 0
	if strings.TrimSpace(page.PageToken) != "" {
		if value, err := strconv.Atoi(page.PageToken); err == nil && value >= 0 {
			offset = value
		}
	}

	query := `
SELECT notification_id, recipient_profile_id, type, channels, title, body, data, status, read_at, created_at, COALESCE(idempotency_key, '')
FROM notifications
WHERE recipient_profile_id = $1`
	args := []any{recipientProfileID}
	if len(statuses) > 0 {
		query += ` AND status = ANY($2)`
		args = append(args, pq.Array(statusesToInts(statuses)))
		query += ` ORDER BY created_at DESC, notification_id DESC LIMIT $3 OFFSET $4`
		args = append(args, limit+1, offset)
	} else {
		query += ` ORDER BY created_at DESC, notification_id DESC LIMIT $2 OFFSET $3`
		args = append(args, limit+1, offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]domain.Notification, 0, limit+1)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, "", err
		}
		notifications = append(notifications, notification)
	}
	next := ""
	if len(notifications) > limit {
		notifications = notifications[:limit]
		next = strconv.Itoa(offset + limit)
	}
	return notifications, next, rows.Err()
}

func (r *Repository) MarkNotificationRead(ctx context.Context, notificationID, recipientProfileID string, readAt time.Time) (domain.Notification, error) {
	const q = `
UPDATE notifications
SET status = $3, read_at = $4
WHERE notification_id = $1 AND recipient_profile_id = $2
RETURNING notification_id, recipient_profile_id, type, channels, title, body, data, status, read_at, created_at, COALESCE(idempotency_key, '')`
	return scanNotification(r.pool.QueryRow(ctx, q, notificationID, recipientProfileID, int16(domain.StatusRead), readAt))
}

func (r *Repository) GetPreference(ctx context.Context, recipientProfileID string) (domain.Preference, error) {
	const q = `
SELECT recipient_profile_id, push_enabled, in_app_enabled, email_enabled, quiet_hours_enabled, quiet_hours_start, quiet_hours_end, muted
FROM notification_preferences
WHERE recipient_profile_id = $1`
	var preference domain.Preference
	if err := r.pool.QueryRow(ctx, q, recipientProfileID).Scan(
		&preference.RecipientProfileID,
		&preference.PushEnabled,
		&preference.InAppEnabled,
		&preference.EmailEnabled,
		&preference.QuietHoursEnabled,
		&preference.QuietHoursStart,
		&preference.QuietHoursEnd,
		&preference.Muted,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.DefaultPreference(recipientProfileID), nil
		}
		return domain.Preference{}, fmt.Errorf("get preference: %w", err)
	}
	return preference, nil
}

func scanNotification(row interface{ Scan(dest ...any) error }) (domain.Notification, error) {
	var (
		notification domain.Notification
		data         []byte
		channels     []int16
		readAt       *time.Time
	)
	if err := row.Scan(
		&notification.ID,
		&notification.RecipientProfileID,
		&notification.Type,
		pq.Array(&channels),
		&notification.Title,
		&notification.Body,
		&data,
		&notification.Status,
		&readAt,
		&notification.CreatedAt,
		&notification.IdempotencyKey,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Notification{}, domain.ErrNotFound
		}
		return domain.Notification{}, fmt.Errorf("scan notification: %w", err)
	}
	notification.Channels = intsToChannels(channels)
	notification.ReadAt = readAt
	notification.Data = map[string]string{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &notification.Data); err != nil {
			return domain.Notification{}, fmt.Errorf("unmarshal notification data: %w", err)
		}
	}
	return notification, nil
}

func channelsToInts(channels []domain.Channel) []int16 {
	out := make([]int16, 0, len(channels))
	for _, channel := range channels {
		out = append(out, int16(channel))
	}
	return out
}

func intsToChannels(values []int16) []domain.Channel {
	out := make([]domain.Channel, 0, len(values))
	for _, value := range values {
		out = append(out, domain.Channel(value))
	}
	return out
}

func statusesToInts(values []domain.Status) []int16 {
	out := make([]int16, 0, len(values))
	for _, value := range values {
		out = append(out, int16(value))
	}
	return out
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
