package chat_test

import (
	"errors"
	"testing"
	"time"

	"github.com/petmatch/chat-service/internal/domain/chat"
)

func TestNewConversationValidatesParticipants(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		cmd     chat.CreateConversationCommand
		wantErr error
	}{
		{
			name: "creates active conversation",
			cmd: chat.CreateConversationCommand{
				ID:               "018f53a1-5e98-756f-9fb4-0d28b2ef0c00",
				MatchID:          "match-1",
				AnimalID:         "animal-1",
				AdopterProfileID: "profile-adopter",
				OwnerProfileID:   "profile-owner",
				IdempotencyKey:   "idem-1",
				Now:              now,
			},
		},
		{
			name: "rejects same participant",
			cmd: chat.CreateConversationCommand{
				ID:               "018f53a1-5e98-756f-9fb4-0d28b2ef0c00",
				MatchID:          "match-1",
				AnimalID:         "animal-1",
				AdopterProfileID: "profile-1",
				OwnerProfileID:   "profile-1",
				IdempotencyKey:   "idem-1",
				Now:              now,
			},
			wantErr: chat.ErrInvalidParticipant,
		},
		{
			name: "rejects missing idempotency key",
			cmd: chat.CreateConversationCommand{
				ID:               "018f53a1-5e98-756f-9fb4-0d28b2ef0c00",
				MatchID:          "match-1",
				AnimalID:         "animal-1",
				AdopterProfileID: "profile-adopter",
				OwnerProfileID:   "profile-owner",
				Now:              now,
			},
			wantErr: chat.ErrMissingIdempotencyKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chat.NewConversation(tt.cmd)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewConversation() error = %v, want wrapped %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewConversation() unexpected error = %v", err)
			}
			if got.Status != chat.ConversationStatusActive {
				t.Fatalf("status = %v, want active", got.Status)
			}
			if got.CreatedAt != now || got.UpdatedAt != now {
				t.Fatalf("timestamps = %s/%s, want %s", got.CreatedAt, got.UpdatedAt, now)
			}
		})
	}
}

func TestNewMessageValidatesContentAndSender(t *testing.T) {
	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	conversation := chat.Conversation{
		ID:               "conversation-1",
		AdopterProfileID: "profile-adopter",
		OwnerProfileID:   "profile-owner",
		Status:           chat.ConversationStatusActive,
	}

	tests := []struct {
		name    string
		cmd     chat.SendMessageCommand
		wantErr error
	}{
		{
			name: "accepts text from participant",
			cmd: chat.SendMessageCommand{
				ID:              "message-1",
				Conversation:    conversation,
				SenderProfileID: "profile-adopter",
				Type:            chat.MessageTypeText,
				Text:            "hello",
				ClientMessageID: "client-1",
				IdempotencyKey:  "idem-1",
				Now:             now,
			},
		},
		{
			name: "rejects outsider",
			cmd: chat.SendMessageCommand{
				ID:              "message-1",
				Conversation:    conversation,
				SenderProfileID: "profile-outsider",
				Type:            chat.MessageTypeText,
				Text:            "hello",
				ClientMessageID: "client-1",
				IdempotencyKey:  "idem-1",
				Now:             now,
			},
			wantErr: chat.ErrForbidden,
		},
		{
			name: "rejects blank text message",
			cmd: chat.SendMessageCommand{
				ID:              "message-1",
				Conversation:    conversation,
				SenderProfileID: "profile-adopter",
				Type:            chat.MessageTypeText,
				Text:            "   ",
				ClientMessageID: "client-1",
				IdempotencyKey:  "idem-1",
				Now:             now,
			},
			wantErr: chat.ErrInvalidMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chat.NewMessage(tt.cmd)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NewMessage() error = %v, want wrapped %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMessage() unexpected error = %v", err)
			}
			if got.Text != "hello" || got.SentAt != now {
				t.Fatalf("message = %+v", got)
			}
		})
	}
}
