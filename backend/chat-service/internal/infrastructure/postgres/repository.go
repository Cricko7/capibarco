package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/petmatch/chat-service/internal/domain/chat"
)

const defaultLimit int32 = 50

// Repository persists chat state in PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a Postgres chat repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// NewPool creates and pings a pgx pool.
func NewPool(ctx context.Context, dsn string, maxOpen, maxIdle int32, maxLifetime time.Duration) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	cfg.MaxConns = maxOpen
	cfg.MinConns = maxIdle
	cfg.MaxConnLifetime = maxLifetime
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

// Ping verifies database connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	return nil
}

func (r *Repository) CreateConversation(ctx context.Context, conversation chat.Conversation) (chat.Conversation, error) {
	const q = `
INSERT INTO conversations (
    conversation_id, match_id, animal_id, adopter_profile_id, owner_profile_id,
    status, idempotency_key, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING conversation_id::text, match_id, animal_id, adopter_profile_id, owner_profile_id,
          status, idempotency_key, created_at, updated_at`

	var created chat.Conversation
	err := r.pool.QueryRow(ctx, q,
		conversation.ID,
		conversation.MatchID,
		conversation.AnimalID,
		conversation.AdopterProfileID,
		conversation.OwnerProfileID,
		conversation.Status,
		conversation.IdempotencyKey,
		conversation.CreatedAt,
		conversation.UpdatedAt,
	).Scan(
		&created.ID,
		&created.MatchID,
		&created.AnimalID,
		&created.AdopterProfileID,
		&created.OwnerProfileID,
		&created.Status,
		&created.IdempotencyKey,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return chat.Conversation{}, mapError(err)
	}
	return created, nil
}

func (r *Repository) GetConversation(ctx context.Context, id string) (chat.Conversation, error) {
	const q = `
SELECT conversation_id::text, match_id, animal_id, adopter_profile_id, owner_profile_id,
       status, idempotency_key, created_at, updated_at
FROM conversations
WHERE conversation_id = $1`
	return r.scanConversation(ctx, q, id)
}

func (r *Repository) GetConversationByIdempotencyKey(ctx context.Context, key string) (chat.Conversation, error) {
	const q = `
SELECT conversation_id::text, match_id, animal_id, adopter_profile_id, owner_profile_id,
       status, idempotency_key, created_at, updated_at
FROM conversations
WHERE idempotency_key = $1`
	return r.scanConversation(ctx, q, key)
}

func (r *Repository) ListConversations(ctx context.Context, filter chat.ListConversationsFilter) ([]chat.Conversation, string, error) {
	limit := normalizeLimit(filter.PageSize)
	offset := decodeOffset(filter.PageToken)
	statuses := make([]int16, 0, len(filter.Statuses))
	for _, status := range filter.Statuses {
		statuses = append(statuses, int16(status))
	}
	if len(statuses) == 0 {
		statuses = []int16{int16(chat.ConversationStatusActive)}
	}
	const q = `
SELECT conversation_id::text, match_id, animal_id, adopter_profile_id, owner_profile_id,
       status, idempotency_key, created_at, updated_at
FROM conversations
WHERE (adopter_profile_id = $1 OR owner_profile_id = $1)
  AND status = ANY($2)
ORDER BY updated_at DESC, conversation_id DESC
LIMIT $3 OFFSET $4`
	rows, err := r.pool.Query(ctx, q, filter.ParticipantProfileID, statuses, limit+1, offset)
	if err != nil {
		return nil, "", mapError(err)
	}
	defer rows.Close()

	conversations, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[conversationRow])
	if err != nil {
		return nil, "", mapError(err)
	}
	hasNext := int32(len(conversations)) > limit
	if hasNext {
		conversations = conversations[:limit]
	}
	result := make([]chat.Conversation, 0, len(conversations))
	for _, row := range conversations {
		result = append(result, row.toDomain())
	}
	if hasNext {
		return result, encodeOffset(offset + limit), nil
	}
	return result, "", nil
}

