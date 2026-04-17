package grpc

import (
	"time"

	chatv1 "github.com/petmatch/chat-service/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/chat-service/gen/go/petmatch/common/v1"
	"github.com/petmatch/chat-service/internal/domain/chat"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func conversationToProto(conversation chat.Conversation) *chatv1.Conversation {
	return &chatv1.Conversation{
		ConversationId:   conversation.ID,
		MatchId:          conversation.MatchID,
		AnimalId:         conversation.AnimalID,
		AdopterProfileId: conversation.AdopterProfileID,
		OwnerProfileId:   conversation.OwnerProfileID,
		Status:           chatv1.ConversationStatus(conversation.Status),
		LastMessage:      messagePtrToProto(conversation.LastMessage),
		UnreadCount:      conversation.UnreadCount,
		CreatedAt:        timeToProto(conversation.CreatedAt),
		UpdatedAt:        timeToProto(conversation.UpdatedAt),
	}
}

func messagePtrToProto(message *chat.Message) *chatv1.Message {
	if message == nil {
		return nil
	}
	return messageToProto(*message)
}

func messageToProto(message chat.Message) *chatv1.Message {
	return &chatv1.Message{
		MessageId:       message.ID,
		ConversationId:  message.ConversationID,
		SenderProfileId: message.SenderProfileID,
		Type:            chatv1.MessageType(message.Type),
		Text:            message.Text,
		Attachments:     photosToProto(message.Attachments),
		Metadata:        message.Metadata,
		SentAt:          timeToProto(message.SentAt),
		EditedAt:        timePtrToProto(message.EditedAt),
		DeletedAt:       timePtrToProto(message.DeletedAt),
	}
}

func messageTypeFromProto(messageType chatv1.MessageType) chat.MessageType {
	return chat.MessageType(messageType)
}

func conversationStatusesFromProto(statuses []chatv1.ConversationStatus) []chat.ConversationStatus {
	result := make([]chat.ConversationStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, chat.ConversationStatus(status))
	}
	return result
}

func photosFromProto(photos []*commonv1.Photo) []chat.Photo {
	result := make([]chat.Photo, 0, len(photos))
	for _, photo := range photos {
		if photo == nil {
			continue
		}
		result = append(result, chat.Photo{
			ID:          photo.PhotoId,
			URL:         photo.Url,
			Blurhash:    photo.Blurhash,
			Width:       photo.Width,
			Height:      photo.Height,
			ContentType: photo.ContentType,
			SortOrder:   photo.SortOrder,
			CreatedAt:   timestampToTime(photo.CreatedAt),
		})
	}
	return result
}

func photosToProto(photos []chat.Photo) []*commonv1.Photo {
	result := make([]*commonv1.Photo, 0, len(photos))
	for _, photo := range photos {
		result = append(result, &commonv1.Photo{
			PhotoId:     photo.ID,
			Url:         photo.URL,
			Blurhash:    photo.Blurhash,
			Width:       photo.Width,
			Height:      photo.Height,
			ContentType: photo.ContentType,
			SortOrder:   photo.SortOrder,
			CreatedAt:   timeToProto(photo.CreatedAt),
		})
	}
	return result
}

func eventToProto(event chat.Event) *chatv1.ChatEvent {
	envelope := &commonv1.EventEnvelope{
		EventId:        event.ID,
		EventType:      event.Type,
		SchemaVersion:  "1.0.0",
		Producer:       "chat-service",
		OccurredAt:     timeToProto(event.OccurredAt),
		TraceId:        event.TraceID,
		CorrelationId:  event.CorrelationID,
		IdempotencyKey: event.IdempotencyKey,
		PartitionKey:   event.PartitionKey,
	}
	protoEvent := &chatv1.ChatEvent{Envelope: envelope}
	switch {
	case event.Conversation != nil:
		protoEvent.Event = &chatv1.ChatEvent_ConversationCreated{
			ConversationCreated: &chatv1.ConversationCreatedEvent{
				Envelope:     envelope,
				Conversation: conversationToProto(*event.Conversation),
			},
		}
	case event.Message != nil:
		protoEvent.Event = &chatv1.ChatEvent_MessageSent{
			MessageSent: &chatv1.MessageSentEvent{
				Envelope: envelope,
				Message:  messageToProto(*event.Message),
			},
		}
	case event.ReadReceipt != nil:
		protoEvent.Event = &chatv1.ChatEvent_MessageRead{
			MessageRead: &chatv1.MessageReadEvent{
				Envelope:        envelope,
				ConversationId:  event.ReadReceipt.ConversationID,
				ReaderProfileId: event.ReadReceipt.ReaderProfileID,
				UpToMessageId:   event.ReadReceipt.UpToMessageID,
				ReadAt:          timeToProto(event.ReadReceipt.ReadAt),
			},
		}
	case event.Typing != nil:
		protoEvent.Event = &chatv1.ChatEvent_ParticipantTyping{
			ParticipantTyping: &chatv1.ParticipantTypingEvent{
				Envelope: envelope,
				Typing: &chatv1.TypingPayload{
					ConversationId: event.Typing.ConversationID,
					ProfileId:      event.Typing.ProfileID,
					Typing:         event.Typing.Typing,
				},
			},
		}
	}
	return protoEvent
}

func timeToProto(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

func timePtrToProto(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}
	return timeToProto(*value)
}

func timestampToTime(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}
