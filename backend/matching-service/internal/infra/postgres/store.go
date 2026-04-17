package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	"github.com/petmatch/petmatch/internal/adapter/pbconv"
	app "github.com/petmatch/petmatch/internal/app/matching"
	domain "github.com/petmatch/petmatch/internal/domain/matching"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const producerName = "matching-service"

// Store implements the matching application persistence port.
type Store struct {
	pool  *pgxpool.Pool
	clock app.Clock
}

// OutboxEvent is a Kafka message persisted transactionally with domain writes.
type OutboxEvent struct {
	ID           string
	Topic        string
	PartitionKey string
	EventType    string
	Payload      []byte
	CreatedAt    time.Time
}

// NewStore creates a PostgreSQL store.
func NewStore(pool *pgxpool.Pool, clock app.Clock) *Store {
	if clock == nil {
		clock = app.SystemClock{}
	}
	return &Store{pool: pool, clock: clock}
}

// Ping verifies database readiness.
func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("%w: postgres pool is not configured", domain.ErrInvalidArgument)
	}
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// RecordSwipe records a swipe, creates a right-swipe match, and stores outbox events atomically.
func (s *Store) RecordSwipe(ctx context.Context, cmd app.RecordSwipeCommand) (app.RecordSwipeResult, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return app.RecordSwipeResult{}, fmt.Errorf("begin record swipe tx: %w", err)
	}
	defer rollback(ctx, tx)

	if result, ok, err := s.findIdempotency(ctx, tx, cmd.IdempotencyKey); err != nil {
		return app.RecordSwipeResult{}, err
	} else if ok {
		if err := tx.Commit(ctx); err != nil {
			return app.RecordSwipeResult{}, fmt.Errorf("commit idempotent record swipe tx: %w", err)
		}
		result.Idempotent = true
		return result, nil
	}

	if err := s.ensureAnimalAvailable(ctx, tx, cmd.AnimalID); err != nil {
		return app.RecordSwipeResult{}, err
	}

	now := s.clock.Now()
	swipe, err := domain.NewSwipe(uuid.NewString(), domain.RecordSwipeCommand(cmd), now)
	if err != nil {
		return app.RecordSwipeResult{}, err
	}
	if err := insertSwipe(ctx, tx, swipe); err != nil {
		if isUniqueViolation(err, "swipes_actor_animal_key") {
			return app.RecordSwipeResult{}, fmt.Errorf("%w: actor %s already swiped animal %s", domain.ErrDuplicateSwipe, swipe.ActorID, swipe.AnimalID)
		}
		return app.RecordSwipeResult{}, err
	}

	result := app.RecordSwipeResult{Swipe: swipe}
	if err := s.insertSwipeRecordedEvent(ctx, tx, swipe, cmd.IdempotencyKey); err != nil {
		return app.RecordSwipeResult{}, err
	}

	if swipe.Direction == domain.SwipeDirectionRight {
		match, err := domain.NewMatchFromSwipe(uuid.NewString(), swipe)
		if err != nil {
			return app.RecordSwipeResult{}, err
		}
		if err := insertMatch(ctx, tx, match); err != nil {
			return app.RecordSwipeResult{}, err
		}
		if err := s.insertMatchCreatedEvent(ctx, tx, match, swipe, cmd.IdempotencyKey); err != nil {
			return app.RecordSwipeResult{}, err
		}
		result.Match = &match
		result.CreatedMatch = true
	}

	matchID := ""
	if result.Match != nil {
		matchID = result.Match.ID
	}
	if err := insertIdempotency(ctx, tx, cmd.IdempotencyKey, cmd.ActorID, swipe.ID, matchID); err != nil {
		if isUniqueViolation(err, "idempotency_keys_pkey") {
			return app.RecordSwipeResult{}, fmt.Errorf("%w: idempotency key was used concurrently", domain.ErrConflict)
		}
		return app.RecordSwipeResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return app.RecordSwipeResult{}, fmt.Errorf("commit record swipe tx: %w", err)
	}
	return result, nil
}

