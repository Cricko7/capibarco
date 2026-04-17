package matching

import (
	"fmt"
	"strings"
	"time"
)

// SwipeDirection is the adopter decision for a feed animal card.
type SwipeDirection int

const (
	SwipeDirectionUnspecified SwipeDirection = iota
	SwipeDirectionLeft
	SwipeDirectionRight
)

// MatchStatus describes the lifecycle state of a match.
type MatchStatus int

const (
	MatchStatusUnspecified MatchStatus = iota
	MatchStatusActive
	MatchStatusArchived
	MatchStatusBlocked
)

// RecordSwipeCommand is the domain input needed to record a swipe.
type RecordSwipeCommand struct {
	ActorID        string
	ActorIsGuest   bool
	AnimalID       string
	OwnerProfileID string
	Direction      SwipeDirection
	FeedCardID     string
	FeedSessionID  string
	IdempotencyKey string
}

// Swipe is an immutable one-sided adopter decision for an animal.
type Swipe struct {
	ID             string
	ActorID        string
	ActorIsGuest   bool
	AnimalID       string
	OwnerProfileID string
	Direction      SwipeDirection
	FeedCardID     string
	FeedSessionID  string
	SwipedAt       time.Time
}

// Match is created immediately from a right swipe.
type Match struct {
	ID               string
	AnimalID         string
	AdopterProfileID string
	OwnerProfileID   string
	ConversationID   string
	Status           MatchStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewSwipe validates command data and creates a swipe.
func NewSwipe(id string, cmd RecordSwipeCommand, now time.Time) (Swipe, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Swipe{}, fmt.Errorf("%w: swipe id is required", ErrInvalidArgument)
	}
	if err := validateRecordSwipeCommand(cmd); err != nil {
		return Swipe{}, err
	}
	if now.IsZero() {
		return Swipe{}, fmt.Errorf("%w: swiped_at is required", ErrInvalidArgument)
	}

	return Swipe{
		ID:             id,
		ActorID:        strings.TrimSpace(cmd.ActorID),
		ActorIsGuest:   cmd.ActorIsGuest,
		AnimalID:       strings.TrimSpace(cmd.AnimalID),
		OwnerProfileID: strings.TrimSpace(cmd.OwnerProfileID),
		Direction:      cmd.Direction,
		FeedCardID:     strings.TrimSpace(cmd.FeedCardID),
		FeedSessionID:  strings.TrimSpace(cmd.FeedSessionID),
		SwipedAt:       now.UTC(),
	}, nil
}

// NewMatchFromSwipe creates an active match from a right swipe.
func NewMatchFromSwipe(id string, swipe Swipe) (Match, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Match{}, fmt.Errorf("%w: match id is required", ErrInvalidArgument)
	}
	if swipe.Direction != SwipeDirectionRight {
		return Match{}, fmt.Errorf("%w: match requires a right swipe", ErrInvalidArgument)
	}
	if strings.TrimSpace(swipe.ActorID) == "" || strings.TrimSpace(swipe.AnimalID) == "" || strings.TrimSpace(swipe.OwnerProfileID) == "" {
		return Match{}, fmt.Errorf("%w: swipe is incomplete", ErrInvalidArgument)
	}
	if swipe.SwipedAt.IsZero() {
		return Match{}, fmt.Errorf("%w: swipe time is required", ErrInvalidArgument)
	}

	now := swipe.SwipedAt.UTC()
	return Match{
		ID:               id,
		AnimalID:         strings.TrimSpace(swipe.AnimalID),
		AdopterProfileID: strings.TrimSpace(swipe.ActorID),
		OwnerProfileID:   strings.TrimSpace(swipe.OwnerProfileID),
		Status:           MatchStatusActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// WithConversation returns a copy of match with a conversation id attached.
func (m Match) WithConversation(conversationID string, now time.Time) (Match, error) {
	conversationID = strings.TrimSpace(conversationID)
	if strings.TrimSpace(m.ID) == "" {
		return Match{}, fmt.Errorf("%w: match id is required", ErrInvalidArgument)
	}
	if conversationID == "" {
		return Match{}, fmt.Errorf("%w: conversation id is required", ErrInvalidArgument)
	}
	if now.IsZero() {
		return Match{}, fmt.Errorf("%w: updated_at is required", ErrInvalidArgument)
	}
	m.ConversationID = conversationID
	m.UpdatedAt = now.UTC()
	return m, nil
}

// Archive returns an archived copy of match.
func (m Match) Archive(now time.Time) (Match, error) {
	if strings.TrimSpace(m.ID) == "" {
		return Match{}, fmt.Errorf("%w: match id is required", ErrInvalidArgument)
	}
	if now.IsZero() {
		return Match{}, fmt.Errorf("%w: updated_at is required", ErrInvalidArgument)
	}
	m.Status = MatchStatusArchived
	m.UpdatedAt = now.UTC()
	return m, nil
}

func validateRecordSwipeCommand(cmd RecordSwipeCommand) error {
	switch {
	case strings.TrimSpace(cmd.ActorID) == "":
		return fmt.Errorf("%w: actor id is required", ErrInvalidArgument)
	case strings.TrimSpace(cmd.AnimalID) == "":
		return fmt.Errorf("%w: animal id is required", ErrInvalidArgument)
	case strings.TrimSpace(cmd.OwnerProfileID) == "":
		return fmt.Errorf("%w: owner profile id is required", ErrInvalidArgument)
	case cmd.Direction != SwipeDirectionLeft && cmd.Direction != SwipeDirectionRight:
		return fmt.Errorf("%w: direction must be left or right", ErrInvalidArgument)
	case strings.TrimSpace(cmd.IdempotencyKey) == "":
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidArgument)
	default:
		return nil
	}
}
