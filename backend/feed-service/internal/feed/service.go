package feed

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	topicCardServed     = "feed.card_served"
	topicCardOpened     = "feed.card_opened"
	topicFiltersApplied = "feed.filters_applied"

	defaultPageSize       = 20
	maxPageSize           = 50
	defaultOrganicAfterAd = 3
	serviceProducer       = "feed-service"
	eventSchemaVersion    = "v1"
	permissionExplainFeed = "feed.ranking.explain"
	roleAdmin             = "admin"
)

// ErrNotFound is returned when a stored feed entity cannot be found.
var ErrNotFound = errors.New("not found")

// Candidate is a rankable animal feed candidate returned by the data layer.
type Candidate struct {
	Animal             *animalv1.AnimalProfile
	OwnerDisplayName   string
	OwnerAverageRating float64
	RankingReasons     []string
	ScoreComponents    map[string]float64
	DistanceKM         int32
	OwnerHidden        bool
	OwnerBlocked       bool
}

// RankingPolicy controls how boosted profiles are mixed with organic results.
type RankingPolicy struct {
	OrganicAfterBoost int
}

// Store persists and retrieves feed candidates, served cards, and idempotent opens.
type Store interface {
	ListCandidates(context.Context, *animalv1.AnimalFilter) ([]Candidate, error)
	SaveServedCards(context.Context, []*feedv1.FeedCard, map[string]map[string]float64) error
	GetServedCard(context.Context, string) (*feedv1.FeedCard, map[string]float64, error)
	RecordCardOpen(context.Context, string, string, time.Time) (*feedv1.RecordCardOpenResponse, bool, error)
}

// SwipeStore returns animal ids previously swiped by an actor.
type SwipeStore interface {
	SeenAnimalIDs(context.Context, *commonv1.Principal) (map[string]struct{}, error)
}

// EntitlementChecker decides whether paid advanced feed filters may be applied.
type EntitlementChecker interface {
	CanUsePaidAdvancedFilters(context.Context, *commonv1.Principal) (bool, error)
}

// Publisher publishes feed telemetry events to the service event bus.
type Publisher interface {
	Publish(context.Context, string, proto.Message) error
}

// AnimalArchivedPayload is the animal.profile_archived payload described by feed-service.md.
type AnimalArchivedPayload struct {
	AnimalID       string `json:"animal_id"`
	OwnerProfileID string `json:"owner_profile_id"`
	PreviousStatus string `json:"previous_status"`
	Reason         string `json:"reason"`
}

// EventApplier applies consumed upstream events to the local feed projection.
type EventApplier interface {
	UpsertAnimalProfile(context.Context, *animalv1.AnimalProfile) error
	ArchiveAnimal(context.Context, AnimalArchivedPayload) error
	UpdateAnimalStatus(context.Context, *animalv1.AnimalStatusChangedEvent) error
	RecordSwipe(context.Context, *matchingv1.Swipe) error
	ActivateBoost(context.Context, *billingv1.Boost) error
	GrantEntitlement(context.Context, *billingv1.Entitlement) error
	UpdateAnimalStats(context.Context, *analyticsv1.AnimalStats) error
}

// Dependencies groups Service dependencies.
type Dependencies struct {
	Store        Store
	Swipes       SwipeStore
	Entitlements EntitlementChecker
	Publisher    Publisher
	Clock        func() time.Time
	IDGenerator  func() string
	Ranking      RankingPolicy
}

// Service implements petmatch.feed.v1.FeedServiceServer.
type Service struct {
	feedv1.UnimplementedFeedServiceServer

	store        Store
	swipes       SwipeStore
	entitlements EntitlementChecker
	publisher    Publisher
	clock        func() time.Time
	newID        func() string
	ranking      RankingPolicy
}

