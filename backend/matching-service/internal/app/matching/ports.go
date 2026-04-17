// Package matching implements application use cases for matching-service.
package matching

import (
	"context"
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/matching"
)

// Clock provides time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// SystemClock reads wall-clock UTC time.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// Store is the persistence port used by the matching application service.
type Store interface {
	RecordSwipe(ctx context.Context, cmd RecordSwipeCommand) (RecordSwipeResult, error)
	UpdateMatchConversation(ctx context.Context, matchID string, conversationID string) error
	GetSwipe(ctx context.Context, swipeID string) (domain.Swipe, error)
	ListSwipes(ctx context.Context, query ListSwipesQuery) (ListSwipesResult, error)
	GetMatch(ctx context.Context, matchID string) (domain.Match, error)
	ListMatches(ctx context.Context, query ListMatchesQuery) (ListMatchesResult, error)
	ArchiveMatchesByAnimal(ctx context.Context, animalID string, reason string) ([]domain.Match, error)
	SetAnimalAvailability(ctx context.Context, animalID string, ownerProfileID string, available bool) error
}

// ChatClient creates adoption conversations for right-swipe matches.
type ChatClient interface {
	CreateConversation(ctx context.Context, match domain.Match, idempotencyKey string) (ChatConversationResult, error)
}

// RecordSwipeCommand is the application input for RecordSwipe.
type RecordSwipeCommand = domain.RecordSwipeCommand

// RecordSwipeResult is returned after a swipe has been recorded.
type RecordSwipeResult struct {
	Swipe          domain.Swipe
	Match          *domain.Match
	CreatedMatch   bool
	Idempotent     bool
	ChatCreated    bool
	ConversationID string
}

// ChatConversationResult is returned by ChatClient.
type ChatConversationResult struct {
	ConversationID string
	Created        bool
}

// PageRequest contains cursor pagination parameters.
type PageRequest struct {
	PageSize  int32
	PageToken string
}

// PageResponse contains cursor pagination metadata.
type PageResponse struct {
	NextPageToken string
	TotalSize     *int32
}

// ListSwipesQuery filters swipes for one actor.
type ListSwipesQuery struct {
	ActorID    string
	Directions []domain.SwipeDirection
	Page       PageRequest
}

// ListSwipesResult is a page of swipes.
type ListSwipesResult struct {
	Swipes []domain.Swipe
	Page   PageResponse
}

// ListMatchesQuery filters matches for one participant profile.
type ListMatchesQuery struct {
	ParticipantProfileID string
	Statuses             []domain.MatchStatus
	Page                 PageRequest
}

// ListMatchesResult is a page of matches.
type ListMatchesResult struct {
	Matches []domain.Match
	Page    PageResponse
}