// UpdateMatchConversation attaches a chat conversation to a match.
func (s *Store) UpdateMatchConversation(ctx context.Context, matchID string, conversationID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE matches
		SET conversation_id = $2, updated_at = now()
		WHERE match_id = $1 AND (conversation_id = '' OR conversation_id = $2)
	`, matchID, conversationID)
	if err != nil {
		return fmt.Errorf("update match conversation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: match %s", domain.ErrNotFound, matchID)
	}
	return nil
}

// GetSwipe returns a swipe by id.
func (s *Store) GetSwipe(ctx context.Context, swipeID string) (domain.Swipe, error) {
	row := s.pool.QueryRow(ctx, selectSwipeSQL+` WHERE swipe_id = $1`, swipeID)
	swipe, err := scanSwipe(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Swipe{}, fmt.Errorf("%w: swipe %s", domain.ErrNotFound, swipeID)
		}
		return domain.Swipe{}, fmt.Errorf("scan swipe: %w", err)
	}
	return swipe, nil
}

// ListSwipes returns a page of swipes for an actor.
func (s *Store) ListSwipes(ctx context.Context, query app.ListSwipesQuery) (app.ListSwipesResult, error) {
	page := normalizePage(query.Page)
	cursor, err := decodeCursor(page.PageToken)
	if err != nil {
		return app.ListSwipesResult{}, err
	}

	args := []any{query.ActorID, page.PageSize + 1}
	filter := "WHERE actor_id = $1"
	if len(query.Directions) > 0 {
		args = append(args, directionsToInt16(query.Directions))
		filter += fmt.Sprintf(" AND direction = ANY($%d)", len(args))
	}
	if cursor.ID != "" {
		args = append(args, cursor.At, cursor.ID)
		filter += fmt.Sprintf(" AND (swiped_at, swipe_id) < ($%d, $%d)", len(args)-1, len(args))
	}

	rows, err := s.pool.Query(ctx, selectSwipeSQL+` `+filter+` ORDER BY swiped_at DESC, swipe_id DESC LIMIT $2`, args...)
	if err != nil {
		return app.ListSwipesResult{}, fmt.Errorf("query swipes: %w", err)
	}
	defer rows.Close()

	swipes := make([]domain.Swipe, 0, page.PageSize)
	for rows.Next() {
		swipe, err := scanSwipe(rows)
		if err != nil {
			return app.ListSwipesResult{}, fmt.Errorf("scan swipe row: %w", err)
		}
		swipes = append(swipes, swipe)
	}
	if err := rows.Err(); err != nil {
		return app.ListSwipesResult{}, fmt.Errorf("iterate swipes: %w", err)
	}

	next := ""
	if len(swipes) > int(page.PageSize) {
		last := swipes[page.PageSize-1]
		next = encodeCursor(last.SwipedAt, last.ID)
		swipes = swipes[:page.PageSize]
	}
	return app.ListSwipesResult{Swipes: swipes, Page: app.PageResponse{NextPageToken: next}}, nil
}

// GetMatch returns a match by id.
func (s *Store) GetMatch(ctx context.Context, matchID string) (domain.Match, error) {
	row := s.pool.QueryRow(ctx, selectMatchSQL+` WHERE match_id = $1`, matchID)
	match, err := scanMatch(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Match{}, fmt.Errorf("%w: match %s", domain.ErrNotFound, matchID)
		}
		return domain.Match{}, fmt.Errorf("scan match: %w", err)
	}
	return match, nil
}

// ListMatches returns matches for a participant.
func (s *Store) ListMatches(ctx context.Context, query app.ListMatchesQuery) (app.ListMatchesResult, error) {
	page := normalizePage(query.Page)
	cursor, err := decodeCursor(page.PageToken)
	if err != nil {
		return app.ListMatchesResult{}, err
	}

	args := []any{query.ParticipantProfileID, page.PageSize + 1}
	filter := "WHERE (adopter_profile_id = $1 OR owner_profile_id = $1)"
	if len(query.Statuses) > 0 {
		args = append(args, statusesToInt16(query.Statuses))
		filter += fmt.Sprintf(" AND status = ANY($%d)", len(args))
	}
	if cursor.ID != "" {
		args = append(args, cursor.At, cursor.ID)
		filter += fmt.Sprintf(" AND (created_at, match_id) < ($%d, $%d)", len(args)-1, len(args))
	}

	rows, err := s.pool.Query(ctx, selectMatchSQL+` `+filter+` ORDER BY created_at DESC, match_id DESC LIMIT $2`, args...)
	if err != nil {
		return app.ListMatchesResult{}, fmt.Errorf("query matches: %w", err)
	}
	defer rows.Close()

	matches := make([]domain.Match, 0, page.PageSize)
	for rows.Next() {
		match, err := scanMatch(rows)
		if err != nil {
			return app.ListMatchesResult{}, fmt.Errorf("scan match row: %w", err)
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return app.ListMatchesResult{}, fmt.Errorf("iterate matches: %w", err)
	}

	next := ""
	if len(matches) > int(page.PageSize) {
		last := matches[page.PageSize-1]
		next = encodeCursor(last.CreatedAt, last.ID)
		matches = matches[:page.PageSize]
	}
	return app.ListMatchesResult{Matches: matches, Page: app.PageResponse{NextPageToken: next}}, nil
}

// ArchiveMatchesByAnimal archives active matches for an animal and stores archive outbox events.
func (s *Store) ArchiveMatchesByAnimal(ctx context.Context, animalID string, reason string) ([]domain.Match, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin archive matches tx: %w", err)
	}
	defer rollback(ctx, tx)

	rows, err := tx.Query(ctx, selectMatchSQL+`
		WHERE animal_id = $1 AND status = $2
		FOR UPDATE
	`, animalID, int16(domain.MatchStatusActive))
	if err != nil {
		return nil, fmt.Errorf("select active animal matches: %w", err)
	}
	matches := make([]domain.Match, 0)
	for rows.Next() {
		match, err := scanMatch(rows)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan active animal match: %w", err)
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate active animal matches: %w", err)
	}
	rows.Close()

	archived := make([]domain.Match, 0, len(matches))
	for _, match := range matches {
		next, err := match.Archive(s.clock.Now())
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE matches SET status = $2, updated_at = $3 WHERE match_id = $1
		`, next.ID, int16(next.Status), next.UpdatedAt); err != nil {
			return nil, fmt.Errorf("archive match %s: %w", next.ID, err)
		}
		if err := s.insertMatchArchivedEvent(ctx, tx, next.ID, reason); err != nil {
			return nil, err
		}
		archived = append(archived, next)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit archive matches tx: %w", err)
	}
	return archived, nil
}

