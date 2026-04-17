package matching_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/petmatch/petmatch/internal/app/matching"
	"github.com/petmatch/petmatch/internal/domain/matching"
	"github.com/stretchr/testify/require"
)

func TestServiceRecordSwipeCallsChatForNewRightSwipe(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	store := &fakeStore{
		recordResult: app.RecordSwipeResult{
			Swipe: matching.Swipe{
				ID:             "swipe-1",
				ActorID:        "adopter-1",
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionRight,
				SwipedAt:       clock.now,
			},
			Match: &matching.Match{
				ID:               "match-1",
				AnimalID:         "animal-1",
				AdopterProfileID: "adopter-1",
				OwnerProfileID:   "owner-1",
				Status:           matching.MatchStatusActive,
				CreatedAt:        clock.now,
				UpdatedAt:        clock.now,
			},
			CreatedMatch: true,
		},
	}
	chat := &fakeChatClient{conversationID: "conversation-1", created: true}
	service := app.NewService(store, chat, clock)

	result, err := service.RecordSwipe(ctx, app.RecordSwipeCommand{
		ActorID:        "adopter-1",
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionRight,
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	require.True(t, result.ChatCreated)
	require.Equal(t, "conversation-1", result.ConversationID)
	require.Equal(t, "conversation-1", result.Match.ConversationID)
	require.Len(t, chat.calls, 1)
	require.Equal(t, "match-1", chat.calls[0].match.ID)
	require.Equal(t, "idem-1", chat.calls[0].idempotencyKey)
	require.Equal(t, "conversation-1", store.updatedConversationID)
}

func TestServiceRecordSwipeDoesNotCallChatForLeftSwipe(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	store := &fakeStore{
		recordResult: app.RecordSwipeResult{
			Swipe: matching.Swipe{
				ID:             "swipe-1",
				ActorID:        "adopter-1",
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionLeft,
				SwipedAt:       clock.now,
			},
		},
	}
	chat := &fakeChatClient{}
	service := app.NewService(store, chat, clock)

	result, err := service.RecordSwipe(ctx, app.RecordSwipeCommand{
		ActorID:        "adopter-1",
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionLeft,
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	require.False(t, result.ChatCreated)
	require.Nil(t, result.Match)
	require.Empty(t, chat.calls)
}

func TestServiceRecordSwipeReturnsIdempotentResultWithoutCallingChatAgain(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	store := &fakeStore{
		recordResult: app.RecordSwipeResult{
			Swipe: matching.Swipe{
				ID:             "swipe-1",
				ActorID:        "adopter-1",
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionRight,
				SwipedAt:       clock.now,
			},
			Match: &matching.Match{
				ID:               "match-1",
				AnimalID:         "animal-1",
				AdopterProfileID: "adopter-1",
				OwnerProfileID:   "owner-1",
				ConversationID:   "conversation-1",
				Status:           matching.MatchStatusActive,
				CreatedAt:        clock.now,
				UpdatedAt:        clock.now,
			},
			Idempotent: true,
		},
	}
	chat := &fakeChatClient{}
	service := app.NewService(store, chat, clock)

	result, err := service.RecordSwipe(ctx, app.RecordSwipeCommand{
		ActorID:        "adopter-1",
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionRight,
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	require.True(t, result.ChatCreated)
	require.Equal(t, "conversation-1", result.ConversationID)
	require.Empty(t, chat.calls)
}

func TestServiceRecordSwipeKeepsMatchWhenChatFails(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	store := &fakeStore{
		recordResult: app.RecordSwipeResult{
			Swipe: matching.Swipe{
				ID:             "swipe-1",
				ActorID:        "adopter-1",
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionRight,
				SwipedAt:       clock.now,
			},
			Match: &matching.Match{
				ID:               "match-1",
				AnimalID:         "animal-1",
				AdopterProfileID: "adopter-1",
				OwnerProfileID:   "owner-1",
				Status:           matching.MatchStatusActive,
				CreatedAt:        clock.now,
				UpdatedAt:        clock.now,
			},
			CreatedMatch: true,
		},
	}
	chat := &fakeChatClient{err: errors.New("chat unavailable")}
	service := app.NewService(store, chat, clock)

	result, err := service.RecordSwipe(ctx, app.RecordSwipeCommand{
		ActorID:        "adopter-1",
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionRight,
		IdempotencyKey: "idem-1",
	})

	require.NoError(t, err)
	require.False(t, result.ChatCreated)
	require.Empty(t, result.ConversationID)
	require.NotNil(t, result.Match)
	require.Len(t, chat.calls, 1)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fakeStore struct {
	recordResult          app.RecordSwipeResult
	recordErr             error
	updatedMatchID        string
	updatedConversationID string
}

func (s *fakeStore) RecordSwipe(ctx context.Context, cmd app.RecordSwipeCommand) (app.RecordSwipeResult, error) {
	return s.recordResult, s.recordErr
}

func (s *fakeStore) UpdateMatchConversation(ctx context.Context, matchID string, conversationID string) error {
	s.updatedMatchID = matchID
	s.updatedConversationID = conversationID
	return nil
}

func (s *fakeStore) GetSwipe(ctx context.Context, swipeID string) (matching.Swipe, error) {
	return matching.Swipe{}, nil
}

func (s *fakeStore) ListSwipes(ctx context.Context, query app.ListSwipesQuery) (app.ListSwipesResult, error) {
	return app.ListSwipesResult{}, nil
}

func (s *fakeStore) GetMatch(ctx context.Context, matchID string) (matching.Match, error) {
	return matching.Match{}, nil
}

func (s *fakeStore) ListMatches(ctx context.Context, query app.ListMatchesQuery) (app.ListMatchesResult, error) {
	return app.ListMatchesResult{}, nil
}

func (s *fakeStore) ArchiveMatchesByAnimal(ctx context.Context, animalID string, reason string) ([]matching.Match, error) {
	return nil, nil
}

func (s *fakeStore) SetAnimalAvailability(ctx context.Context, animalID string, ownerProfileID string, available bool) error {
	return nil
}

type fakeChatClient struct {
	conversationID string
	created        bool
	err            error
	calls          []fakeChatCall
}

type fakeChatCall struct {
	match          matching.Match
	idempotencyKey string
}

func (c *fakeChatClient) CreateConversation(ctx context.Context, match matching.Match, idempotencyKey string) (app.ChatConversationResult, error) {
	c.calls = append(c.calls, fakeChatCall{match: match, idempotencyKey: idempotencyKey})
	return app.ChatConversationResult{ConversationID: c.conversationID, Created: c.created}, c.err
}
