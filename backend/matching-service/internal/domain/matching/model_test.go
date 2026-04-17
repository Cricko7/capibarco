package matching_test

import (
	"errors"
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/domain/matching"
	"github.com/stretchr/testify/require"
)

func TestNewSwipeValidatesRequiredFields(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		cmd  matching.RecordSwipeCommand
	}{
		{
			name: "missing actor id",
			cmd: matching.RecordSwipeCommand{
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionRight,
				IdempotencyKey: "idem-1",
			},
		},
		{
			name: "missing animal id",
			cmd: matching.RecordSwipeCommand{
				ActorID:        "actor-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionRight,
				IdempotencyKey: "idem-1",
			},
		},
		{
			name: "missing owner profile id",
			cmd: matching.RecordSwipeCommand{
				ActorID:        "actor-1",
				AnimalID:       "animal-1",
				Direction:      matching.SwipeDirectionRight,
				IdempotencyKey: "idem-1",
			},
		},
		{
			name: "unspecified direction",
			cmd: matching.RecordSwipeCommand{
				ActorID:        "actor-1",
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				IdempotencyKey: "idem-1",
			},
		},
		{
			name: "missing idempotency key",
			cmd: matching.RecordSwipeCommand{
				ActorID:        "actor-1",
				AnimalID:       "animal-1",
				OwnerProfileID: "owner-1",
				Direction:      matching.SwipeDirectionRight,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := matching.NewSwipe("swipe-1", tt.cmd, now)
			require.ErrorIs(t, err, matching.ErrInvalidArgument)
		})
	}
}

func TestNewSwipeCreatesSwipe(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	cmd := matching.RecordSwipeCommand{
		ActorID:        "actor-1",
		ActorIsGuest:   true,
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionLeft,
		FeedCardID:     "card-1",
		FeedSessionID:  "session-1",
		IdempotencyKey: "idem-1",
	}

	swipe, err := matching.NewSwipe("swipe-1", cmd, now)

	require.NoError(t, err)
	require.Equal(t, "swipe-1", swipe.ID)
	require.Equal(t, "actor-1", swipe.ActorID)
	require.True(t, swipe.ActorIsGuest)
	require.Equal(t, matching.SwipeDirectionLeft, swipe.Direction)
	require.Equal(t, "card-1", swipe.FeedCardID)
	require.Equal(t, "session-1", swipe.FeedSessionID)
	require.Equal(t, now, swipe.SwipedAt)
}

func TestNewMatchFromSwipeRequiresRightSwipe(t *testing.T) {
	swipe := matching.Swipe{
		ID:             "swipe-1",
		ActorID:        "adopter-1",
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionLeft,
		SwipedAt:       time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
	}

	_, err := matching.NewMatchFromSwipe("match-1", swipe)

	require.True(t, errors.Is(err, matching.ErrInvalidArgument))
}

func TestNewMatchFromSwipeCreatesActiveMatch(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	swipe := matching.Swipe{
		ID:             "swipe-1",
		ActorID:        "adopter-1",
		AnimalID:       "animal-1",
		OwnerProfileID: "owner-1",
		Direction:      matching.SwipeDirectionRight,
		SwipedAt:       now,
	}

	match, err := matching.NewMatchFromSwipe("match-1", swipe)

	require.NoError(t, err)
	require.Equal(t, "match-1", match.ID)
	require.Equal(t, "adopter-1", match.AdopterProfileID)
	require.Equal(t, "owner-1", match.OwnerProfileID)
	require.Equal(t, matching.MatchStatusActive, match.Status)
	require.Equal(t, now, match.CreatedAt)
	require.Equal(t, now, match.UpdatedAt)
	require.Empty(t, match.ConversationID)
}
