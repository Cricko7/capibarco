package grpc

import (
	"context"
	"fmt"
	"io"
	"time"

	chatv1 "github.com/petmatch/chat-service/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/chat-service/gen/go/petmatch/common/v1"
	appchat "github.com/petmatch/chat-service/internal/application/chat"
	"github.com/petmatch/chat-service/internal/domain/chat"
	"github.com/petmatch/chat-service/internal/infrastructure/realtime"
	"github.com/petmatch/chat-service/internal/observability"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ChatServer implements petmatch.chat.v1.ChatServiceServer.
type ChatServer struct {
	chatv1.ChatServiceServer
	service *appchat.Service
	hub     *realtime.Hub
	metrics *observability.Metrics
	auth    chat.TokenValidator
}

// NewChatServer creates a gRPC chat server.
func NewChatServer(service *appchat.Service, hub *realtime.Hub, metrics *observability.Metrics, auth chat.TokenValidator) *ChatServer {
	return &ChatServer{service: service, hub: hub, metrics: metrics, auth: auth}
}

func (s *ChatServer) CreateConversation(ctx context.Context, req *chatv1.CreateConversationRequest) (*chatv1.CreateConversationResponse, error) {
	conversation, err := s.service.CreateConversation(ctx, appchat.CreateConversationInput{
		MatchID:          req.GetMatchId(),
		AnimalID:         req.GetAnimalId(),
		AdopterProfileID: req.GetAdopterProfileId(),
		OwnerProfileID:   req.GetOwnerProfileId(),
		IdempotencyKey:   req.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &chatv1.CreateConversationResponse{Conversation: conversationToProto(conversation)}, nil
}

func (s *ChatServer) GetConversation(ctx context.Context, req *chatv1.GetConversationRequest) (*chatv1.GetConversationResponse, error) {
	conversation, err := s.service.GetConversation(ctx, req.GetConversationId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &chatv1.GetConversationResponse{Conversation: conversationToProto(conversation)}, nil
}

func (s *ChatServer) ListConversations(ctx context.Context, req *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error) {
	page := req.GetPage()
	conversations, token, err := s.service.ListConversations(ctx, chat.ListConversationsFilter{
		ParticipantProfileID: req.GetParticipantProfileId(),
		Statuses:             conversationStatusesFromProto(req.GetStatuses()),
		PageSize:             page.GetPageSize(),
		PageToken:            page.GetPageToken(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	items := make([]*chatv1.Conversation, 0, len(conversations))
	for _, conversation := range conversations {
		items = append(items, conversationToProto(conversation))
	}
	return &chatv1.ListConversationsResponse{Conversations: items, Page: pageResponse(token)}, nil
}

func (s *ChatServer) ListMessages(ctx context.Context, req *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	page := req.GetPage()
	messages, token, err := s.service.ListMessages(ctx, chat.ListMessagesFilter{
		ConversationID: req.GetConversationId(),
		PageSize:       page.GetPageSize(),
		PageToken:      page.GetPageToken(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	items := make([]*chatv1.Message, 0, len(messages))
	for _, message := range messages {
		items = append(items, messageToProto(message))
	}
	return &chatv1.ListMessagesResponse{Messages: items, Page: pageResponse(token)}, nil
}

func (s *ChatServer) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	message, err := s.service.SendMessage(ctx, sendInputFromProto(req))
	if err != nil {
		return nil, toStatusError(err)
	}
	return &chatv1.SendMessageResponse{Message: messageToProto(message)}, nil
}

func (s *ChatServer) UpdateMessage(ctx context.Context, req *chatv1.UpdateMessageRequest) (*chatv1.UpdateMessageResponse, error) {
	if req.GetMessage() == nil {
		return nil, status.Error(codes.InvalidArgument, "message is required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "update_mask is required")
	}
	allowed := false
	for _, path := range req.GetUpdateMask().GetPaths() {
		if path == "text" {
			allowed = true
			continue
		}
		return nil, status.Errorf(codes.InvalidArgument, "unsupported update path %q", path)
	}
	if !allowed {
		return nil, status.Error(codes.InvalidArgument, "text update is required")
	}
	message, err := s.service.UpdateMessage(ctx, appchat.UpdateMessageInput{
		MessageID: req.GetMessageId(),
		Text:      req.GetMessage().GetText(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &chatv1.UpdateMessageResponse{Message: messageToProto(message)}, nil
}

func (s *ChatServer) MarkRead(ctx context.Context, req *chatv1.MarkReadRequest) (*chatv1.MarkReadResponse, error) {
	receipt, err := s.service.MarkRead(ctx, appchat.MarkReadInput{
		ConversationID:  req.GetConversationId(),
		ReaderProfileID: req.GetReaderProfileId(),
		UpToMessageID:   req.GetUpToMessageId(),
	})
	if err != nil {
		return nil, toStatusError(err)
	}
	return &chatv1.MarkReadResponse{ReadAt: timestamppb.New(receipt.ReadAt)}, nil
}

func (s *ChatServer) Connect(stream chatv1.ChatService_ConnectServer) error {
	ctx := stream.Context()
	incomingFrames := make(chan *chatv1.ClientChatFrame)
	incomingErr := make(chan error, 1)
	go func() {
		defer close(incomingFrames)
		for {
			frame, err := stream.Recv()
			if err != nil {
				incomingErr <- err
				return
			}
			select {
			case incomingFrames <- frame:
			case <-ctx.Done():
				return
			}
		}
	}()

	var authedConversationID string
	var subscription <-chan chat.Event
	var cancelSubscription func()
	defer func() {
		if cancelSubscription != nil {
			cancelSubscription()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return toStatusError(ctx.Err())
		case err := <-incomingErr:
			if err == io.EOF {
				return nil
			}
			return toStatusError(err)
		case event, ok := <-subscription:
			if !ok {
				subscription = nil
				continue
			}
			if err := stream.Send(eventFrame(event)); err != nil {
				return toStatusError(err)
			}
		case frame, ok := <-incomingFrames:
			if !ok {
				return nil
			}
			switch payload := frame.GetPayload().(type) {
			case *chatv1.ClientChatFrame_Auth:
				if payload.Auth.GetAccessToken() == "" || payload.Auth.GetConversationId() == "" {
					if sendErr := stream.Send(errorFrame(frame.GetFrameId(), codes.Unauthenticated, "access_token and conversation_id are required", false)); sendErr != nil {
						return sendErr
					}
					continue
				}
				if _, validateErr := s.auth.ValidateToken(ctx, payload.Auth.GetAccessToken()); validateErr != nil {
					if sendErr := stream.Send(errorFrame(frame.GetFrameId(), codes.Unauthenticated, "access token is invalid", false)); sendErr != nil {
						return sendErr
					}
					continue
				}
				authedConversationID = payload.Auth.GetConversationId()
				if cancelSubscription != nil {
					cancelSubscription()
				}
				var events <-chan chat.Event
				events, cancelSubscription = s.hub.Subscribe(authedConversationID)
				subscription = events
				if sendErr := stream.Send(ackFrame(frame.GetFrameId(), "")); sendErr != nil {
					return sendErr
				}
			case *chatv1.ClientChatFrame_Message:
				if err := requireConversation(authedConversationID, payload.Message.GetConversationId()); err != nil {
					if sendErr := stream.Send(errorFrame(frame.GetFrameId(), codes.PermissionDenied, err.Error(), false)); sendErr != nil {
						return sendErr
					}
					continue
				}
				message, sendErr := s.service.SendMessage(ctx, sendInputFromProto(payload.Message))
				if sendErr != nil {
					if errFrame := stream.Send(errorFrame(frame.GetFrameId(), status.Code(toStatusError(sendErr)), sendErr.Error(), true)); errFrame != nil {
						return errFrame
					}
					continue
				}
				if err := stream.Send(ackFrame(frame.GetFrameId(), message.ID)); err != nil {
					return err
				}
			case *chatv1.ClientChatFrame_ReadReceipt:
				if err := requireConversation(authedConversationID, payload.ReadReceipt.GetConversationId()); err != nil {
					if sendErr := stream.Send(errorFrame(frame.GetFrameId(), codes.PermissionDenied, err.Error(), false)); sendErr != nil {
						return sendErr
					}
					continue
				}
				if _, readErr := s.service.MarkRead(ctx, appchat.MarkReadInput{
					ConversationID:  payload.ReadReceipt.GetConversationId(),
					ReaderProfileID: payload.ReadReceipt.GetReaderProfileId(),
					UpToMessageID:   payload.ReadReceipt.GetUpToMessageId(),
				}); readErr != nil {
					if errFrame := stream.Send(errorFrame(frame.GetFrameId(), status.Code(toStatusError(readErr)), readErr.Error(), true)); errFrame != nil {
						return errFrame
					}
					continue
				}
				if err := stream.Send(ackFrame(frame.GetFrameId(), "")); err != nil {
					return err
				}
			case *chatv1.ClientChatFrame_Typing:
				if err := requireConversation(authedConversationID, payload.Typing.GetConversationId()); err != nil {
					if sendErr := stream.Send(errorFrame(frame.GetFrameId(), codes.PermissionDenied, err.Error(), false)); sendErr != nil {
						return sendErr
					}
					continue
				}
				event := chat.Event{
					ID:           frame.GetFrameId(),
					Type:         chat.EventTypeParticipantTyping,
					OccurredAt:   time.Now().UTC(),
					PartitionKey: payload.Typing.GetConversationId(),
					Typing: &chat.Typing{
						ConversationID: payload.Typing.GetConversationId(),
						ProfileID:      payload.Typing.GetProfileId(),
						Typing:         payload.Typing.GetTyping(),
					},
				}
				if err := s.hub.Publish(ctx, event); err != nil {
					return toStatusError(err)
				}
				if err := stream.Send(ackFrame(frame.GetFrameId(), "")); err != nil {
					return err
				}
			default:
				if err := stream.Send(errorFrame(frame.GetFrameId(), codes.InvalidArgument, "unsupported frame payload", false)); err != nil {
					return err
				}
			}
		}
	}
}

func (s *ChatServer) SubscribeConversation(req *chatv1.SubscribeConversationRequest, stream chatv1.ChatService_SubscribeConversationServer) error {
	ctx := stream.Context()
	conversation, err := s.service.GetConversation(ctx, req.GetConversationId())
	if err != nil {
		return toStatusError(err)
	}
	if !conversation.HasParticipant(req.GetParticipantProfileId()) {
		return toStatusError(chat.ErrForbidden)
	}
	ch, cancel := s.hub.Subscribe(req.GetConversationId())
	defer cancel()
	if s.metrics != nil {
		s.metrics.RealtimeClients.Inc()
		defer s.metrics.RealtimeClients.Dec()
	}
	for {
		select {
		case <-ctx.Done():
			return toStatusError(ctx.Err())
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(eventToProto(event)); err != nil {
				return toStatusError(err)
			}
		}
	}
}

func sendInputFromProto(req *chatv1.SendMessageRequest) appchat.SendMessageInput {
	return appchat.SendMessageInput{
		ConversationID:  req.GetConversationId(),
		SenderProfileID: req.GetSenderProfileId(),
		Type:            messageTypeFromProto(req.GetType()),
		Text:            req.GetText(),
		Attachments:     photosFromProto(req.GetAttachments()),
		ClientMessageID: req.GetClientMessageId(),
		IdempotencyKey:  req.GetIdempotencyKey(),
	}
}

func pageResponse(token string) *commonv1.PageResponse {
	return &commonv1.PageResponse{NextPageToken: token}
}

func ackFrame(frameID string, messageID string) *chatv1.ServerChatFrame {
	ack := &chatv1.ChatAck{AckedFrameId: frameID}
	if messageID != "" {
		ack.MessageId = &messageID
	}
	return &chatv1.ServerChatFrame{
		FrameId:      frameID,
		Type:         chatv1.ChatFrameType_CHAT_FRAME_TYPE_ACK,
		ServerSentAt: timestamppb.Now(),
		Payload:      &chatv1.ServerChatFrame_Ack{Ack: ack},
	}
}

func eventFrame(event chat.Event) *chatv1.ServerChatFrame {
	return &chatv1.ServerChatFrame{
		FrameId:      event.ID,
		Type:         chatFrameType(event),
		ServerSentAt: timestamppb.Now(),
		Payload:      &chatv1.ServerChatFrame_Event{Event: eventToProto(event)},
	}
}

func errorFrame(frameID string, code codes.Code, message string, retryable bool) *chatv1.ServerChatFrame {
	return &chatv1.ServerChatFrame{
		FrameId:      frameID,
		Type:         chatv1.ChatFrameType_CHAT_FRAME_TYPE_ERROR,
		ServerSentAt: timestamppb.Now(),
		Payload: &chatv1.ServerChatFrame_Error{Error: &chatv1.ChatError{
			Code:      code.String(),
			Message:   message,
			Retryable: retryable,
		}},
	}
}

func chatFrameType(event chat.Event) chatv1.ChatFrameType {
	switch event.Type {
	case chat.EventTypeMessageSent:
		return chatv1.ChatFrameType_CHAT_FRAME_TYPE_MESSAGE
	case chat.EventTypeParticipantTyping:
		return chatv1.ChatFrameType_CHAT_FRAME_TYPE_TYPING
	case chat.EventTypeMessageRead:
		return chatv1.ChatFrameType_CHAT_FRAME_TYPE_READ_RECEIPT
	default:
		return chatv1.ChatFrameType_CHAT_FRAME_TYPE_UNSPECIFIED
	}
}

func requireConversation(authed string, requested string) error {
	if authed == "" {
		return fmt.Errorf("chat stream is not authenticated")
	}
	if authed != requested {
		return fmt.Errorf("frame conversation does not match authenticated conversation")
	}
	return nil
}
