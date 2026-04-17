// Package chatgrpc contains the chat-service gRPC client adapter.
package chatgrpc

import (
	"context"
	"fmt"
	"time"

	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	app "github.com/petmatch/petmatch/internal/app/matching"
	domain "github.com/petmatch/petmatch/internal/domain/matching"
	"github.com/petmatch/petmatch/internal/pkg/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Client creates chat conversations through chat-service.
type Client struct {
	conn    *grpc.ClientConn
	client  chatv1.ChatServiceClient
	timeout time.Duration
	retries int
	breaker *resilience.CircuitBreaker
}

// Dial creates a chat gRPC client.
func Dial(ctx context.Context, address string, timeout time.Duration, retries int) (*Client, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if retries <= 0 {
		retries = 3
	}
	conn, err := grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("dial chat service: %w", err)
	}
	return &Client{
		conn:    conn,
		client:  chatv1.NewChatServiceClient(conn),
		timeout: timeout,
		retries: retries,
		breaker: resilience.NewCircuitBreaker(5, 30*time.Second),
	}, nil
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close chat grpc connection: %w", err)
	}
	return nil
}

// CreateConversation creates or returns an existing chat conversation for a match.
func (c *Client) CreateConversation(ctx context.Context, match domain.Match, idempotencyKey string) (app.ChatConversationResult, error) {
	var result app.ChatConversationResult
	err := resilience.Retry(ctx, c.retries, 100*time.Millisecond, func(ctx context.Context) error {
		return c.breaker.Execute(func() error {
			callCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()
			resp, err := c.client.CreateConversation(callCtx, &chatv1.CreateConversationRequest{
				MatchId:          match.ID,
				AnimalId:         match.AnimalID,
				AdopterProfileId: match.AdopterProfileID,
				OwnerProfileId:   match.OwnerProfileID,
				IdempotencyKey:   idempotencyKey,
			})
			if err != nil {
				if retryableStatus(err) {
					return err
				}
				return fmt.Errorf("create chat conversation: %w", err)
			}
			conversation := resp.GetConversation()
			if conversation == nil || conversation.GetConversationId() == "" {
				return fmt.Errorf("create chat conversation: empty conversation")
			}
			result = app.ChatConversationResult{ConversationID: conversation.GetConversationId(), Created: true}
			return nil
		})
	})
	if err != nil {
		return app.ChatConversationResult{}, err
	}
	return result, nil
}

func retryableStatus(err error) bool {
	code := status.Code(err)
	return code == codes.Unavailable || code == codes.ResourceExhausted || code == codes.DeadlineExceeded || code == codes.Aborted
}
