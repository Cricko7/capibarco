package httpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/petmatch/petmatch/internal/app/gateway"
	kafkaevents "github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/pkg/problem"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
	"google.golang.org/protobuf/encoding/protojson"

	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func (s *Server) chatWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.WarnContext(c.Request.Context(), "upgrade websocket", "error", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.DebugContext(c.Request.Context(), "close websocket", "error", err)
		}
	}()

	stream, err := s.chat.Connect(c.Request.Context())
	if err != nil {
		_ = conn.WriteJSON(gin.H{"error": err.Error()})
		return
	}
	connectionID := uuid.NewString()
	principal, _ := gateway.PrincipalFromContext(c.Request.Context())
	s.publishWebSocketEvent(c, kafkaevents.TopicWebSocketConnected, connectionID, map[string]any{
		"connection_id": connectionID,
		"actor_id":      principal.ActorID,
		"ip":            c.ClientIP(),
		"user_agent":    c.Request.UserAgent(),
		"connected_at":  time.Now().UTC().Format(time.RFC3339Nano),
	})

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	errCh := make(chan error, 2)
	go func() {
		for {
			_, payload, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			var frame chatv1.ClientChatFrame
			if err := protojson.Unmarshal(payload, &frame); err != nil {
				errCh <- fmt.Errorf("decode websocket frame: %w", err)
				return
			}
			if err := stream.Send(&frame); err != nil {
				errCh <- fmt.Errorf("send chat frame: %w", err)
				return
			}
		}
	}()
	go func() {
		for {
			frame, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(frame)
			if err != nil {
				errCh <- fmt.Errorf("encode chat frame: %w", err)
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				errCh <- err
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			s.logger.WarnContext(c.Request.Context(), "websocket bridge stopped", "error", err)
		}
	}
	s.publishWebSocketEvent(c, kafkaevents.TopicWebSocketDisconnected, connectionID, map[string]any{
		"connection_id":   connectionID,
		"actor_id":        principal.ActorID,
		"disconnected_at": time.Now().UTC().Format(time.RFC3339Nano),
		"reason":          "closed",
	})
}

func (s *Server) streamNotifications(c *gin.Context) {
	if s.notifications == nil {
		problem.Abort(c, gateway.ErrDependencyDisabled)
		return
	}
	principal, ok := gateway.PrincipalFromContext(c.Request.Context())
	if !ok {
		problem.Abort(c, gateway.ErrUnauthenticated)
		return
	}
	stream, err := s.notifications.StreamNotifications(c.Request.Context(), &notificationv1.StreamNotificationsRequest{RecipientProfileId: principal.ActorID})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	beginSSE(c)
	c.Stream(func(w io.Writer) bool {
		notification, err := stream.Recv()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				s.logger.WarnContext(c.Request.Context(), "notification stream closed", "error", err)
			}
			return false
		}
		payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(notification)
		if err != nil {
			s.logger.WarnContext(c.Request.Context(), "marshal notification", "error", err)
			return true
		}
		_, _ = fmt.Fprintf(w, "event: notification\ndata: %s\n\n", payload)
		return true
	})
}

func beginSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	c.Writer.Flush()
}

func (s *Server) publishWebSocketEvent(c *gin.Context, topic string, connectionID string, payload map[string]any) {
	payload["request_id"] = requestid.From(c.Request.Context())
	if err := s.publisher.Publish(c.Request.Context(), topic, connectionID, payload); err != nil {
		s.logger.WarnContext(c.Request.Context(), "publish websocket event", "topic", topic, "error", err)
	}
}
