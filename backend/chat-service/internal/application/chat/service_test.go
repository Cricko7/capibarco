package chat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appchat "github.com/petmatch/chat-service/internal/application/chat"
	"github.com/petmatch/chat-service/internal/domain/chat"
)

func TestServiceCreateConversationIsIdempotent(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	repo := newMemoryRepo()
	ids := fixedIDs{"conversation-1", "event-1"}
	service := appchat.NewService(repo, repo, repo, repo, clock, &ids)

	first, err := service.CreateConversation(ctx, appchat.CreateConversationInput{
		MatchID:          "match-1",
		AnimalID:         "animal-1",
		AdopterProfileID: "adopter",
		OwnerProfileID:   "owner",
		IdempotencyKey:   "idem-conversation",
	})
	if err != nil {
		t.Fatalf("first CreateConversation() error = %v", err)
	}

	second, err := service.CreateConversation(ctx, appchat.CreateConversationInput{
		MatchID:          "match-1",
		AnimalID:         "animal-1",
		AdopterProfileID: "adopter",
		OwnerProfileID:   "owner",
		IdempotencyKey:   "idem-conversation",
	})
	if err != nil {
		t.Fatalf("second CreateConversation() error = %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("idempotent create returned %q then %q", first.ID, second.ID)
	}
	if repo.createdConversations != 1 {
		t.Fatalf("created conversations = %d, want 1", repo.createdConversations)
	}
}

func TestServiceCreateConversationAllowsDirectProfileChat(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)}
	repo := newMemoryRepo()
	ids := fixedIDs{"conversation-1", "event-1"}
	service := appchat.NewService(repo, repo, repo, repo, clock, &ids)

	conversation, err := service.CreateConversation(ctx, appchat.CreateConversationInput{
		AdopterProfileID: "adopter",
		OwnerProfileID:   "owner",
		IdempotencyKey:   "idem-direct-conversation",
	})
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if conversation.MatchID != "" {
		t.Fatalf("MatchID = %q, want empty", conversation.MatchID)
	}
	if conversation.AnimalID != "" {
		t.Fatalf("AnimalID = %q, want empty", conversation.AnimalID)
	}
}

func TestServiceSendMessageRejectsArchivedConversation(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepo()
	repo.conversations["conversation-1"] = chat.Conversation{
		ID:               "conversation-1",
		AdopterProfileID: "adopter",
		OwnerProfileID:   "owner",
		Status:           chat.ConversationStatusArchived,
	}
	ids := fixedIDs{"message-1", "event-1"}
	service := appchat.NewService(repo, repo, repo, repo, fixedClock{now: time.Now()}, &ids)

	_, err := service.SendMessage(ctx, appchat.SendMessageInput{
		ConversationID:  "conversation-1",
		SenderProfileID: "adopter",
		Type:            chat.MessageTypeText,
		Text:            "hello",
		ClientMessageID: "client-1",
		IdempotencyKey:  "idem-message",
	})
	if !errors.Is(err, chat.ErrConversationClosed) {
		t.Fatalf("SendMessage() error = %v, want %v", err, chat.ErrConversationClosed)
	}
}

func TestServiceListMessagesReturnsNotFoundForMissingConversation(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryRepo()
	service := appchat.NewService(repo, repo, repo, repo, fixedClock{now: time.Now()}, &fixedIDs{})

	_, _, err := service.ListMessages(ctx, chat.ListMessagesFilter{ConversationID: "missing-conversation"})
	if !errors.Is(err, chat.ErrNotFound) {
		t.Fatalf("ListMessages() error = %v, want %v", err, chat.ErrNotFound)
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type fixedIDs []string

func (g *fixedIDs) NewID() string {
	if len(*g) == 0 {
		return "fallback-id"
	}
	id := (*g)[0]
	*g = (*g)[1:]
	return id
}

type memoryRepo struct {
	conversations        map[string]chat.Conversation
	conversationsByKey   map[string]chat.Conversation
	messages             map[string]chat.Message
	messagesByKey        map[string]chat.Message
	createdConversations int
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		conversations:      make(map[string]chat.Conversation),
		conversationsByKey: make(map[string]chat.Conversation),
		messages:           make(map[string]chat.Message),
		messagesByKey:      make(map[string]chat.Message),
	}
}

func (r *memoryRepo) CreateConversation(_ context.Context, conversation chat.Conversation) (chat.Conversation, error) {
	r.createdConversations++
	r.conversations[conversation.ID] = conversation
	r.conversationsByKey[conversation.IdempotencyKey] = conversation
	return conversation, nil
}

func (r *memoryRepo) GetConversation(_ context.Context, id string) (chat.Conversation, error) {
	conversation, ok := r.conversations[id]
	if !ok {
		return chat.Conversation{}, chat.ErrNotFound
	}
	return conversation, nil
}

func (r *memoryRepo) GetConversationByIdempotencyKey(_ context.Context, key string) (chat.Conversation, error) {
	conversation, ok := r.conversationsByKey[key]
	if !ok {
		return chat.Conversation{}, chat.ErrNotFound
	}
	return conversation, nil
}

func (r *memoryRepo) ListConversations(_ context.Context, _ chat.ListConversationsFilter) ([]chat.Conversation, string, error) {
	return nil, "", nil
}

func (r *memoryRepo) CreateMessage(_ context.Context, message chat.Message) (chat.Message, error) {
	r.messages[message.ID] = message
	r.messagesByKey[message.IdempotencyKey] = message
	return message, nil
}

func (r *memoryRepo) GetMessageByIdempotencyKey(_ context.Context, key string) (chat.Message, error) {
	message, ok := r.messagesByKey[key]
	if !ok {
		return chat.Message{}, chat.ErrNotFound
	}
	return message, nil
}

func (r *memoryRepo) ListMessages(_ context.Context, _ chat.ListMessagesFilter) ([]chat.Message, string, error) {
	return nil, "", nil
}

func (r *memoryRepo) UpdateMessage(_ context.Context, message chat.Message) (chat.Message, error) {
	r.messages[message.ID] = message
	return message, nil
}

func (r *memoryRepo) MarkRead(_ context.Context, receipt chat.ReadReceipt) (chat.ReadReceipt, error) {
	return receipt, nil
}

func (r *memoryRepo) Publish(_ context.Context, _ chat.Event) error {
	return nil
}
