package grpc

import (
	"context"
	"io"
	"testing"
	"time"

	chatv1 "github.com/petmatch/chat-service/gen/go/petmatch/chat/v1"
	"github.com/petmatch/chat-service/internal/domain/chat"
	"github.com/petmatch/chat-service/internal/infrastructure/realtime"
	"google.golang.org/grpc/metadata"
)

func TestConnectStreamsHubEventsAfterAuth(t *testing.T) {
	hub := realtime.NewHub()
	server := NewChatServer(nil, hub, nil, stubTokenValidator{})
	stream := newConnectStreamStub()

	done := make(chan error, 1)
	go func() {
		done <- server.Connect(stream)
	}()

	stream.push(&chatv1.ClientChatFrame{
		FrameId: "frame-auth",
		Payload: &chatv1.ClientChatFrame_Auth{
			Auth: &chatv1.ChatAuthPayload{
				AccessToken:    "valid-token",
				ConversationId: "conversation-1",
			},
		},
	})

	ack := stream.take(t)
	if ack.GetAck().GetAckedFrameId() != "frame-auth" {
		t.Fatalf("ack frame id = %q, want %q", ack.GetAck().GetAckedFrameId(), "frame-auth")
	}

	now := time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC)
	if err := hub.Publish(context.Background(), chat.Event{
		ID:           "event-1",
		Type:         chat.EventTypeMessageSent,
		OccurredAt:   now,
		PartitionKey: "conversation-1",
		Message: &chat.Message{
			ID:              "message-1",
			ConversationID:  "conversation-1",
			SenderProfileID: "profile-2",
			Type:            chat.MessageTypeText,
			Text:            "hello from realtime",
			SentAt:          now,
		},
	}); err != nil {
		t.Fatalf("hub.Publish() error = %v", err)
	}

	eventFrame := stream.take(t)
	if got := eventFrame.GetEvent().GetMessageSent().GetMessage().GetMessageId(); got != "message-1" {
		t.Fatalf("message id = %q, want %q", got, "message-1")
	}

	stream.closeInput()
	if err := <-done; err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
}

type stubTokenValidator struct{}

func (stubTokenValidator) ValidateToken(context.Context, string) (chat.Principal, error) {
	return chat.Principal{Subject: "user-1"}, nil
}

type connectStreamStub struct {
	ctx    context.Context
	recvCh chan *chatv1.ClientChatFrame
	sendCh chan *chatv1.ServerChatFrame
}

func newConnectStreamStub() *connectStreamStub {
	return &connectStreamStub{
		ctx:    context.Background(),
		recvCh: make(chan *chatv1.ClientChatFrame, 8),
		sendCh: make(chan *chatv1.ServerChatFrame, 8),
	}
}

func (s *connectStreamStub) push(frame *chatv1.ClientChatFrame) {
	s.recvCh <- frame
}

func (s *connectStreamStub) take(t *testing.T) *chatv1.ServerChatFrame {
	t.Helper()
	select {
	case frame := <-s.sendCh:
		return frame
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server frame")
		return nil
	}
}

func (s *connectStreamStub) closeInput() {
	close(s.recvCh)
}

func (s *connectStreamStub) SetHeader(metadata.MD) error { return nil }

func (s *connectStreamStub) SendHeader(metadata.MD) error { return nil }

func (s *connectStreamStub) SetTrailer(metadata.MD) {}

func (s *connectStreamStub) Context() context.Context { return s.ctx }

func (s *connectStreamStub) Send(frame *chatv1.ServerChatFrame) error {
	select {
	case s.sendCh <- frame:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
}

func (s *connectStreamStub) Recv() (*chatv1.ClientChatFrame, error) {
	select {
	case frame, ok := <-s.recvCh:
		if !ok {
			return nil, io.EOF
		}
		return frame, nil
	case <-s.ctx.Done():
		return nil, io.EOF
	}
}

func (s *connectStreamStub) SendMsg(message any) error {
	frame, ok := message.(*chatv1.ServerChatFrame)
	if !ok {
		return nil
	}
	return s.Send(frame)
}

func (s *connectStreamStub) RecvMsg(message any) error {
	frame, err := s.Recv()
	if err != nil {
		return err
	}
	target, ok := message.(*chatv1.ClientChatFrame)
	if !ok {
		return nil
	}
	*target = *frame
	return nil
}
