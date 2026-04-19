package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	domain "github.com/petmatch/chat-service/internal/domain/chat"
)

const defaultPageSize int32 = 50
const maxPageSize int32 = 100

// Service coordinates chat use cases.
type Service struct {
	conversations domain.ConversationRepository
	messages      domain.MessageRepository
	reads         domain.ReadRepository
	publisher     domain.EventPublisher
	clock         domain.Clock
	ids           domain.IDGenerator
	validate      *validator.Validate
}

// NewService creates a chat application service.
func NewService(
	conversations domain.ConversationRepository,
	messages domain.MessageRepository,
	reads domain.ReadRepository,
	publisher domain.EventPublisher,
	clock domain.Clock,
	ids domain.IDGenerator,
) *Service {
	return &Service{
		conversations: conversations,
		messages:      messages,
		reads:         reads,
		publisher:     publisher,
		clock:         clock,
		ids:           ids,
		validate:      validator.New(),
	}
}

// CreateConversationInput is input for conversation creation.
type CreateConversationInput struct {
	MatchID          string
	AnimalID         string
	AdopterProfileID string `validate:"required"`
	OwnerProfileID   string `validate:"required"`
	IdempotencyKey   string `validate:"required"`
}

// SendMessageInput is input for message creation.
type SendMessageInput struct {
	ConversationID  string             `validate:"required"`
	SenderProfileID string             `validate:"required"`
	Type            domain.MessageType `validate:"required"`
	Text            string
	Attachments     []domain.Photo
	Metadata        map[string]string
	ClientMessageID string `validate:"required"`
	IdempotencyKey  string `validate:"required"`
}

// UpdateMessageInput is input for message updates.
type UpdateMessageInput struct {
	MessageID string `validate:"required"`
	Text      string `validate:"required"`
}

// MarkReadInput is input for read receipt updates.
type MarkReadInput struct {
	ConversationID  string `validate:"required"`
	ReaderProfileID string `validate:"required"`
	UpToMessageID   string `validate:"required"`
}

// CreateConversation creates or returns an idempotently existing conversation.
func (s *Service) CreateConversation(ctx context.Context, input CreateConversationInput) (domain.Conversation, error) {
	if err := s.validate.Struct(input); err != nil {
		return domain.Conversation{}, fmt.Errorf("validate create conversation: %w", err)
	}
	existing, err := s.conversations.GetConversationByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Conversation{}, fmt.Errorf("check conversation idempotency: %w", err)
	}

	conversation, err := domain.NewConversation(domain.CreateConversationCommand{
		ID:               s.ids.NewID(),
		MatchID:          input.MatchID,
		AnimalID:         input.AnimalID,
		AdopterProfileID: input.AdopterProfileID,
		OwnerProfileID:   input.OwnerProfileID,
		IdempotencyKey:   input.IdempotencyKey,
		Now:              s.clock.Now(),
	})
	if err != nil {
		return domain.Conversation{}, err
	}
	created, err := s.conversations.CreateConversation(ctx, conversation)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	if publishErr := s.publisher.Publish(ctx, domain.Event{
		ID:             s.ids.NewID(),
		Type:           domain.EventTypeConversationCreated,
		OccurredAt:     s.clock.Now(),
		IdempotencyKey: input.IdempotencyKey,
		PartitionKey:   created.ID,
		Conversation:   &created,
	}); publishErr != nil {
		return domain.Conversation{}, fmt.Errorf("publish conversation created: %w", publishErr)
	}
	return created, nil
}

// GetConversation returns a conversation by ID.
func (s *Service) GetConversation(ctx context.Context, id string) (domain.Conversation, error) {
	conversation, err := s.conversations.GetConversation(ctx, id)
	if err != nil {
		return domain.Conversation{}, fmt.Errorf("get conversation: %w", err)
	}
	return conversation, nil
}

// ListConversations returns participant conversations.
func (s *Service) ListConversations(ctx context.Context, filter domain.ListConversationsFilter) ([]domain.Conversation, string, error) {
	filter.PageSize = normalizePageSize(filter.PageSize)
	items, token, err := s.conversations.ListConversations(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("list conversations: %w", err)
	}
	return items, token, nil
}

