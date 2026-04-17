package chat

import (
	"context"
	"time"
)

// ListConversationsFilter describes conversation pagination and participant filters.
type ListConversationsFilter struct {
	ParticipantProfileID string
	Statuses             []ConversationStatus
	PageSize             int32
	PageToken            string
}

// ListMessagesFilter describes message pagination.
type ListMessagesFilter struct {
	ConversationID string
	PageSize       int32
	PageToken      string
}

// ConversationRepository persists conversations.
type ConversationRepository interface {
	CreateConversation(ctx context.Context, conversation Conversation) (Conversation, error)
	GetConversation(ctx context.Context, id string) (Conversation, error)
	GetConversationByIdempotencyKey(ctx context.Context, key string) (Conversation, error)
	ListConversations(ctx context.Context, filter ListConversationsFilter) ([]Conversation, string, error)
}

// MessageRepository persists messages.
type MessageRepository interface {
	CreateMessage(ctx context.Context, message Message) (Message, error)
	GetMessageByIdempotencyKey(ctx context.Context, key string) (Message, error)
	ListMessages(ctx context.Context, filter ListMessagesFilter) ([]Message, string, error)
	UpdateMessage(ctx context.Context, message Message) (Message, error)
}

// ReadRepository persists read positions.
type ReadRepository interface {
	MarkRead(ctx context.Context, receipt ReadReceipt) (ReadReceipt, error)
}

// EventPublisher publishes chat domain events.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

// Clock abstracts time for deterministic use cases.
type Clock interface {
	Now() time.Time
}

// IDGenerator abstracts ID generation for deterministic use cases.
type IDGenerator interface {
	NewID() string
}