// NewService creates a production service with safe defaults for optional dependencies.
func NewService(deps Dependencies) *Service {
	service := &Service{
		store:        deps.Store,
		swipes:       deps.Swipes,
		entitlements: deps.Entitlements,
		publisher:    deps.Publisher,
		clock:        deps.Clock,
		newID:        deps.IDGenerator,
		ranking:      deps.Ranking,
	}
	if service.store == nil {
		service.store = emptyStore{}
	}
	if service.swipes == nil {
		service.swipes = emptySwipeStore{}
	}
	if service.entitlements == nil {
		service.entitlements = emptyEntitlements{}
	}
	if service.publisher == nil {
		service.publisher = noopPublisher{}
	}
	if service.clock == nil {
		service.clock = time.Now
	}
	if service.newID == nil {
		service.newID = randomID
	}
	if service.ranking.OrganicAfterBoost <= 0 {
		service.ranking.OrganicAfterBoost = defaultOrganicAfterAd
	}
	return service
}

// GetFeed returns a filtered, ranked, paginated feed and publishes serving telemetry.
func (s *Service) GetFeed(ctx context.Context, req *feedv1.GetFeedRequest) (*feedv1.GetFeedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	pageSize, offset, err := pageBounds(req.Page)
	if err != nil {
		return nil, err
	}

	paidFiltersUsed, effectiveFilter, err := s.effectiveAnimalFilter(ctx, req)
	if err != nil {
		return nil, err
	}

	candidates, err := s.store.ListCandidates(ctx, effectiveFilter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list candidates: %v", err)
	}
	seen, err := s.swipes.SeenAnimalIDs(ctx, req.Principal)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load swipes: %v", err)
	}

	excluded := excludedAnimalIDs(req.Filter)
	candidates = filterCandidates(candidates, req.Surface, excluded, seen)
	ranked := RankCandidates(candidates, s.ranking)
	window, nextToken := paginate(ranked, offset, pageSize)

	sessionID := s.newID()
	now := s.clock()
	cards := make([]*feedv1.FeedCard, 0, len(window))
	scores := make(map[string]map[string]float64, len(window))
	for _, candidate := range window {
		cardID := s.newID()
		card := &feedv1.FeedCard{
			FeedCardId:         cardID,
			FeedSessionId:      sessionID,
			Animal:             proto.Clone(candidate.Animal).(*animalv1.AnimalProfile),
			OwnerProfileId:     candidate.Animal.OwnerProfileId,
			OwnerDisplayName:   candidate.OwnerDisplayName,
			OwnerAverageRating: candidate.OwnerAverageRating,
			Boosted:            candidate.Animal.Boosted,
			RankingReasons:     append([]string(nil), candidate.RankingReasons...),
			ServedAt:           timestamppb.New(now),
		}
		cards = append(cards, card)
		scores[cardID] = cloneScores(candidate.ScoreComponents)
	}
	if err := s.store.SaveServedCards(ctx, cards, scores); err != nil {
		return nil, status.Errorf(codes.Internal, "save served cards: %v", err)
	}

	if req.Filter != nil {
		if err := s.publishFiltersApplied(ctx, req, paidFiltersUsed, now); err != nil {
			return nil, err
		}
	}
	for _, card := range cards {
		if err := s.publishCardServed(ctx, req, card, now); err != nil {
			return nil, err
		}
	}

	response := &feedv1.GetFeedResponse{
		Cards:         cards,
		FeedSessionId: sessionID,
		Page:          &commonv1.PageResponse{NextPageToken: nextToken},
	}
	return response, nil
}

// StreamFeed streams cards from GetFeed for client-side prefetching.
func (s *Service) StreamFeed(req *feedv1.GetFeedRequest, stream feedv1.FeedService_StreamFeedServer) error {
	resp, err := s.GetFeed(stream.Context(), req)
	if err != nil {
		return err
	}
	for _, card := range resp.Cards {
		if err := stream.Send(card); err != nil {
			return status.Errorf(codes.Unavailable, "send feed card: %v", err)
		}
	}
	return nil
}

