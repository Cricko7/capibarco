package chat

import "time"

const (
	EventTypeConversationCreated = "chat.conversation_created"
	EventTypeMessageSent         = "chat.message_sent"
	EventTypeMessageRead         = "chat.message_read"
	EventTypeParticipantTyping   = "chat.participant_typing"
)

// Event is a domain event emitted by chat use cases.
type Event struct {
	ID             string
	Type           string
	OccurredAt     time.Time
	TraceID        string
	CorrelationID  string
	IdempotencyKey string
	PartitionKey   string
	Conversation   *Conversation
	Message        *Message
	ReadReceipt    *ReadReceipt
	Typing         *Typing
}

// Typing describes a realtime typing state change.
type Typing struct {
	ConversationID string
	ProfileID      string
	Typing         bool
}
