CREATE TABLE IF NOT EXISTS device_tokens (
  device_token_id TEXT PRIMARY KEY,
  profile_id TEXT NOT NULL,
  token TEXT NOT NULL,
  platform TEXT NOT NULL,
  locale TEXT NOT NULL DEFAULT '',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (profile_id, token)
);

CREATE TABLE IF NOT EXISTS notification_preferences (
  recipient_profile_id TEXT PRIMARY KEY,
  push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  in_app_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  quiet_hours_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  quiet_hours_start TEXT NOT NULL DEFAULT '',
  quiet_hours_end TEXT NOT NULL DEFAULT '',
  muted BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notifications (
  notification_id TEXT PRIMARY KEY,
  recipient_profile_id TEXT NOT NULL,
  type SMALLINT NOT NULL,
  channels SMALLINT[] NOT NULL DEFAULT '{}',
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,
  status SMALLINT NOT NULL,
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  idempotency_key TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notifications_recipient_idempotency
  ON notifications(recipient_profile_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_recipient_created_at
  ON notifications(recipient_profile_id, created_at DESC, notification_id DESC);