// GetFeedCard returns a previously served feed card.
func (s *Service) GetFeedCard(ctx context.Context, req *feedv1.GetFeedCardRequest) (*feedv1.GetFeedCardResponse, error) {
	if req == nil || req.FeedCardId == "" {
		return nil, status.Error(codes.InvalidArgument, "feed_card_id is required")
	}
	card, _, err := s.store.GetServedCard(ctx, req.FeedCardId)
	if err != nil {
		return nil, asStatus(err, "get feed card")
	}
	return &feedv1.GetFeedCardResponse{Card: card}, nil
}

// RecordCardOpen records a card open once per idempotency key and publishes telemetry.
func (s *Service) RecordCardOpen(ctx context.Context, req *feedv1.RecordCardOpenRequest) (*feedv1.RecordCardOpenResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.FeedCardId == "" {
		return nil, status.Error(codes.InvalidArgument, "feed_card_id is required")
	}
	if req.AnimalId == "" {
		return nil, status.Error(codes.InvalidArgument, "animal_id is required")
	}
	key := req.IdempotencyKey
	if key == "" {
		key = req.FeedSessionId + ":" + req.FeedCardId + ":" + actorID(req.Principal)
	}
	openedAt := s.clock()
	resp, created, err := s.store.RecordCardOpen(ctx, key, s.newID(), openedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "record card open: %v", err)
	}
	if !created {
		return resp, nil
	}
	event := &feedv1.FeedCardOpenedEvent{
		Envelope:   s.envelope(topicCardOpened, openedAt, req.AnimalId, key),
		CardOpenId: resp.CardOpenId,
		FeedCardId: req.FeedCardId,
		AnimalId:   req.AnimalId,
		ActorId:    optionalActorID(req.Principal),
	}
	if err := s.publisher.Publish(ctx, topicCardOpened, event); err != nil {
		return nil, status.Errorf(codes.Internal, "publish card opened: %v", err)
	}
	return resp, nil
}

// ExplainRanking returns ranking reasons and score components for admin/debug use.
func (s *Service) ExplainRanking(ctx context.Context, req *feedv1.ExplainRankingRequest) (*feedv1.ExplainRankingResponse, error) {
	if req == nil || req.FeedCardId == "" {
		return nil, status.Error(codes.InvalidArgument, "feed_card_id is required")
	}
	if !canExplainRanking(req.Principal) {
		return nil, status.Error(codes.PermissionDenied, "ranking explanation requires admin privileges")
	}
	card, scores, err := s.store.GetServedCard(ctx, req.FeedCardId)
	if err != nil {
		return nil, asStatus(err, "explain ranking")
	}
	return &feedv1.ExplainRankingResponse{
		RankingReasons:  append([]string(nil), card.RankingReasons...),
		ScoreComponents: scores,
	}, nil
}

// RankCandidates sorts candidates by score and interleaves boosted profiles with organic results.
func RankCandidates(candidates []Candidate, policy RankingPolicy) []Candidate {
	if policy.OrganicAfterBoost <= 0 {
		policy.OrganicAfterBoost = defaultOrganicAfterAd
	}

	boosted := make([]Candidate, 0, len(candidates))
	organic := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Animal != nil && candidate.Animal.Boosted {
			boosted = append(boosted, candidate)
			continue
		}
		organic = append(organic, candidate)
	}
	sortCandidates(boosted)
	sortCandidates(organic)

	ranked := make([]Candidate, 0, len(candidates))
	boostedIndex := 0
	organicIndex := 0
	for boostedIndex < len(boosted) || organicIndex < len(organic) {
		if boostedIndex < len(boosted) {
			ranked = append(ranked, boosted[boostedIndex])
			boostedIndex++
		}
		for i := 0; i < policy.OrganicAfterBoost && organicIndex < len(organic); i++ {
			ranked = append(ranked, organic[organicIndex])
			organicIndex++
		}
		if organicIndex >= len(organic) {
			for boostedIndex < len(boosted) {
				ranked = append(ranked, boosted[boostedIndex])
				boostedIndex++
			}
		}
	}
	return ranked
}