// SendMessage creates or returns an idempotently existing message.
func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (domain.Message, error) {
	if err := s.validate.Struct(input); err != nil {
		return domain.Message{}, fmt.Errorf("validate send message: %w", err)
	}
	existing, err := s.messages.GetMessageByIdempotencyKey(ctx, input.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Message{}, fmt.Errorf("check message idempotency: %w", err)
	}
	conversation, err := s.conversations.GetConversation(ctx, input.ConversationID)
	if err != nil {
		return domain.Message{}, fmt.Errorf("get conversation for send: %w", err)
	}
	metadata := make(map[string]string, len(input.Metadata)+1)
	for key, value := range input.Metadata {
		metadata[key] = value
	}
	if recipientProfileID := conversation.CounterpartProfileID(input.SenderProfileID); recipientProfileID != "" {
		metadata["recipient_profile_id"] = recipientProfileID
	}
	message, err := domain.NewMessage(domain.SendMessageCommand{
		ID:              s.ids.NewID(),
		Conversation:    conversation,
		SenderProfileID: input.SenderProfileID,
		Type:            input.Type,
		Text:            input.Text,
		Attachments:     input.Attachments,
		Metadata:        metadata,
		ClientMessageID: input.ClientMessageID,
		IdempotencyKey:  input.IdempotencyKey,
		Now:             s.clock.Now(),
	})
	if err != nil {
		return domain.Message{}, err
	}
	created, err := s.messages.CreateMessage(ctx, message)
	if err != nil {
		return domain.Message{}, fmt.Errorf("create message: %w", err)
	}
	if publishErr := s.publisher.Publish(ctx, domain.Event{
		ID:             s.ids.NewID(),
		Type:           domain.EventTypeMessageSent,
		OccurredAt:     s.clock.Now(),
		IdempotencyKey: input.IdempotencyKey,
		PartitionKey:   created.ConversationID,
		Message:        &created,
	}); publishErr != nil {
		return domain.Message{}, fmt.Errorf("publish message sent: %w", publishErr)
	}
	return created, nil
}

// ListMessages returns messages for a conversation.
func (s *Service) ListMessages(ctx context.Context, filter domain.ListMessagesFilter) ([]domain.Message, string, error) {
	filter.PageSize = normalizePageSize(filter.PageSize)
	if _, err := s.conversations.GetConversation(ctx, filter.ConversationID); err != nil {
		return nil, "", fmt.Errorf("get conversation for messages: %w", err)
	}
	items, token, err := s.messages.ListMessages(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("list messages: %w", err)
	}
	return items, token, nil
}

// UpdateMessage updates mutable message fields.
func (s *Service) UpdateMessage(ctx context.Context, input UpdateMessageInput) (domain.Message, error) {
	if err := s.validate.Struct(input); err != nil {
		return domain.Message{}, fmt.Errorf("validate update message: %w", err)
	}
	message, err := s.messages.UpdateMessage(ctx, domain.Message{ID: input.MessageID, Text: input.Text})
	if err != nil {
		return domain.Message{}, fmt.Errorf("update message: %w", err)
	}
	return message, nil
}

// MarkRead updates read state and emits a read event.
func (s *Service) MarkRead(ctx context.Context, input MarkReadInput) (domain.ReadReceipt, error) {
	if err := s.validate.Struct(input); err != nil {
		return domain.ReadReceipt{}, fmt.Errorf("validate mark read: %w", err)
	}
	receipt := domain.ReadReceipt{
		ConversationID:  input.ConversationID,
		ReaderProfileID: input.ReaderProfileID,
		UpToMessageID:   input.UpToMessageID,
		ReadAt:          s.clock.Now(),
	}
	updated, err := s.reads.MarkRead(ctx, receipt)
	if err != nil {
		return domain.ReadReceipt{}, fmt.Errorf("mark read: %w", err)
	}
	if publishErr := s.publisher.Publish(ctx, domain.Event{
		ID:           s.ids.NewID(),
		Type:         domain.EventTypeMessageRead,
		OccurredAt:   updated.ReadAt,
		PartitionKey: updated.ConversationID,
		ReadReceipt:  &updated,
	}); publishErr != nil {
		return domain.ReadReceipt{}, fmt.Errorf("publish message read: %w", publishErr)
	}
	return updated, nil
}

func normalizePageSize(size int32) int32 {
	if size <= 0 {
		return defaultPageSize
	}
	if size > maxPageSize {
		return maxPageSize
	}
	return size
}