// SetAnimalAvailability records whether new matches may be created for an animal.
func (s *Store) SetAnimalAvailability(ctx context.Context, animalID string, ownerProfileID string, available bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO animal_availability (animal_id, owner_profile_id, available, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (animal_id)
		DO UPDATE SET owner_profile_id = EXCLUDED.owner_profile_id, available = EXCLUDED.available, updated_at = now()
	`, animalID, ownerProfileID, available)
	if err != nil {
		return fmt.Errorf("upsert animal availability: %w", err)
	}
	return nil
}

// FetchOutbox claims unpublished outbox events for delivery.
func (s *Store) FetchOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		WITH picked AS (
			SELECT event_id
			FROM outbox_events
			WHERE published_at IS NULL
				AND (locked_until IS NULL OR locked_until < now())
				AND attempts < 20
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE outbox_events e
		SET locked_until = now() + interval '30 seconds', attempts = attempts + 1
		FROM picked
		WHERE e.event_id = picked.event_id
		RETURNING e.event_id, e.topic, e.partition_key, e.event_type, e.payload, e.created_at
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0, limit)
	for rows.Next() {
		var event OutboxEvent
		if err := rows.Scan(&event.ID, &event.Topic, &event.PartitionKey, &event.EventType, &event.Payload, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}
	return events, nil
}

// MarkOutboxPublished marks one outbox event as delivered.
func (s *Store) MarkOutboxPublished(ctx context.Context, eventID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = now(), locked_until = NULL, last_error = ''
		WHERE event_id = $1
	`, eventID)
	if err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

// MarkOutboxFailed records a delivery failure.
func (s *Store) MarkOutboxFailed(ctx context.Context, eventID string, cause error) error {
	msg := ""
	if cause != nil {
		msg = cause.Error()
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events
		SET locked_until = NULL, last_error = $2
		WHERE event_id = $1
	`, eventID, msg)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}

func (s *Store) findIdempotency(ctx context.Context, tx pgx.Tx, key string) (app.RecordSwipeResult, bool, error) {
	var swipeID string
	var matchID sql.NullString
	err := tx.QueryRow(ctx, `
		SELECT swipe_id, NULLIF(match_id, '') FROM idempotency_keys WHERE key = $1
	`, key).Scan(&swipeID, &matchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return app.RecordSwipeResult{}, false, nil
		}
		return app.RecordSwipeResult{}, false, fmt.Errorf("select idempotency key: %w", err)
	}

	swipe, err := getSwipeTx(ctx, tx, swipeID)
	if err != nil {
		return app.RecordSwipeResult{}, false, err
	}
	result := app.RecordSwipeResult{Swipe: swipe, Idempotent: true}
	if matchID.Valid {
		match, err := getMatchTx(ctx, tx, matchID.String)
		if err != nil {
			return app.RecordSwipeResult{}, false, err
		}
		result.Match = &match
		if match.ConversationID != "" {
			result.ChatCreated = true
			result.ConversationID = match.ConversationID
		}
	}
	return result, true, nil
}

func (s *Store) ensureAnimalAvailable(ctx context.Context, tx pgx.Tx, animalID string) error {
	var available bool
	err := tx.QueryRow(ctx, `SELECT available FROM animal_availability WHERE animal_id = $1`, animalID).Scan(&available)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("select animal availability: %w", err)
	}
	if !available {
		return fmt.Errorf("%w: animal %s", domain.ErrUnavailableAnimal, animalID)
	}
	return nil
}

func insertSwipe(ctx context.Context, tx pgx.Tx, swipe domain.Swipe) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO swipes (
			swipe_id, actor_id, actor_is_guest, animal_id, owner_profile_id,
			direction, feed_card_id, feed_session_id, swiped_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, swipe.ID, swipe.ActorID, swipe.ActorIsGuest, swipe.AnimalID, swipe.OwnerProfileID,
		int16(swipe.Direction), swipe.FeedCardID, swipe.FeedSessionID, swipe.SwipedAt)
	if err != nil {
		return fmt.Errorf("insert swipe: %w", err)
	}
	return nil
}

func insertMatch(ctx context.Context, tx pgx.Tx, match domain.Match) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO matches (
			match_id, animal_id, adopter_profile_id, owner_profile_id,
			conversation_id, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, match.ID, match.AnimalID, match.AdopterProfileID, match.OwnerProfileID,
		match.ConversationID, int16(match.Status), match.CreatedAt, match.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert match: %w", err)
	}
	return nil
}

func insertIdempotency(ctx context.Context, tx pgx.Tx, key string, actorID string, swipeID string, matchID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys (key, actor_id, operation, swipe_id, match_id, created_at)
		VALUES ($1, $2, 'record_swipe', $3, $4, now())
	`, key, actorID, swipeID, matchID)
	if err != nil {
		return fmt.Errorf("insert idempotency key: %w", err)
	}
	return nil
}

func (s *Store) insertSwipeRecordedEvent(ctx context.Context, tx pgx.Tx, swipe domain.Swipe, idempotencyKey string) error {
	event := &matchingv1.SwipeRecordedEvent{
		Envelope: s.envelope(ctx, "matching.swipe_recorded", swipe.ActorID, idempotencyKey),
		Swipe:    pbconv.SwipeToProto(swipe),
	}
	payload, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal swipe recorded event: %w", err)
	}
	return insertOutbox(ctx, tx, event.Envelope.EventId, "matching.swipe_recorded", swipe.ActorID, event.Envelope.EventType, payload, s.clock.Now())
}

