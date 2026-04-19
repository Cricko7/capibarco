package matching

import (
	"context"
	"fmt"
	"strings"

	domain "github.com/petmatch/petmatch/internal/domain/matching"
)

// Service orchestrates matching use cases.
type Service struct {
	store Store
	chat  ChatClient
	clock Clock
}

// NewService creates a matching application service.
func NewService(store Store, chat ChatClient, clock Clock) *Service {
	if clock == nil {
		clock = SystemClock{}
	}
	return &Service{store: store, chat: chat, clock: clock}
}

// RecordSwipe records a swipe and creates a match notification event for new right swipes.
func (s *Service) RecordSwipe(ctx context.Context, cmd RecordSwipeCommand) (RecordSwipeResult, error) {
	if s == nil || s.store == nil {
		return RecordSwipeResult{}, fmt.Errorf("%w: matching store is not configured", domain.ErrInvalidArgument)
	}
	if err := validateRecordSwipeInput(cmd); err != nil {
		return RecordSwipeResult{}, err
	}

	result, err := s.store.RecordSwipe(ctx, normalizeRecordSwipeCommand(cmd))
	if err != nil {
		return RecordSwipeResult{}, fmt.Errorf("record swipe: %w", err)
	}
	if result.Match == nil {
		return result, nil
	}
	if result.Match.ConversationID != "" {
		result.ChatCreated = true
		result.ConversationID = result.Match.ConversationID
		return result, nil
	}
	return result, nil
}

// GetSwipe returns one swipe by id.
func (s *Service) GetSwipe(ctx context.Context, swipeID string) (domain.Swipe, error) {
	if strings.TrimSpace(swipeID) == "" {
		return domain.Swipe{}, fmt.Errorf("%w: swipe id is required", domain.ErrInvalidArgument)
	}
	swipe, err := s.store.GetSwipe(ctx, strings.TrimSpace(swipeID))
	if err != nil {
		return domain.Swipe{}, fmt.Errorf("get swipe: %w", err)
	}
	return swipe, nil
}

// ListSwipes returns swipes for an actor with cursor pagination.
func (s *Service) ListSwipes(ctx context.Context, query ListSwipesQuery) (ListSwipesResult, error) {
	if strings.TrimSpace(query.ActorID) == "" {
		return ListSwipesResult{}, fmt.Errorf("%w: actor id is required", domain.ErrInvalidArgument)
	}
	query.ActorID = strings.TrimSpace(query.ActorID)
	query.Page = normalizePage(query.Page)
	result, err := s.store.ListSwipes(ctx, query)
	if err != nil {
		return ListSwipesResult{}, fmt.Errorf("list swipes: %w", err)
	}
	return result, nil
}

// GetMatch returns one match by id.
func (s *Service) GetMatch(ctx context.Context, matchID string) (domain.Match, error) {
	if strings.TrimSpace(matchID) == "" {
		return domain.Match{}, fmt.Errorf("%w: match id is required", domain.ErrInvalidArgument)
	}
	match, err := s.store.GetMatch(ctx, strings.TrimSpace(matchID))
	if err != nil {
		return domain.Match{}, fmt.Errorf("get match: %w", err)
	}
	return match, nil
}

// ListMatches returns matches for an adopter or owner profile.
func (s *Service) ListMatches(ctx context.Context, query ListMatchesQuery) (ListMatchesResult, error) {
	if strings.TrimSpace(query.ParticipantProfileID) == "" {
		return ListMatchesResult{}, fmt.Errorf("%w: participant profile id is required", domain.ErrInvalidArgument)
	}
	query.ParticipantProfileID = strings.TrimSpace(query.ParticipantProfileID)
	query.Page = normalizePage(query.Page)
	result, err := s.store.ListMatches(ctx, query)
	if err != nil {
		return ListMatchesResult{}, fmt.Errorf("list matches: %w", err)
	}
	return result, nil
}

// HandleAnimalArchived archives active matches for an animal profile archive event.
func (s *Service) HandleAnimalArchived(ctx context.Context, animalID string, reason string) ([]domain.Match, error) {
	if strings.TrimSpace(animalID) == "" {
		return nil, fmt.Errorf("%w: animal id is required", domain.ErrInvalidArgument)
	}
	matches, err := s.store.ArchiveMatchesByAnimal(ctx, strings.TrimSpace(animalID), strings.TrimSpace(reason))
	if err != nil {
		return nil, fmt.Errorf("archive matches by animal: %w", err)
	}
	return matches, nil
}

// HandleAnimalStatusChanged updates availability and archives matches for unavailable animals.
func (s *Service) HandleAnimalStatusChanged(ctx context.Context, animalID string, ownerProfileID string, available bool, reason string) ([]domain.Match, error) {
	if strings.TrimSpace(animalID) == "" {
		return nil, fmt.Errorf("%w: animal id is required", domain.ErrInvalidArgument)
	}
	if err := s.store.SetAnimalAvailability(ctx, strings.TrimSpace(animalID), strings.TrimSpace(ownerProfileID), available); err != nil {
		return nil, fmt.Errorf("set animal availability: %w", err)
	}
	if available {
		return nil, nil
	}
	return s.HandleAnimalArchived(ctx, animalID, reason)
}

// HandleConversationCreated backfills a conversation id after asynchronous chat creation.
func (s *Service) HandleConversationCreated(ctx context.Context, matchID string, conversationID string) error {
	if strings.TrimSpace(matchID) == "" {
		return fmt.Errorf("%w: match id is required", domain.ErrInvalidArgument)
	}
	if strings.TrimSpace(conversationID) == "" {
		return fmt.Errorf("%w: conversation id is required", domain.ErrInvalidArgument)
	}
	if err := s.store.UpdateMatchConversation(ctx, strings.TrimSpace(matchID), strings.TrimSpace(conversationID)); err != nil {
		return fmt.Errorf("backfill match conversation: %w", err)
	}
	return nil
}

func validateRecordSwipeInput(cmd RecordSwipeCommand) error {
	_, err := domain.NewSwipe("validation-swipe-id", cmd, SystemClock{}.Now())
	return err
}

func normalizeRecordSwipeCommand(cmd RecordSwipeCommand) RecordSwipeCommand {
	cmd.ActorID = strings.TrimSpace(cmd.ActorID)
	cmd.AnimalID = strings.TrimSpace(cmd.AnimalID)
	cmd.OwnerProfileID = strings.TrimSpace(cmd.OwnerProfileID)
	cmd.FeedCardID = strings.TrimSpace(cmd.FeedCardID)
	cmd.FeedSessionID = strings.TrimSpace(cmd.FeedSessionID)
	cmd.IdempotencyKey = strings.TrimSpace(cmd.IdempotencyKey)
	return cmd
}

func normalizePage(page PageRequest) PageRequest {
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
	page.PageToken = strings.TrimSpace(page.PageToken)
	return page
}
