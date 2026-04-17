// Package memory provides in-memory adapters for local feed-service runs and tests.
package memory

import (
	"context"
	"sync"
	"time"

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

// Store is an in-memory feed projection and repository.
type Store struct {
	mu         sync.RWMutex
	candidates []feed.Candidate
	cards      map[string]storedCard
	opens      map[string]*feedv1.RecordCardOpenResponse
	seen       map[string]map[string]struct{}
	entitled   map[string]struct{}
}

type storedCard struct {
	card   *feedv1.FeedCard
	scores map[string]float64
}

// NewStore creates an in-memory store.
func NewStore(candidates []feed.Candidate) *Store {
	cloned := make([]feed.Candidate, len(candidates))
	copy(cloned, candidates)
	return &Store{
		candidates: cloned,
		cards:      map[string]storedCard{},
		opens:      map[string]*feedv1.RecordCardOpenResponse{},
		seen:       map[string]map[string]struct{}{},
		entitled:   map[string]struct{}{},
	}
}

// ListCandidates returns candidates matching the basic animal filter.
func (s *Store) ListCandidates(ctx context.Context, filter *animalv1.AnimalFilter) ([]feed.Candidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	matched := make([]feed.Candidate, 0, len(s.candidates))
	for _, candidate := range s.candidates {
		if matchesAnimalFilter(candidate, filter) {
			matched = append(matched, cloneCandidate(candidate))
		}
	}
	return matched, nil
}

// SaveServedCards stores generated cards for later GetFeedCard and ExplainRanking calls.
func (s *Store) SaveServedCards(ctx context.Context, cards []*feedv1.FeedCard, scores map[string]map[string]float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, card := range cards {
		s.cards[card.FeedCardId] = storedCard{
			card:   proto.Clone(card).(*feedv1.FeedCard),
			scores: cloneScores(scores[card.FeedCardId]),
		}
	}
	return nil
}

// GetServedCard returns a previously served card and its ranking scores.
func (s *Store) GetServedCard(ctx context.Context, feedCardID string) (*feedv1.FeedCard, map[string]float64, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.cards[feedCardID]
	if !ok {
		return nil, nil, feed.ErrNotFound
	}
	return proto.Clone(stored.card).(*feedv1.FeedCard), cloneScores(stored.scores), nil
}

// RecordCardOpen stores an idempotent card-open result.
func (s *Store) RecordCardOpen(ctx context.Context, key string, cardOpenID string, openedAt time.Time) (*feedv1.RecordCardOpenResponse, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.opens[key]; ok {
		return proto.Clone(existing).(*feedv1.RecordCardOpenResponse), false, nil
	}
	resp := &feedv1.RecordCardOpenResponse{CardOpenId: cardOpenID, OpenedAt: timestamppb.New(openedAt)}
	s.opens[key] = proto.Clone(resp).(*feedv1.RecordCardOpenResponse)
	return resp, true, nil
}

// UpsertAnimalProfile adds or updates an animal in the feed projection.
func (s *Store) UpsertAnimalProfile(ctx context.Context, animal *animalv1.AnimalProfile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if animal == nil || animal.AnimalId == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	candidate := feed.Candidate{
		Animal:           proto.Clone(animal).(*animalv1.AnimalProfile),
		OwnerDisplayName: animal.OwnerProfileId,
		RankingReasons:   []string{"recently published"},
		ScoreComponents:  map[string]float64{"freshness": 1},
	}
	for i := range s.candidates {
		if s.candidates[i].Animal.GetAnimalId() == animal.AnimalId {
			s.candidates[i] = mergeCandidateProjection(s.candidates[i], candidate)
			return nil
		}
	}
	s.candidates = append(s.candidates, candidate)
	return nil
}

// ArchiveAnimal removes an animal from active feed candidates and cached served cards.
func (s *Store) ArchiveAnimal(ctx context.Context, event feed.AnimalArchivedPayload) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.AnimalID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.removeAnimalLocked(event.AnimalID)
	return nil
}

