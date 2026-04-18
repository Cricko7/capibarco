DROP INDEX IF EXISTS idx_notifications_recipient_created_at;
DROP INDEX IF EXISTS idx_notifications_recipient_idempotency;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS device_tokens;
