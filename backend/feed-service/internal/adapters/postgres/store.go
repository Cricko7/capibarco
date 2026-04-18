// Package postgres provides PostgreSQL adapters for feed-service projections.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	"github.com/petmatch/petmatch/internal/feed"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Store is a PostgreSQL feed projection and repository.
type Store struct {
	db *sql.DB
}

// NewStore creates a PostgreSQL store backed by db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// EnsureSchema creates the minimal feed-service tables when they do not exist.
func (s *Store) EnsureSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("postgres store requires db")
	}
	return s.applyMigrations(ctx)
}

// ListCandidates returns candidates matching the basic animal filter.
func (s *Store) ListCandidates(ctx context.Context, filter *animalv1.AnimalFilter) ([]feed.Candidate, error) {
	query, args := listCandidatesSQL(filter)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	defer rows.Close()

	var candidates []feed.Candidate
	for rows.Next() {
		var row candidateRow
		if err := rows.Scan(
			&row.AnimalID,
			&row.AnimalProto,
			&row.OwnerDisplayName,
			&row.OwnerAverageRating,
			pq.Array(&row.RankingReasons),
			&row.ScoreComponents,
			&row.DistanceKM,
			&row.OwnerHidden,
			&row.OwnerBlocked,
		); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		candidate, err := row.candidate()
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read candidates: %w", err)
	}
	return candidates, nil
}

