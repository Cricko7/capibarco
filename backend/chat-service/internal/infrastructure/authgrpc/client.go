package authgrpc

import (
	"context"
	"encoding/json"
	"fmt"

	authv1 "github.com/petmatch/chat-service/gen/go/petmatch/auth"
	"github.com/petmatch/chat-service/internal/domain/chat"
	"github.com/petmatch/chat-service/internal/infrastructure/breaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (jsonCodec) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
func (jsonCodec) Name() string                    { return "json" }

// Client validates tokens through auth-service.
type Client struct {
	conn     *grpc.ClientConn
	client   authv1.AuthServiceClient
	executor *breaker.Executor
}

// NewClient connects to auth-service.
func NewClient(ctx context.Context, address string, executor *breaker.Executor) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create auth grpc client: %w", err)
	}
	client := &Client{
		conn:     conn,
		client:   authv1.NewAuthServiceClient(conn),
		executor: executor,
	}
	conn.Connect()
	_ = ctx
	return client, nil
}

// Close closes the underlying auth-service connection.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("close auth grpc client: %w", err)
	}
	return nil
}

// ValidateToken validates an access token and returns principal claims.
func (c *Client) ValidateToken(ctx context.Context, accessToken string) (chat.Principal, error) {
	var response *authv1.ValidateTokenResponse
	err := c.executor.Do(ctx, func(callCtx context.Context) error {
		var callErr error
		response, callErr = c.client.ValidateToken(
			callCtx,
			&authv1.ValidateTokenRequest{AccessToken: accessToken},
			grpc.ForceCodec(jsonCodec{}),
		)
		return callErr
	})
	if err != nil {
		return chat.Principal{}, fmt.Errorf("validate token through auth-service: %w", err)
	}
	if !response.GetValid() {
		return chat.Principal{}, fmt.Errorf("auth-service rejected token: %w", chat.ErrForbidden)
	}
	return chat.Principal{
		Subject:     response.GetSubject(),
		TenantID:    response.GetTenantId(),
		Email:       response.GetEmail(),
		Roles:       append([]string(nil), response.GetRoles()...),
		Permissions: append([]string(nil), response.GetPermissions()...),
		TokenID:     response.GetTokenId(),
	}, nil
}
