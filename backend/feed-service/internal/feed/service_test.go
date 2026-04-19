package feed_test

import (
	"context"
	"errors"
	"testing"
	"time"

	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	"github.com/petmatch/petmatch/internal/adapters/memory"
	"github.com/petmatch/petmatch/internal/feed"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRankCandidatesInterleavesBoostedWithOrganicResults(t *testing.T) {
	ranked := feed.RankCandidates([]feed.Candidate{
		testCandidate("boosted-1", true, 0.99),
		testCandidate("boosted-2", true, 0.98),
		testCandidate("organic-1", false, 0.97),
		testCandidate("organic-2", false, 0.96),
		testCandidate("organic-3", false, 0.95),
	}, feed.RankingPolicy{OrganicAfterBoost: 1})

	got := animalIDs(ranked)
	want := []string{"boosted-1", "organic-1", "boosted-2", "organic-2", "organic-3"}
	assertStrings(t, got, want)
}

func TestGetFeedFiltersRanksAndPublishesTelemetry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	radius := int32(10)
	boostedOnly := true

	store := memory.NewStore([]feed.Candidate{
		testCandidateWithDistance("boosted-far", true, 0.99, 50),
		testCandidateWithDistance("organic-near", false, 0.80, 2),
		testCandidateWithDistance("private-cat", false, 0.79, 2, withVisibility(commonv1.Visibility_VISIBILITY_PRIVATE)),
		testCandidateWithDistance("archived-cat", false, 0.78, 2, withStatus(animalv1.AnimalStatus_ANIMAL_STATUS_ARCHIVED)),
		testCandidateWithDistance("swiped-cat", false, 0.77, 2),
		testCandidateWithDistance("excluded-cat", false, 0.76, 2),
		testCandidateWithDistance("dog-near", false, 0.75, 2, withSpecies(animalv1.Species_SPECIES_DOG)),
	})
	publisher := memory.NewPublisher()
	service := feed.NewService(feed.Dependencies{
		Store:        store,
		Swipes:       memory.StaticSwipeStore{"user-1": {"swiped-cat": {}}},
		Entitlements: memory.StaticEntitlements{PaidAdvancedFilters: false},
		Publisher:    publisher,
		Clock:        func() time.Time { return now },
		IDGenerator:  sequenceIDs("session-1", "card-1", "card-2", "event-1", "event-2", "event-3"),
		Ranking:      feed.RankingPolicy{OrganicAfterBoost: 1},
	})

	resp, err := service.GetFeed(ctx, &feedv1.GetFeedRequest{
		Principal: &commonv1.Principal{
			ActorId:   "user-1",
			ActorType: commonv1.ActorType_ACTOR_TYPE_USER,
		},
		Surface: feedv1.FeedSurface_FEED_SURFACE_MAIN,
		Filter: &feedv1.FeedFilter{
			AnimalFilter: &animalv1.AnimalFilter{
				Species:     []animalv1.Species{animalv1.Species_SPECIES_CAT},
				RadiusKm:    &radius,
				BoostedOnly: &boostedOnly,
			},
			UsePaidAdvancedFilters: true,
			ExcludedAnimalIds:      []string{"excluded-cat"},
		},
		Page: &commonv1.PageRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("GetFeed returned error: %v", err)
	}

	assertStrings(t, feedAnimalIDs(resp.Cards), []string{"boosted-far", "organic-near"})
	if resp.FeedSessionId != "session-1" {
		t.Fatalf("feed session id = %q, want session-1", resp.FeedSessionId)
	}
	if got := resp.Cards[0].ServedAt.AsTime(); !got.Equal(now) {
		t.Fatalf("served_at = %s, want %s", got, now)
	}

	waitForTopics(t, publisher, []string{"feed.filters_applied", "feed.card_served", "feed.card_served"})
	filtersEvent := publisher.Events()[0].(*feedv1.FeedFiltersAppliedEvent)
	if filtersEvent.PaidFiltersUsed {
		t.Fatal("paid filters were reported as used without entitlement")
	}
}

func TestGetFeedExcludesCurrentActorsOwnAnimals(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore([]feed.Candidate{
		testCandidateWithDistance("own-animal", false, 0.99, 1, withOwnerProfileID("user-1")),
		testCandidateWithDistance("other-animal", false, 0.98, 1, withOwnerProfileID("owner-2")),
	})
	service := feed.NewService(feed.Dependencies{
		Store:       store,
		Publisher:   memory.NewPublisher(),
		Clock:       func() time.Time { return time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC) },
		IDGenerator: sequenceIDs("session-1", "card-1"),
	})

	resp, err := service.GetFeed(ctx, &feedv1.GetFeedRequest{
		Principal: &commonv1.Principal{ActorId: "user-1"},
		Surface:   feedv1.FeedSurface_FEED_SURFACE_MAIN,
		Page:      &commonv1.PageRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("GetFeed returned error: %v", err)
	}

	assertStrings(t, feedAnimalIDs(resp.Cards), []string{"other-animal"})
}

func TestGetFeedReturnsCardsWhenTelemetryPublishFails(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore([]feed.Candidate{
		testCandidate("animal-1", false, 0.80),
	})
	service := feed.NewService(feed.Dependencies{
		Store:       store,
		Publisher:   failingPublisher{err: errors.New("kafka unavailable")},
		Clock:       func() time.Time { return time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC) },
		IDGenerator: sequenceIDs("session-1", "card-1", "event-1"),
	})

	resp, err := service.GetFeed(ctx, &feedv1.GetFeedRequest{
		Principal: &commonv1.Principal{ActorId: "user-1"},
		Surface:   feedv1.FeedSurface_FEED_SURFACE_MAIN,
		Page:      &commonv1.PageRequest{PageSize: 10},
	})
	if err != nil {
		t.Fatalf("GetFeed returned error: %v", err)
	}

	assertStrings(t, feedAnimalIDs(resp.Cards), []string{"animal-1"})
}