// SaveServedCards stores generated cards for later GetFeedCard and ExplainRanking calls.
func (s *Store) SaveServedCards(ctx context.Context, cards []*feedv1.FeedCard, scores map[string]map[string]float64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save served cards: %w", err)
	}
	defer rollback(tx)

	for _, card := range cards {
		cardBytes, err := proto.Marshal(card)
		if err != nil {
			return fmt.Errorf("marshal feed card %q: %w", card.GetFeedCardId(), err)
		}
		scoreBytes, err := json.Marshal(scores[card.GetFeedCardId()])
		if err != nil {
			return fmt.Errorf("marshal card scores %q: %w", card.GetFeedCardId(), err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO feed_served_cards (feed_card_id, animal_id, feed_card_proto, score_components)
VALUES ($1, $2, $3, $4)
ON CONFLICT (feed_card_id) DO UPDATE SET
	animal_id = EXCLUDED.animal_id,
	feed_card_proto = EXCLUDED.feed_card_proto,
	score_components = EXCLUDED.score_components`,
			card.GetFeedCardId(),
			card.GetAnimal().GetAnimalId(),
			cardBytes,
			scoreBytes,
		)
		if err != nil {
			return fmt.Errorf("save served card %q: %w", card.GetFeedCardId(), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save served cards: %w", err)
	}
	return nil
}

// GetServedCard returns a previously served card and its ranking scores.
func (s *Store) GetServedCard(ctx context.Context, feedCardID string) (*feedv1.FeedCard, map[string]float64, error) {
	var cardBytes []byte
	var scoreBytes []byte
	err := s.db.QueryRowContext(ctx, `
SELECT feed_card_proto, score_components
FROM feed_served_cards
WHERE feed_card_id = $1`, feedCardID).Scan(&cardBytes, &scoreBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, feed.ErrNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("get served card %q: %w", feedCardID, err)
	}

	card := &feedv1.FeedCard{}
	if err := proto.Unmarshal(cardBytes, card); err != nil {
		return nil, nil, fmt.Errorf("unmarshal served card %q: %w", feedCardID, err)
	}
	scores, err := decodeScores(scoreBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("decode served card scores %q: %w", feedCardID, err)
	}
	return card, scores, nil
}

// RecordCardOpen stores an idempotent card-open result.
func (s *Store) RecordCardOpen(ctx context.Context, key string, cardOpenID string, openedAt time.Time) (*feedv1.RecordCardOpenResponse, bool, error) {
	var storedID string
	var storedAt time.Time
	err := s.db.QueryRowContext(ctx, `
INSERT INTO feed_card_opens (idempotency_key, card_open_id, opened_at)
VALUES ($1, $2, $3)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING card_open_id, opened_at`, key, cardOpenID, openedAt).Scan(&storedID, &storedAt)
	if err == nil {
		return &feedv1.RecordCardOpenResponse{CardOpenId: storedID, OpenedAt: timestamppb.New(storedAt)}, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("record card open %q: %w", key, err)
	}
	err = s.db.QueryRowContext(ctx, `
SELECT card_open_id, opened_at
FROM feed_card_opens
WHERE idempotency_key = $1`, key).Scan(&storedID, &storedAt)
	if err != nil {
		return nil, false, fmt.Errorf("load existing card open %q: %w", key, err)
	}
	return &feedv1.RecordCardOpenResponse{CardOpenId: storedID, OpenedAt: timestamppb.New(storedAt)}, false, nil
}

// UpsertAnimalProfile adds or updates an animal in the feed projection.
func (s *Store) UpsertAnimalProfile(ctx context.Context, animal *animalv1.AnimalProfile) error {
	if animal == nil || animal.AnimalId == "" {
		return nil
	}
	candidate := feed.Candidate{
		Animal:           proto.Clone(animal).(*animalv1.AnimalProfile),
		OwnerDisplayName: animal.OwnerProfileId,
		RankingReasons:   []string{"recently published"},
		ScoreComponents:  map[string]float64{"freshness": 1},
	}
	row, err := newCandidateRow(candidate)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, upsertCandidateSQL, row.args()...)
	if err != nil {
		return fmt.Errorf("upsert animal profile %q: %w", animal.AnimalId, err)
	}
	return nil
}

// ArchiveAnimal removes an animal from active feed candidates and cached served cards.
func (s *Store) ArchiveAnimal(ctx context.Context, event feed.AnimalArchivedPayload) error {
	if event.AnimalID == "" {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin archive animal %q: %w", event.AnimalID, err)
	}
	defer rollback(tx)
	if _, err := tx.ExecContext(ctx, `DELETE FROM feed_candidates WHERE animal_id = $1`, event.AnimalID); err != nil {
		return fmt.Errorf("delete candidate %q: %w", event.AnimalID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM feed_served_cards WHERE animal_id = $1`, event.AnimalID); err != nil {
		return fmt.Errorf("delete served cards for animal %q: %w", event.AnimalID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit archive animal %q: %w", event.AnimalID, err)
	}
	return nil
}

// UpdateAnimalStatus updates feed eligibility after an upstream status change.
func (s *Store) UpdateAnimalStatus(ctx context.Context, event *animalv1.AnimalStatusChangedEvent) error {
	if event == nil || event.AnimalId == "" {
		return nil
	}
	if event.NewStatus != animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE {
		return s.ArchiveAnimal(ctx, feed.AnimalArchivedPayload{AnimalID: event.AnimalId})
	}
	_, err := s.db.ExecContext(ctx, `UPDATE feed_candidates SET status = $2 WHERE animal_id = $1`, event.AnimalId, int32(event.NewStatus))
	if err != nil {
		return fmt.Errorf("update animal status %q: %w", event.AnimalId, err)
	}
	return nil
}

// RecordSwipe stores a user's swiped animal id for feed suppression.
func (s *Store) RecordSwipe(ctx context.Context, swipe *matchingv1.Swipe) error {
	if swipe == nil || swipe.ActorId == "" || swipe.AnimalId == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO feed_seen_animals (actor_id, animal_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING`, swipe.ActorId, swipe.AnimalId)
	if err != nil {
		return fmt.Errorf("record swipe %q/%q: %w", swipe.ActorId, swipe.AnimalId, err)
	}
	return nil
}

// ActivateBoost updates boost priority for an animal.
func (s *Store) ActivateBoost(ctx context.Context, boost *billingv1.Boost) error {
	if boost == nil || boost.AnimalId == "" {
		return nil
	}
	candidate, err := s.getCandidate(ctx, boost.AnimalId)
	if errors.Is(err, feed.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	candidate.Animal.Boosted = boost.Active
	candidate.Animal.BoostExpiresAt = boost.ExpiresAt
	if candidate.ScoreComponents == nil {
		candidate.ScoreComponents = map[string]float64{}
	}
	if boost.Active {
		candidate.ScoreComponents["boost"] = 1
		candidate.RankingReasons = appendReason(candidate.RankingReasons, "active boost")
	} else {
		delete(candidate.ScoreComponents, "boost")
	}
	return s.upsertCandidate(ctx, candidate)
}

// GrantEntitlement updates the advanced-filter entitlement cache.
func (s *Store) GrantEntitlement(ctx context.Context, entitlement *billingv1.Entitlement) error {
	if entitlement == nil || entitlement.OwnerProfileId == "" {
		return nil
	}
	if entitlement.Type != billingv1.EntitlementType_ENTITLEMENT_TYPE_ADVANCED_FILTERS {
		return nil
	}
	if entitlement.Active {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO feed_advanced_filter_entitlements (owner_profile_id)
VALUES ($1)
ON CONFLICT DO NOTHING`, entitlement.OwnerProfileId)
		if err != nil {
			return fmt.Errorf("grant entitlement %q: %w", entitlement.OwnerProfileId, err)
		}
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM feed_advanced_filter_entitlements WHERE owner_profile_id = $1`, entitlement.OwnerProfileId)
	if err != nil {
		return fmt.Errorf("revoke entitlement %q: %w", entitlement.OwnerProfileId, err)
	}
	return nil
}

// UpdateAnimalStats updates ranking score components from aggregated analytics.
func (s *Store) UpdateAnimalStats(ctx context.Context, stats *analyticsv1.AnimalStats) error {
	if stats == nil || stats.AnimalId == "" {
		return nil
	}
	candidate, err := s.getCandidate(ctx, stats.AnimalId)
	if errors.Is(err, feed.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if candidate.ScoreComponents == nil {
		candidate.ScoreComponents = map[string]float64{}
	}
	candidate.ScoreComponents["ctr"] = stats.Ctr
	candidate.RankingReasons = appendReason(candidate.RankingReasons, "analytics engagement")
	return s.upsertCandidate(ctx, candidate)
}

// SeenAnimalIDs returns swiped animal ids stored by consumed matching events.
func (s *Store) SeenAnimalIDs(ctx context.Context, principal *commonv1.Principal) (map[string]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT animal_id FROM feed_seen_animals WHERE actor_id = $1`, principal.GetActorId())
	if err != nil {
		return nil, fmt.Errorf("load seen animals: %w", err)
	}
	defer rows.Close()

	seen := map[string]struct{}{}
	for rows.Next() {
		var animalID string
		if err := rows.Scan(&animalID); err != nil {
			return nil, fmt.Errorf("scan seen animal: %w", err)
		}
		seen[animalID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read seen animals: %w", err)
	}
	return seen, nil
}

// CanUsePaidAdvancedFilters checks the entitlement cache built from billing events.
func (s *Store) CanUsePaidAdvancedFilters(ctx context.Context, principal *commonv1.Principal) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1 FROM feed_advanced_filter_entitlements WHERE owner_profile_id = $1
)`, principal.GetActorId()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check advanced filter entitlement: %w", err)
	}
	return exists, nil
}

func (s *Store) getCandidate(ctx context.Context, animalID string) (feed.Candidate, error) {
	var row candidateRow
	err := s.db.QueryRowContext(ctx, `
SELECT animal_id, animal_proto, owner_display_name, owner_average_rating, ranking_reasons,
	score_components, distance_km, owner_hidden, owner_blocked
FROM feed_candidates
WHERE animal_id = $1`, animalID).Scan(
		&row.AnimalID,
		&row.AnimalProto,
		&row.OwnerDisplayName,
		&row.OwnerAverageRating,
		pq.Array(&row.RankingReasons),
		&row.ScoreComponents,
		&row.DistanceKM,
		&row.OwnerHidden,
		&row.OwnerBlocked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.Candidate{}, feed.ErrNotFound
	}
	if err != nil {
		return feed.Candidate{}, fmt.Errorf("get candidate %q: %w", animalID, err)
	}
	return row.candidate()
}

func (s *Store) upsertCandidate(ctx context.Context, candidate feed.Candidate) error {
	row, err := newCandidateRow(candidate)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, upsertCandidateSQL, row.args()...); err != nil {
		return fmt.Errorf("upsert candidate %q: %w", row.AnimalID, err)
	}
	return nil
}

type candidateRow struct {
	AnimalID           string
	AnimalProto        []byte
	OwnerDisplayName   string
	OwnerAverageRating float64
	RankingReasons     []string
	ScoreComponents    []byte
	DistanceKM         int32
	OwnerHidden        bool
	OwnerBlocked       bool
	Species            int32
	Status             int32
	City               string
	Boosted            bool
	OwnerProfileID     string
	Vaccinated         bool
	Sterilized         bool
}

func newCandidateRow(candidate feed.Candidate) (candidateRow, error) {
	if candidate.Animal == nil || candidate.Animal.AnimalId == "" {
		return candidateRow{}, errors.New("candidate animal_id is required")
	}
	animalBytes, err := proto.Marshal(candidate.Animal)
	if err != nil {
		return candidateRow{}, fmt.Errorf("marshal animal %q: %w", candidate.Animal.AnimalId, err)
	}
	scoreBytes, err := json.Marshal(candidate.ScoreComponents)
	if err != nil {
		return candidateRow{}, fmt.Errorf("marshal scores for animal %q: %w", candidate.Animal.AnimalId, err)
	}
	return candidateRow{
		AnimalID:           candidate.Animal.AnimalId,
		AnimalProto:        animalBytes,
		OwnerDisplayName:   candidate.OwnerDisplayName,
		OwnerAverageRating: candidate.OwnerAverageRating,
		RankingReasons:     append([]string(nil), candidate.RankingReasons...),
		ScoreComponents:    scoreBytes,
		DistanceKM:         candidate.DistanceKM,
		OwnerHidden:        candidate.OwnerHidden,
		OwnerBlocked:       candidate.OwnerBlocked,
		Species:            int32(candidate.Animal.Species),
		Status:             int32(candidate.Animal.Status),
		City:               candidate.Animal.GetLocation().GetCity(),
		Boosted:            candidate.Animal.Boosted,
		OwnerProfileID:     candidate.Animal.OwnerProfileId,
		Vaccinated:         candidate.Animal.Vaccinated,
		Sterilized:         candidate.Animal.Sterilized,
	}, nil
}

func (r candidateRow) candidate() (feed.Candidate, error) {
	animal := &animalv1.AnimalProfile{}
	if err := proto.Unmarshal(r.AnimalProto, animal); err != nil {
		return feed.Candidate{}, fmt.Errorf("unmarshal animal %q: %w", r.AnimalID, err)
	}
	scores, err := decodeScores(r.ScoreComponents)
	if err != nil {
		return feed.Candidate{}, fmt.Errorf("decode scores for animal %q: %w", r.AnimalID, err)
	}
	return feed.Candidate{
		Animal:             animal,
		OwnerDisplayName:   r.OwnerDisplayName,
		OwnerAverageRating: r.OwnerAverageRating,
		RankingReasons:     append([]string(nil), r.RankingReasons...),
		ScoreComponents:    scores,
		DistanceKM:         r.DistanceKM,
		OwnerHidden:        r.OwnerHidden,
		OwnerBlocked:       r.OwnerBlocked,
	}, nil
}

func (r candidateRow) args() []any {
	return []any{
		r.AnimalID,
		r.AnimalProto,
		r.OwnerDisplayName,
		r.OwnerAverageRating,
		pq.Array(r.RankingReasons),
		r.ScoreComponents,
		r.DistanceKM,
		r.OwnerHidden,
		r.OwnerBlocked,
		r.Species,
		r.Status,
		r.City,
		r.Boosted,
		r.OwnerProfileID,
		r.Vaccinated,
		r.Sterilized,
	}
}

func listCandidatesSQL(filter *animalv1.AnimalFilter) (string, []any) {
	query := `
SELECT animal_id, animal_proto, owner_display_name, owner_average_rating, ranking_reasons,
	score_components, distance_km, owner_hidden, owner_blocked
FROM feed_candidates`
	var predicates []string
	var args []any
	add := func(predicate string, value any) {
		args = append(args, value)
		predicates = append(predicates, fmt.Sprintf(predicate, len(args)))
	}
	if filter != nil {
		if len(filter.Species) > 0 {
			add("species = ANY($%d)", enumSlice(filter.Species))
		}
		if len(filter.Statuses) > 0 {
			add("status = ANY($%d)", enumSlice(filter.Statuses))
		}
		if filter.City != nil {
			add("city = $%d", filter.GetCity())
		}
		if filter.RadiusKm != nil {
			add("distance_km <= $%d", filter.GetRadiusKm())
		}
		if filter.BoostedOnly != nil && filter.GetBoostedOnly() {
			add("boosted = $%d", true)
		}
		if filter.OwnerProfileId != nil {
			add("owner_profile_id = $%d", filter.GetOwnerProfileId())
		}
		if filter.Vaccinated != nil {
			add("vaccinated = $%d", filter.GetVaccinated())
		}
		if filter.Sterilized != nil {
			add("sterilized = $%d", filter.GetSterilized())
		}
	}
	if len(predicates) > 0 {
		query += "\nWHERE " + strings.Join(predicates, " AND ")
	}
	query += "\nORDER BY animal_id"
	return query, args
}

func enumSlice[T ~int32](values []T) []int32 {
	out := make([]int32, len(values))
	for i, value := range values {
		out[i] = int32(value)
	}
	return out
}

func decodeScores(raw []byte) (map[string]float64, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var scores map[string]float64
	if err := json.Unmarshal(raw, &scores); err != nil {
		return nil, err
	}
	return scores, nil
}

func appendReason(reasons []string, reason string) []string {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

const upsertCandidateSQL = `
INSERT INTO feed_candidates (
	animal_id, animal_proto, owner_display_name, owner_average_rating, ranking_reasons,
	score_components, distance_km, owner_hidden, owner_blocked, species, status, city,
	boosted, owner_profile_id, vaccinated, sterilized
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
ON CONFLICT (animal_id) DO UPDATE SET
	animal_proto = EXCLUDED.animal_proto,
	owner_display_name = EXCLUDED.owner_display_name,
	owner_average_rating = EXCLUDED.owner_average_rating,
	ranking_reasons = EXCLUDED.ranking_reasons,
	score_components = EXCLUDED.score_components,
	distance_km = EXCLUDED.distance_km,
	owner_hidden = EXCLUDED.owner_hidden,
	owner_blocked = EXCLUDED.owner_blocked,
	species = EXCLUDED.species,
	status = EXCLUDED.status,
	city = EXCLUDED.city,
	boosted = EXCLUDED.boosted,
	owner_profile_id = EXCLUDED.owner_profile_id,
	vaccinated = EXCLUDED.vaccinated,
	sterilized = EXCLUDED.sterilized`
