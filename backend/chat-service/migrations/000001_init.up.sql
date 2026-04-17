CREATE TABLE conversations (
    conversation_id UUID PRIMARY KEY,
    match_id TEXT NOT NULL,
    animal_id TEXT NOT NULL,
    adopter_profile_id TEXT NOT NULL,
    owner_profile_id TEXT NOT NULL,
    status SMALLINT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK (adopter_profile_id <> owner_profile_id)
);

CREATE INDEX conversations_participant_updated_idx
    ON conversations (updated_at DESC)
    INCLUDE (adopter_profile_id, owner_profile_id, status);

CREATE TABLE messages (
    message_id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(conversation_id) ON DELETE CASCADE,
    sender_profile_id TEXT NOT NULL,
    message_type SMALLINT NOT NULL,
    text TEXT NOT NULL DEFAULT '',
    attachments JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    client_message_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    sent_at TIMESTAMPTZ NOT NULL,
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    UNIQUE (conversation_id, client_message_id)
);

CREATE INDEX messages_conversation_sent_idx ON messages (conversation_id, sent_at DESC, message_id DESC);

CREATE TABLE read_receipts (
    conversation_id UUID NOT NULL REFERENCES conversations(conversation_id) ON DELETE CASCADE,
    reader_profile_id TEXT NOT NULL,
    up_to_message_id UUID NOT NULL REFERENCES messages(message_id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (conversation_id, reader_profile_id)
);