// UpdateAnimalStatus updates feed eligibility after an upstream status change.
func (s *Store) UpdateAnimalStatus(ctx context.Context, event *animalv1.AnimalStatusChangedEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event == nil || event.AnimalId == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.NewStatus != animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE {
		s.removeAnimalLocked(event.AnimalId)
		return nil
	}
	for i := range s.candidates {
		if s.candidates[i].Animal.GetAnimalId() == event.AnimalId {
			s.candidates[i].Animal.Status = event.NewStatus
			return nil
		}
	}
	return nil
}

// RecordSwipe stores a user's swiped animal id for feed suppression.
func (s *Store) RecordSwipe(ctx context.Context, swipe *matchingv1.Swipe) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if swipe == nil || swipe.ActorId == "" || swipe.AnimalId == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.seen[swipe.ActorId] == nil {
		s.seen[swipe.ActorId] = map[string]struct{}{}
	}
	s.seen[swipe.ActorId][swipe.AnimalId] = struct{}{}
	return nil
}

// ActivateBoost updates boost priority for an animal.
func (s *Store) ActivateBoost(ctx context.Context, boost *billingv1.Boost) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if boost == nil || boost.AnimalId == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.candidates {
		if s.candidates[i].Animal.GetAnimalId() == boost.AnimalId {
			s.candidates[i].Animal.Boosted = boost.Active
			s.candidates[i].Animal.BoostExpiresAt = boost.ExpiresAt
			if boost.Active {
				s.candidates[i].ScoreComponents["boost"] = 1
				s.candidates[i].RankingReasons = appendReason(s.candidates[i].RankingReasons, "active boost")
			} else {
				delete(s.candidates[i].ScoreComponents, "boost")
			}
			return nil
		}
	}
	return nil
}

// GrantEntitlement updates the in-memory advanced-filter entitlement cache.
func (s *Store) GrantEntitlement(ctx context.Context, entitlement *billingv1.Entitlement) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if entitlement == nil || entitlement.OwnerProfileId == "" {
		return nil
	}
	if entitlement.Type != billingv1.EntitlementType_ENTITLEMENT_TYPE_ADVANCED_FILTERS {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if entitlement.Active {
		s.entitled[entitlement.OwnerProfileId] = struct{}{}
		return nil
	}
	delete(s.entitled, entitlement.OwnerProfileId)
	return nil
}

// UpdateAnimalStats updates ranking score components from aggregated analytics.
func (s *Store) UpdateAnimalStats(ctx context.Context, stats *analyticsv1.AnimalStats) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if stats == nil || stats.AnimalId == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.candidates {
		if s.candidates[i].Animal.GetAnimalId() == stats.AnimalId {
			s.candidates[i].ScoreComponents["ctr"] = stats.Ctr
			s.candidates[i].RankingReasons = appendReason(s.candidates[i].RankingReasons, "analytics engagement")
			return nil
		}
	}
	return nil
}

// SeenAnimalIDs returns swiped animal ids stored by consumed matching events.
func (s *Store) SeenAnimalIDs(ctx context.Context, principal *commonv1.Principal) (map[string]struct{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := s.seen[principal.GetActorId()]
	cloned := make(map[string]struct{}, len(seen))
	for id := range seen {
		cloned[id] = struct{}{}
	}
	return cloned, nil
}

// CanUsePaidAdvancedFilters checks the entitlement cache built from billing events.
func (s *Store) CanUsePaidAdvancedFilters(ctx context.Context, principal *commonv1.Principal) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.entitled[principal.GetActorId()]
	return ok, nil
}

// StaticSwipeStore is a fixed SwipeStore keyed by actor id.
type StaticSwipeStore map[string]map[string]struct{}

// SeenAnimalIDs returns a copy of actor seen ids.
func (s StaticSwipeStore) SeenAnimalIDs(ctx context.Context, principal *commonv1.Principal) (map[string]struct{}, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	seen := s[principal.GetActorId()]
	cloned := make(map[string]struct{}, len(seen))
	for id := range seen {
		cloned[id] = struct{}{}
	}
	return cloned, nil
}

// StaticEntitlements is a fixed EntitlementChecker for tests.
type StaticEntitlements struct {
	PaidAdvancedFilters bool
}

// CanUsePaidAdvancedFilters returns the configured entitlement value.
func (s StaticEntitlements) CanUsePaidAdvancedFilters(context.Context, *commonv1.Principal) (bool, error) {
	return s.PaidAdvancedFilters, nil
}