func TestGetFeedDoesNotWaitForTelemetryPublish(t *testing.T) {
	ctx := context.Background()
	publisher := newBlockingPublisher()
	store := memory.NewStore([]feed.Candidate{
		testCandidate("animal-1", false, 0.80),
	})
	service := feed.NewService(feed.Dependencies{
		Store:       store,
		Publisher:   publisher,
		Clock:       func() time.Time { return time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC) },
		IDGenerator: sequenceIDs("session-1", "card-1", "event-1"),
	})

	type result struct {
		resp *feedv1.GetFeedResponse
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := service.GetFeed(ctx, &feedv1.GetFeedRequest{
			Principal: &commonv1.Principal{ActorId: "user-1"},
			Surface:   feedv1.FeedSurface_FEED_SURFACE_MAIN,
			Page:      &commonv1.PageRequest{PageSize: 10},
		})
		done <- result{resp: resp, err: err}
	}()

	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("GetFeed returned error: %v", got.err)
		}
		assertStrings(t, feedAnimalIDs(got.resp.Cards), []string{"animal-1"})
	case <-time.After(100 * time.Millisecond):
		publisher.Release()
		got := <-done
		if got.err != nil {
			t.Fatalf("GetFeed returned error after blocking: %v", got.err)
		}
		t.Fatal("GetFeed waited for telemetry publisher")
	}

	publisher.Release()
}

func TestRecordCardOpenIsIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 17, 12, 30, 0, 0, time.UTC)
	store := memory.NewStore([]feed.Candidate{testCandidate("animal-1", false, 0.80)})
	publisher := memory.NewPublisher()
	service := feed.NewService(feed.Dependencies{
		Store:       store,
		Publisher:   publisher,
		Clock:       func() time.Time { return now },
		IDGenerator: sequenceIDs("open-1", "event-1"),
	})

	first, err := service.RecordCardOpen(ctx, &feedv1.RecordCardOpenRequest{
		FeedCardId:     "card-1",
		AnimalId:       "animal-1",
		FeedSessionId:  "session-1",
		IdempotencyKey: "open-key",
		Principal:      &commonv1.Principal{ActorId: "user-1"},
	})
	if err != nil {
		t.Fatalf("first RecordCardOpen returned error: %v", err)
	}
	second, err := service.RecordCardOpen(ctx, &feedv1.RecordCardOpenRequest{
		FeedCardId:     "card-1",
		AnimalId:       "animal-1",
		FeedSessionId:  "session-1",
		IdempotencyKey: "open-key",
		Principal:      &commonv1.Principal{ActorId: "user-1"},
	})
	if err != nil {
		t.Fatalf("second RecordCardOpen returned error: %v", err)
	}

	if first.CardOpenId != second.CardOpenId {
		t.Fatalf("idempotent open id changed: %q != %q", first.CardOpenId, second.CardOpenId)
	}
	if got := len(publisher.Events()); got != 1 {
		t.Fatalf("published events = %d, want 1", got)
	}
	if got := first.OpenedAt.AsTime(); !got.Equal(now) {
		t.Fatalf("opened_at = %s, want %s", got, now)
	}
}

