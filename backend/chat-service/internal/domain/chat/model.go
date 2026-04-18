package chat

import (
	"fmt"
	"strings"
	"time"
)

// ConversationStatus describes the lifecycle state of a conversation.
type ConversationStatus int16

const (
	ConversationStatusUnspecified ConversationStatus = iota
	ConversationStatusActive
	ConversationStatusArchived
	ConversationStatusBlocked
)

// MessageType describes the semantic kind of a message.
type MessageType int16

const (
	MessageTypeUnspecified MessageType = iota
	MessageTypeText
	MessageTypeImage
	MessageTypeSystem
	MessageTypeAdoptionStatus
)

// Photo is the chat-domain representation of a shared image asset.
type Photo struct {
	ID          string
	URL         string
	Blurhash    string
	Width       int32
	Height      int32
	ContentType string
	SortOrder   int32
	CreatedAt   time.Time
}

// Conversation is an adoption conversation between an adopter and an owner.
type Conversation struct {
	ID               string
	MatchID          string
	AnimalID         string
	AdopterProfileID string
	OwnerProfileID   string
	Status           ConversationStatus
	LastMessage      *Message
	UnreadCount      int32
	IdempotencyKey   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Message is a persisted chat message.
type Message struct {
	ID              string
	ConversationID  string
	SenderProfileID string
	Type            MessageType
	Text            string
	Attachments     []Photo
	Metadata        map[string]string
	ClientMessageID string
	IdempotencyKey  string
	SentAt          time.Time
	EditedAt        *time.Time
	DeletedAt       *time.Time
}

// ReadReceipt stores a participant's read position.
type ReadReceipt struct {
	ConversationID  string
	ReaderProfileID string
	UpToMessageID   string
	ReadAt          time.Time
}

// CreateConversationCommand carries validated input for conversation creation.
type CreateConversationCommand struct {
	ID               string
	MatchID          string
	AnimalID         string
	AdopterProfileID string
	OwnerProfileID   string
	IdempotencyKey   string
	Now              time.Time
}

// SendMessageCommand carries validated input for message creation.
type SendMessageCommand struct {
	ID              string
	Conversation    Conversation
	SenderProfileID string
	Type            MessageType
	Text            string
	Attachments     []Photo
	Metadata        map[string]string
	ClientMessageID string
	IdempotencyKey  string
	Now             time.Time
}

// NewConversation creates a conversation and enforces participant invariants.
func NewConversation(cmd CreateConversationCommand) (Conversation, error) {
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return Conversation{}, fmt.Errorf("create conversation: %w", ErrMissingIdempotencyKey)
	}
	if strings.TrimSpace(cmd.ID) == "" ||
		strings.TrimSpace(cmd.AdopterProfileID) == "" ||
		strings.TrimSpace(cmd.OwnerProfileID) == "" {
		return Conversation{}, fmt.Errorf("create conversation: %w", ErrInvalidParticipant)
	}
	if cmd.AdopterProfileID == cmd.OwnerProfileID {
		return Conversation{}, fmt.Errorf("create conversation: %w", ErrInvalidParticipant)
	}
	if cmd.Now.IsZero() {
		cmd.Now = time.Now().UTC()
	}

	return Conversation{
		ID:               strings.TrimSpace(cmd.ID),
		MatchID:          strings.TrimSpace(cmd.MatchID),
		AnimalID:         strings.TrimSpace(cmd.AnimalID),
		AdopterProfileID: strings.TrimSpace(cmd.AdopterProfileID),
		OwnerProfileID:   strings.TrimSpace(cmd.OwnerProfileID),
		Status:           ConversationStatusActive,
		IdempotencyKey:   strings.TrimSpace(cmd.IdempotencyKey),
		CreatedAt:        cmd.Now,
		UpdatedAt:        cmd.Now,
	}, nil
}

// NewMessage creates a message and enforces sender/content invariants.
func NewMessage(cmd SendMessageCommand) (Message, error) {
	if strings.TrimSpace(cmd.IdempotencyKey) == "" {
		return Message{}, fmt.Errorf("send message: %w", ErrMissingIdempotencyKey)
	}
	if cmd.Conversation.Status != ConversationStatusActive {
		return Message{}, fmt.Errorf("send message: %w", ErrConversationClosed)
	}
	if !cmd.Conversation.HasParticipant(cmd.SenderProfileID) {
		return Message{}, fmt.Errorf("send message: %w", ErrForbidden)
	}
	if strings.TrimSpace(cmd.ID) == "" || strings.TrimSpace(cmd.ClientMessageID) == "" {
		return Message{}, fmt.Errorf("send message: %w", ErrInvalidMessage)
	}
	if cmd.Type == MessageTypeUnspecified {
		return Message{}, fmt.Errorf("send message: %w", ErrInvalidMessage)
	}
	if cmd.Type == MessageTypeText && strings.TrimSpace(cmd.Text) == "" {
		return Message{}, fmt.Errorf("send message: %w", ErrInvalidMessage)
	}
	if len(cmd.Attachments) == 0 && cmd.Type == MessageTypeImage {
		return Message{}, fmt.Errorf("send message: %w", ErrInvalidMessage)
	}
	if cmd.Now.IsZero() {
		cmd.Now = time.Now().UTC()
	}
	metadata := make(map[string]string, len(cmd.Metadata))
	for key, value := range cmd.Metadata {
		metadata[key] = value
	}

	return Message{
		ID:              strings.TrimSpace(cmd.ID),
		ConversationID:  cmd.Conversation.ID,
		SenderProfileID: strings.TrimSpace(cmd.SenderProfileID),
		Type:            cmd.Type,
		Text:            strings.TrimSpace(cmd.Text),
		Attachments:     append([]Photo(nil), cmd.Attachments...),
		Metadata:        metadata,
		ClientMessageID: strings.TrimSpace(cmd.ClientMessageID),
		IdempotencyKey:  strings.TrimSpace(cmd.IdempotencyKey),
		SentAt:          cmd.Now,
	}, nil
}

// HasParticipant reports whether profileID belongs to the conversation.
func (c Conversation) HasParticipant(profileID string) bool {
	return profileID != "" && (profileID == c.AdopterProfileID || profileID == c.OwnerProfileID)
}

// UpdateText edits a message text.
func (m Message) UpdateText(text string, now time.Time) (Message, error) {
	if strings.TrimSpace(text) == "" {
		return Message{}, fmt.Errorf("update message: %w", ErrInvalidMessage)
	}
	editedAt := now
	m.Text = strings.TrimSpace(text)
	m.EditedAt = &editedAt
	return m, nil
}