// Publisher stores published events in memory.
type Publisher struct {
	mu     sync.RWMutex
	topics []string
	events []proto.Message
}

// NewPublisher creates an in-memory publisher.
func NewPublisher() *Publisher {
	return &Publisher{}
}

// Publish records an event and its topic.
func (p *Publisher) Publish(ctx context.Context, topic string, event proto.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.topics = append(p.topics, topic)
	p.events = append(p.events, proto.Clone(event))
	return nil
}

// Topics returns published topics in order.
func (p *Publisher) Topics() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	topics := make([]string, len(p.topics))
	copy(topics, p.topics)
	return topics
}

// Events returns published events in order.
func (p *Publisher) Events() []proto.Message {
	p.mu.RLock()
	defer p.mu.RUnlock()

	events := make([]proto.Message, len(p.events))
	for i, event := range p.events {
		events[i] = proto.Clone(event)
	}
	return events
}

func matchesAnimalFilter(candidate feed.Candidate, filter *animalv1.AnimalFilter) bool {
	if candidate.Animal == nil || filter == nil {
		return true
	}
	animal := candidate.Animal
	if len(filter.Species) > 0 && !containsSpecies(filter.Species, animal.Species) {
		return false
	}
	if len(filter.Statuses) > 0 && !containsStatus(filter.Statuses, animal.Status) {
		return false
	}
	if filter.City != nil && animal.GetLocation().GetCity() != filter.GetCity() {
		return false
	}
	if filter.RadiusKm != nil && candidate.DistanceKM > filter.GetRadiusKm() {
		return false
	}
	if filter.BoostedOnly != nil && filter.GetBoostedOnly() && !animal.Boosted {
		return false
	}
	if filter.OwnerProfileId != nil && animal.OwnerProfileId != filter.GetOwnerProfileId() {
		return false
	}
	if filter.Vaccinated != nil && animal.Vaccinated != filter.GetVaccinated() {
		return false
	}
	if filter.Sterilized != nil && animal.Sterilized != filter.GetSterilized() {
		return false
	}
	return true
}

func containsSpecies(values []animalv1.Species, needle animalv1.Species) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func containsStatus(values []animalv1.AnimalStatus, needle animalv1.AnimalStatus) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func cloneCandidate(candidate feed.Candidate) feed.Candidate {
	cloned := candidate
	if candidate.Animal != nil {
		cloned.Animal = proto.Clone(candidate.Animal).(*animalv1.AnimalProfile)
	}
	cloned.RankingReasons = append([]string(nil), candidate.RankingReasons...)
	cloned.ScoreComponents = cloneScores(candidate.ScoreComponents)
	return cloned
}

func cloneScores(scores map[string]float64) map[string]float64 {
	if len(scores) == 0 {
		return nil
	}
	cloned := make(map[string]float64, len(scores))
	for key, value := range scores {
		cloned[key] = value
	}
	return cloned
}

func (s *Store) removeAnimalLocked(animalID string) {
	candidates := s.candidates[:0]
	for _, candidate := range s.candidates {
		if candidate.Animal.GetAnimalId() != animalID {
			candidates = append(candidates, candidate)
		}
	}
	s.candidates = candidates
	for cardID, stored := range s.cards {
		if stored.card.GetAnimal().GetAnimalId() == animalID {
			delete(s.cards, cardID)
		}
	}
}

func mergeCandidateProjection(existing feed.Candidate, incoming feed.Candidate) feed.Candidate {
	incoming.OwnerDisplayName = firstNonEmpty(existing.OwnerDisplayName, incoming.OwnerDisplayName)
	incoming.OwnerAverageRating = existing.OwnerAverageRating
	if len(existing.RankingReasons) > 0 {
		incoming.RankingReasons = existing.RankingReasons
	}
	if len(existing.ScoreComponents) > 0 {
		incoming.ScoreComponents = cloneScores(existing.ScoreComponents)
	}
	incoming.DistanceKM = existing.DistanceKM
	incoming.OwnerHidden = existing.OwnerHidden
	incoming.OwnerBlocked = existing.OwnerBlocked
	return incoming
}

func appendReason(reasons []string, reason string) []string {
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func firstNonEmpty(first string, second string) string {
	if first != "" {
		return first
	}
	return second
}
