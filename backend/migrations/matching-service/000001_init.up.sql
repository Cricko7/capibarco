CREATE TABLE animal_availability (
    animal_id TEXT PRIMARY KEY,
    owner_profile_id TEXT NOT NULL DEFAULT '',
    available BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE swipes (
    swipe_id TEXT PRIMARY KEY,
    actor_id TEXT NOT NULL,
    actor_is_guest BOOLEAN NOT NULL DEFAULT FALSE,
    animal_id TEXT NOT NULL,
    owner_profile_id TEXT NOT NULL,
    direction SMALLINT NOT NULL,
    feed_card_id TEXT NOT NULL DEFAULT '',
    feed_session_id TEXT NOT NULL DEFAULT '',
    swiped_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT swipes_direction_check CHECK (direction IN (1, 2))
);

CREATE UNIQUE INDEX swipes_actor_animal_key ON swipes (actor_id, animal_id);
CREATE INDEX swipes_actor_time_idx ON swipes (actor_id, swiped_at DESC, swipe_id DESC);
CREATE INDEX swipes_animal_idx ON swipes (animal_id);

CREATE TABLE matches (
    match_id TEXT PRIMARY KEY,
    animal_id TEXT NOT NULL,
    adopter_profile_id TEXT NOT NULL,
    owner_profile_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL DEFAULT '',
    status SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT matches_status_check CHECK (status IN (1, 2, 3))
);

CREATE UNIQUE INDEX matches_active_adopter_animal_key ON matches (animal_id, adopter_profile_id) WHERE status = 1;
CREATE INDEX matches_adopter_time_idx ON matches (adopter_profile_id, created_at DESC, match_id DESC);
CREATE INDEX matches_owner_time_idx ON matches (owner_profile_id, created_at DESC, match_id DESC);
CREATE INDEX matches_animal_status_idx ON matches (animal_id, status);

CREATE TABLE idempotency_keys (
    key TEXT PRIMARY KEY,
    actor_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    swipe_id TEXT NOT NULL REFERENCES swipes (swipe_id) ON DELETE RESTRICT,
    match_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idempotency_actor_idx ON idempotency_keys (actor_id, created_at DESC);

CREATE TABLE outbox_events (
    event_id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    partition_key TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload BYTEA NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    locked_until TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX outbox_events_unpublished_idx ON outbox_events (created_at)
WHERE published_at IS NULL;

CREATE TABLE processed_events (
    topic TEXT NOT NULL,
    partition_id INTEGER NOT NULL,
    message_offset BIGINT NOT NULL,
    event_id TEXT NOT NULL DEFAULT '',
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (topic, partition_id, message_offset)
);