func (r *Repository) CreateMessage(ctx context.Context, message chat.Message) (chat.Message, error) {
	attachments, err := json.Marshal(message.Attachments)
	if err != nil {
		return chat.Message{}, fmt.Errorf("marshal attachments: %w", err)
	}
	metadata, err := json.Marshal(message.Metadata)
	if err != nil {
		return chat.Message{}, fmt.Errorf("marshal metadata: %w", err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return chat.Message{}, fmt.Errorf("begin create message tx: %w", err)
	}
	defer rollback(ctx, tx)

	const insertMessage = `
INSERT INTO messages (
    message_id, conversation_id, sender_profile_id, message_type, text,
    attachments, metadata, client_message_id, idempotency_key, sent_at, edited_at, deleted_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
RETURNING message_id::text, conversation_id::text, sender_profile_id, message_type, text,
          attachments, metadata, client_message_id, idempotency_key, sent_at, edited_at, deleted_at`
	created, err := scanMessageRow(tx.QueryRow(ctx, insertMessage,
		message.ID,
		message.ConversationID,
		message.SenderProfileID,
		message.Type,
		message.Text,
		attachments,
		metadata,
		message.ClientMessageID,
		message.IdempotencyKey,
		message.SentAt,
		message.EditedAt,
		message.DeletedAt,
	))
	if err != nil {
		return chat.Message{}, mapError(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE conversations SET updated_at = $1 WHERE conversation_id = $2`, created.SentAt, created.ConversationID); err != nil {
		return chat.Message{}, mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return chat.Message{}, fmt.Errorf("commit create message tx: %w", err)
	}
	return created, nil
}

func (r *Repository) GetMessageByIdempotencyKey(ctx context.Context, key string) (chat.Message, error) {
	const q = `
SELECT message_id::text, conversation_id::text, sender_profile_id, message_type, text,
       attachments, metadata, client_message_id, idempotency_key, sent_at, edited_at, deleted_at
FROM messages
WHERE idempotency_key = $1`
	message, err := scanMessageRow(r.pool.QueryRow(ctx, q, key))
	if err != nil {
		return chat.Message{}, mapError(err)
	}
	return message, nil
}

func (r *Repository) ListMessages(ctx context.Context, filter chat.ListMessagesFilter) ([]chat.Message, string, error) {
	limit := normalizeLimit(filter.PageSize)
	offset := decodeOffset(filter.PageToken)
	const q = `
SELECT message_id::text, conversation_id::text, sender_profile_id, message_type, text,
       attachments, metadata, client_message_id, idempotency_key, sent_at, edited_at, deleted_at
FROM messages
WHERE conversation_id = $1 AND deleted_at IS NULL
ORDER BY sent_at DESC, message_id DESC
LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, q, filter.ConversationID, limit+1, offset)
	if err != nil {
		return nil, "", mapError(err)
	}
	defer rows.Close()

	messages := make([]chat.Message, 0, limit)
	for rows.Next() {
		message, scanErr := scanMessageRows(rows)
		if scanErr != nil {
			return nil, "", scanErr
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, "", mapError(err)
	}
	hasNext := int32(len(messages)) > limit
	if hasNext {
		messages = messages[:limit]
		return messages, encodeOffset(offset + limit), nil
	}
	return messages, "", nil
}

func (r *Repository) UpdateMessage(ctx context.Context, message chat.Message) (chat.Message, error) {
	now := time.Now().UTC()
	const q = `
UPDATE messages
SET text = $2, edited_at = $3
WHERE message_id = $1 AND deleted_at IS NULL
RETURNING message_id::text, conversation_id::text, sender_profile_id, message_type, text,
          attachments, metadata, client_message_id, idempotency_key, sent_at, edited_at, deleted_at`
	updated, err := scanMessageRow(r.pool.QueryRow(ctx, q, message.ID, strings.TrimSpace(message.Text), now))
	if err != nil {
		return chat.Message{}, mapError(err)
	}
	return updated, nil
}

func (r *Repository) MarkRead(ctx context.Context, receipt chat.ReadReceipt) (chat.ReadReceipt, error) {
	const q = `
INSERT INTO read_receipts (conversation_id, reader_profile_id, up_to_message_id, read_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (conversation_id, reader_profile_id)
DO UPDATE SET up_to_message_id = EXCLUDED.up_to_message_id, read_at = EXCLUDED.read_at
RETURNING conversation_id::text, reader_profile_id, up_to_message_id::text, read_at`
	var updated chat.ReadReceipt
	if err := r.pool.QueryRow(ctx, q,
		receipt.ConversationID,
		receipt.ReaderProfileID,
		receipt.UpToMessageID,
		receipt.ReadAt,
	).Scan(&updated.ConversationID, &updated.ReaderProfileID, &updated.UpToMessageID, &updated.ReadAt); err != nil {
		return chat.ReadReceipt{}, mapError(err)
	}
	return updated, nil
}

func (r *Repository) scanConversation(ctx context.Context, query string, arg string) (chat.Conversation, error) {
	var conversation chat.Conversation
	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&conversation.ID,
		&conversation.MatchID,
		&conversation.AnimalID,
		&conversation.AdopterProfileID,
		&conversation.OwnerProfileID,
		&conversation.Status,
		&conversation.IdempotencyKey,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return chat.Conversation{}, mapError(err)
	}
	return conversation, nil
}

type conversationRow struct {
	ConversationID   string                  `db:"conversation_id"`
	MatchID          string                  `db:"match_id"`
	AnimalID         string                  `db:"animal_id"`
	AdopterProfileID string                  `db:"adopter_profile_id"`
	OwnerProfileID   string                  `db:"owner_profile_id"`
	Status           chat.ConversationStatus `db:"status"`
	IdempotencyKey   string                  `db:"idempotency_key"`
	CreatedAt        time.Time               `db:"created_at"`
	UpdatedAt        time.Time               `db:"updated_at"`
}

func (r conversationRow) toDomain() chat.Conversation {
	return chat.Conversation{
		ID:               r.ConversationID,
		MatchID:          r.MatchID,
		AnimalID:         r.AnimalID,
		AdopterProfileID: r.AdopterProfileID,
		OwnerProfileID:   r.OwnerProfileID,
		Status:           r.Status,
		IdempotencyKey:   r.IdempotencyKey,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

type scanner interface {
	Scan(dest ...any) error
}

type rowsScanner interface {
	Scan(dest ...any) error
}

func scanMessageRow(row scanner) (chat.Message, error) {
	var message chat.Message
	var attachments []byte
	var metadata []byte
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderProfileID,
		&message.Type,
		&message.Text,
		&attachments,
		&metadata,
		&message.ClientMessageID,
		&message.IdempotencyKey,
		&message.SentAt,
		&message.EditedAt,
		&message.DeletedAt,
	); err != nil {
		return chat.Message{}, err
	}
	if err := json.Unmarshal(attachments, &message.Attachments); err != nil {
		return chat.Message{}, fmt.Errorf("unmarshal attachments: %w", err)
	}
	if err := json.Unmarshal(metadata, &message.Metadata); err != nil {
		return chat.Message{}, fmt.Errorf("unmarshal metadata: %w", err)
	}
	return message, nil
}

func scanMessageRows(rows rowsScanner) (chat.Message, error) {
	message, err := scanMessageRow(rows)
	if err != nil {
		return chat.Message{}, mapError(err)
	}
	return message, nil
}

func mapError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("postgres: %w", chat.ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "22P02":
			return fmt.Errorf("postgres invalid identifier: %w", chat.ErrNotFound)
		case "23505":
			return fmt.Errorf("postgres unique violation: %w", chat.ErrMissingIdempotencyKey)
		}
	}
	return fmt.Errorf("postgres: %w", err)
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func normalizeLimit(size int32) int32 {
	if size <= 0 {
		return defaultLimit
	}
	return size
}

func encodeOffset(offset int32) string {
	payload := strconv.FormatInt(int64(offset), 10)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeOffset(token string) int32 {
	if token == "" {
		return 0
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0
	}
	offset, err := strconv.ParseInt(string(raw), 10, 32)
	if err != nil || offset < 0 {
		return 0
	}
	return int32(offset)
}