func (s *Service) effectiveAnimalFilter(ctx context.Context, req *feedv1.GetFeedRequest) (bool, *animalv1.AnimalFilter, error) {
	if req.Filter == nil || req.Filter.AnimalFilter == nil {
		return false, nil, nil
	}
	filter := proto.Clone(req.Filter.AnimalFilter).(*animalv1.AnimalFilter)
	if !req.Filter.UsePaidAdvancedFilters {
		stripPaidAdvancedFilters(filter)
		return false, filter, nil
	}
	allowed, err := s.entitlements.CanUsePaidAdvancedFilters(ctx, req.Principal)
	if err != nil {
		return false, nil, status.Errorf(codes.Internal, "check entitlements: %v", err)
	}
	if !allowed {
		stripPaidAdvancedFilters(filter)
		return false, filter, nil
	}
	return true, filter, nil
}

func stripPaidAdvancedFilters(filter *animalv1.AnimalFilter) {
	filter.Near = nil
	filter.RadiusKm = nil
	filter.Traits = nil
	filter.Vaccinated = nil
	filter.Sterilized = nil
	filter.BoostedOnly = nil
}

func filterCandidates(candidates []Candidate, surface feedv1.FeedSurface, excluded map[string]struct{}, seen map[string]struct{}) []Candidate {
	filtered := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Animal == nil {
			continue
		}
		id := candidate.Animal.AnimalId
		if _, ok := excluded[id]; ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if candidate.OwnerHidden || candidate.OwnerBlocked {
			continue
		}
		if candidate.Animal.Status != animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE {
			continue
		}
		if !visibleOnSurface(candidate.Animal.Visibility, surface) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func visibleOnSurface(visibility commonv1.Visibility, surface feedv1.FeedSurface) bool {
	if visibility == commonv1.Visibility_VISIBILITY_PUBLIC {
		return true
	}
	return surface == feedv1.FeedSurface_FEED_SURFACE_OWNER_PROFILE &&
		visibility == commonv1.Visibility_VISIBILITY_UNLISTED
}

func excludedAnimalIDs(filter *feedv1.FeedFilter) map[string]struct{} {
	excluded := map[string]struct{}{}
	if filter == nil {
		return excluded
	}
	for _, id := range filter.ExcludedAnimalIds {
		excluded[id] = struct{}{}
	}
	return excluded
}

func pageBounds(req *commonv1.PageRequest) (int, int, error) {
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	offset := 0
	if token := req.GetPageToken(); token != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			return 0, 0, status.Error(codes.InvalidArgument, "invalid page token")
		}
		parsed, err := strconv.Atoi(string(decoded))
		if err != nil || parsed < 0 {
			return 0, 0, status.Error(codes.InvalidArgument, "invalid page token")
		}
		offset = parsed
	}
	return pageSize, offset, nil
}

func paginate(candidates []Candidate, offset int, pageSize int) ([]Candidate, string) {
	if offset >= len(candidates) {
		return nil, ""
	}
	end := offset + pageSize
	if end > len(candidates) {
		end = len(candidates)
	}
	nextToken := ""
	if end < len(candidates) {
		nextToken = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(end)))
	}
	return candidates[offset:end], nextToken
}

func sortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return totalScore(candidates[i]) > totalScore(candidates[j])
	})
}

func totalScore(candidate Candidate) float64 {
	total := 0.0
	for _, score := range candidate.ScoreComponents {
		total += score
	}
	return total
}