func (s *Store) insertMatchCreatedEvent(ctx context.Context, tx pgx.Tx, match domain.Match, swipe domain.Swipe, idempotencyKey string) error {
	event := &matchingv1.MatchCreatedEvent{
		Envelope: s.envelope(ctx, "matching.match_created", match.ID, idempotencyKey),
		Match:    pbconv.MatchToProto(match),
		Swipe:    pbconv.SwipeToProto(swipe),
	}
	payload, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal match created event: %w", err)
	}
	return insertOutbox(ctx, tx, event.Envelope.EventId, "matching.match_created", match.ID, event.Envelope.EventType, payload, s.clock.Now())
}

func (s *Store) insertMatchArchivedEvent(ctx context.Context, tx pgx.Tx, matchID string, reason string) error {
	event := &matchingv1.MatchArchivedEvent{
		Envelope: s.envelope(ctx, "matching.match_archived", matchID, matchID),
		MatchId:  matchID,
		Reason:   reason,
	}
	payload, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal match archived event: %w", err)
	}
	return insertOutbox(ctx, tx, event.Envelope.EventId, "matching.match_archived", matchID, event.Envelope.EventType, payload, s.clock.Now())
}

func (s *Store) envelope(ctx context.Context, eventType string, partitionKey string, idempotencyKey string) *commonv1.EventEnvelope {
	now := s.clock.Now()
	traceID := requestid.From(ctx)
	return &commonv1.EventEnvelope{
		EventId:        uuid.NewString(),
		EventType:      eventType,
		SchemaVersion:  "v1",
		Producer:       producerName,
		OccurredAt:     timestamppb.New(now),
		TraceId:        traceID,
		CorrelationId:  traceID,
		IdempotencyKey: idempotencyKey,
		PartitionKey:   partitionKey,
	}
}