func testCandidate(id string, boosted bool, score float64) feed.Candidate {
	return testCandidateWithDistance(id, boosted, score, 0)
}

func testCandidateWithDistance(id string, boosted bool, score float64, distanceKM int32, opts ...func(*feed.Candidate)) feed.Candidate {
	candidate := feed.Candidate{
		Animal: &animalv1.AnimalProfile{
			AnimalId:       id,
			OwnerProfileId: "owner-" + id,
			Name:           id,
			Species:        animalv1.Species_SPECIES_CAT,
			Status:         animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE,
			Visibility:     commonv1.Visibility_VISIBILITY_PUBLIC,
			Boosted:        boosted,
			Audit: &commonv1.AuditMetadata{
				CreatedAt: timestamppb.New(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		OwnerDisplayName:   "Owner " + id,
		OwnerAverageRating: 4.8,
		ScoreComponents:    map[string]float64{"base": score},
		RankingReasons:     []string{"base relevance"},
		DistanceKM:         distanceKM,
	}
	for _, opt := range opts {
		opt(&candidate)
	}
	return candidate
}

func withSpecies(species animalv1.Species) func(*feed.Candidate) {
	return func(candidate *feed.Candidate) {
		candidate.Animal.Species = species
	}
}

func withStatus(status animalv1.AnimalStatus) func(*feed.Candidate) {
	return func(candidate *feed.Candidate) {
		candidate.Animal.Status = status
	}
}

func withVisibility(visibility commonv1.Visibility) func(*feed.Candidate) {
	return func(candidate *feed.Candidate) {
		candidate.Animal.Visibility = visibility
	}
}

func withOwnerProfileID(ownerProfileID string) func(*feed.Candidate) {
	return func(candidate *feed.Candidate) {
		candidate.Animal.OwnerProfileId = ownerProfileID
	}
}

func feedAnimalIDs(cards []*feedv1.FeedCard) []string {
	ids := make([]string, 0, len(cards))
	for _, card := range cards {
		ids = append(ids, card.Animal.AnimalId)
	}
	return ids
}

func animalIDs(candidates []feed.Candidate) []string {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Animal.AnimalId)
	}
	return ids
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d = %q, want %q; full got %v", i, got[i], want[i], got)
		}
	}
}

func waitForTopics(t *testing.T, publisher *memory.Publisher, want []string) {
	t.Helper()
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		got := publisher.Topics()
		if len(got) == len(want) {
			assertStrings(t, got, want)
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	assertStrings(t, publisher.Topics(), want)
}

func sequenceIDs(ids ...string) func() string {
	next := 0
	return func() string {
		if next >= len(ids) {
			return "extra-id"
		}
		id := ids[next]
		next++
		return id
	}
}

type failingPublisher struct {
	err error
}

func (p failingPublisher) Publish(context.Context, string, proto.Message) error {
	return p.err
}

type blockingPublisher struct {
	release chan struct{}
}

func newBlockingPublisher() *blockingPublisher {
	return &blockingPublisher{release: make(chan struct{})}
}

func (p *blockingPublisher) Publish(ctx context.Context, _ string, _ proto.Message) error {
	select {
	case <-p.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *blockingPublisher) Release() {
	select {
	case <-p.release:
	default:
		close(p.release)
	}
}