func (s *Service) publishFiltersApplied(ctx context.Context, req *feedv1.GetFeedRequest, paidFiltersUsed bool, occurredAt time.Time) error {
	event := &feedv1.FeedFiltersAppliedEvent{
		Envelope:        s.envelope(topicFiltersApplied, occurredAt, actorID(req.Principal), ""),
		ActorId:         optionalActorID(req.Principal),
		Filter:          proto.Clone(req.Filter).(*feedv1.FeedFilter),
		PaidFiltersUsed: paidFiltersUsed,
	}
	if err := s.publisher.Publish(ctx, topicFiltersApplied, event); err != nil {
		return status.Errorf(codes.Internal, "publish filters applied: %v", err)
	}
	return nil
}

func (s *Service) publishCardServed(ctx context.Context, req *feedv1.GetFeedRequest, card *feedv1.FeedCard, occurredAt time.Time) error {
	event := &feedv1.FeedCardServedEvent{
		Envelope:      s.envelope(topicCardServed, occurredAt, card.FeedSessionId, ""),
		FeedCardId:    card.FeedCardId,
		FeedSessionId: card.FeedSessionId,
		AnimalId:      card.Animal.AnimalId,
		ActorId:       optionalActorID(req.Principal),
		Boosted:       card.Boosted,
		Surface:       req.Surface,
	}
	if err := s.publisher.Publish(ctx, topicCardServed, event); err != nil {
		return status.Errorf(codes.Internal, "publish card served: %v", err)
	}
	return nil
}

func (s *Service) envelope(eventType string, occurredAt time.Time, partitionKey string, idempotencyKey string) *commonv1.EventEnvelope {
	return &commonv1.EventEnvelope{
		EventId:        s.newID(),
		EventType:      eventType,
		SchemaVersion:  eventSchemaVersion,
		Producer:       serviceProducer,
		OccurredAt:     timestamppb.New(occurredAt),
		IdempotencyKey: idempotencyKey,
		PartitionKey:   partitionKey,
	}
}

func canExplainRanking(principal *commonv1.Principal) bool {
	if principal.GetActorType() == commonv1.ActorType_ACTOR_TYPE_ADMIN {
		return true
	}
	for _, role := range principal.GetRoles() {
		if role == roleAdmin {
			return true
		}
	}
	for _, permission := range principal.GetPermissions() {
		if permission == permissionExplainFeed {
			return true
		}
	}
	return false
}

func optionalActorID(principal *commonv1.Principal) *string {
	id := actorID(principal)
	if id == "" {
		return nil
	}
	return &id
}

func actorID(principal *commonv1.Principal) string {
	if principal == nil {
		return ""
	}
	return principal.ActorId
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

func asStatus(err error, operation string) error {
	if errors.Is(err, ErrNotFound) {
		return status.Errorf(codes.NotFound, "%s: %v", operation, err)
	}
	return status.Errorf(codes.Internal, "%s: %v", operation, err)
}

func randomID() string {
	var bytes [16]byte
	if _, err := io.ReadFull(rand.Reader, bytes[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes[:])
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, string, proto.Message) error {
	return nil
}

type emptyStore struct{}

func (emptyStore) ListCandidates(context.Context, *animalv1.AnimalFilter) ([]Candidate, error) {
	return nil, nil
}

func (emptyStore) SaveServedCards(context.Context, []*feedv1.FeedCard, map[string]map[string]float64) error {
	return nil
}

func (emptyStore) GetServedCard(context.Context, string) (*feedv1.FeedCard, map[string]float64, error) {
	return nil, nil, ErrNotFound
}

func (emptyStore) RecordCardOpen(_ context.Context, _ string, cardOpenID string, openedAt time.Time) (*feedv1.RecordCardOpenResponse, bool, error) {
	return &feedv1.RecordCardOpenResponse{
		CardOpenId: cardOpenID,
		OpenedAt:   timestamppb.New(openedAt),
	}, true, nil
}

type emptySwipeStore struct{}

func (emptySwipeStore) SeenAnimalIDs(context.Context, *commonv1.Principal) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}

type emptyEntitlements struct{}

func (emptyEntitlements) CanUsePaidAdvancedFilters(context.Context, *commonv1.Principal) (bool, error) {
	return false, nil
}