func insertOutbox(ctx context.Context, tx pgx.Tx, eventID string, topic string, partitionKey string, eventType string, payload []byte, createdAt time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (event_id, topic, partition_key, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, eventID, topic, partitionKey, eventType, payload, createdAt)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

const selectSwipeSQL = `
	SELECT swipe_id, actor_id, actor_is_guest, animal_id, owner_profile_id,
		direction, feed_card_id, feed_session_id, swiped_at
	FROM swipes`

const selectMatchSQL = `
	SELECT match_id, animal_id, adopter_profile_id, owner_profile_id,
		conversation_id, status, created_at, updated_at
	FROM matches`

type scanner interface {
	Scan(dest ...any) error
}

func scanSwipe(row scanner) (domain.Swipe, error) {
	var swipe domain.Swipe
	var direction int16
	if err := row.Scan(
		&swipe.ID,
		&swipe.ActorID,
		&swipe.ActorIsGuest,
		&swipe.AnimalID,
		&swipe.OwnerProfileID,
		&direction,
		&swipe.FeedCardID,
		&swipe.FeedSessionID,
		&swipe.SwipedAt,
	); err != nil {
		return domain.Swipe{}, err
	}
	swipe.Direction = domain.SwipeDirection(direction)
	return swipe, nil
}

func scanMatch(row scanner) (domain.Match, error) {
	var match domain.Match
	var status int16
	if err := row.Scan(
		&match.ID,
		&match.AnimalID,
		&match.AdopterProfileID,
		&match.OwnerProfileID,
		&match.ConversationID,
		&status,
		&match.CreatedAt,
		&match.UpdatedAt,
	); err != nil {
		return domain.Match{}, err
	}
	match.Status = domain.MatchStatus(status)
	return match, nil
}

func getSwipeTx(ctx context.Context, tx pgx.Tx, swipeID string) (domain.Swipe, error) {
	swipe, err := scanSwipe(tx.QueryRow(ctx, selectSwipeSQL+` WHERE swipe_id = $1`, swipeID))
	if err != nil {
		return domain.Swipe{}, fmt.Errorf("select swipe %s: %w", swipeID, err)
	}
	return swipe, nil
}

func getMatchTx(ctx context.Context, tx pgx.Tx, matchID string) (domain.Match, error) {
	match, err := scanMatch(tx.QueryRow(ctx, selectMatchSQL+` WHERE match_id = $1`, matchID))
	if err != nil {
		return domain.Match{}, fmt.Errorf("select match %s: %w", matchID, err)
	}
	return match, nil
}

func rollback(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "23505" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

func directionsToInt16(directions []domain.SwipeDirection) []int16 {
	values := make([]int16, 0, len(directions))
	for _, direction := range directions {
		values = append(values, int16(direction))
	}
	return values
}

func statusesToInt16(statuses []domain.MatchStatus) []int16 {
	values := make([]int16, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, int16(status))
	}
	return values
}

func normalizePage(page app.PageRequest) app.PageRequest {
	const (
		defaultPageSize = int32(50)
		maxPageSize     = int32(100)
	)
	if page.PageSize <= 0 {
		page.PageSize = defaultPageSize
	}
	if page.PageSize > maxPageSize {
		page.PageSize = maxPageSize
	}
	return page
}

type cursorToken struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func encodeCursor(at time.Time, id string) string {
	payload, _ := json.Marshal(cursorToken{At: at.UTC(), ID: id})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeCursor(token string) (cursorToken, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return cursorToken{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return cursorToken{}, fmt.Errorf("%w: invalid page token", domain.ErrInvalidArgument)
	}
	var cursor cursorToken
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return cursorToken{}, fmt.Errorf("%w: invalid page token", domain.ErrInvalidArgument)
	}
	if cursor.ID == "" || cursor.At.IsZero() {
		return cursorToken{}, fmt.Errorf("%w: invalid page token", domain.ErrInvalidArgument)
	}
	return cursor, nil
}
