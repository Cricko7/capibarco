CREATE TABLE IF NOT EXISTS raw_events (
    event_id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS aggregated_metrics (
    profile_id TEXT NOT NULL,
    bucket_size TEXT NOT NULL,
    bucket_at TIMESTAMPTZ NOT NULL,
    event_type TEXT NOT NULL,
    value BIGINT NOT NULL,
    PRIMARY KEY (profile_id, bucket_size, bucket_at, event_type)
);

CREATE INDEX IF NOT EXISTS idx_aggregated_metrics_bucket ON aggregated_metrics(bucket_size, bucket_at DESC);
