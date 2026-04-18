package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/petmatch/petmatch/internal/app/gateway"
	"github.com/petmatch/petmatch/internal/pkg/resilience"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

func init() {
	encoding.RegisterCodec(jsonCodec{})
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (jsonCodec) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
func (jsonCodec) Name() string                    { return "json" }

// AuthClient calls auth-service's manually registered JSON gRPC service.
type AuthClient struct {
	conn    *grpc.ClientConn
	timeout time.Duration
	res     *resilience.Client
}

// NewAuthClient creates an auth-service client.
func NewAuthClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *AuthClient {
	return &AuthClient{conn: conn, timeout: timeout, res: res}
}

// Register proxies registration to auth-service.
func (c *AuthClient) Register(ctx context.Context, input gateway.RegisterInput) (*gateway.AuthResponse, error) {
	req := registerRequest{TenantID: input.TenantID, Email: input.Email, Password: input.Password, IP: input.IP}
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*gateway.AuthResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		var out authResponse
		if err := c.conn.Invoke(ctx, "/auth.v1.AuthService/Register", &req, &out, grpc.ForceCodec(jsonCodec{})); err != nil {
			return nil, err
		}
		return out.toGateway(), nil
	})
}

// Login proxies credentials to auth-service.
func (c *AuthClient) Login(ctx context.Context, input gateway.LoginInput) (*gateway.AuthResponse, error) {
	req := loginRequest{TenantID: input.TenantID, Email: input.Email, Password: input.Password, IP: input.IP}
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*gateway.AuthResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		var out authResponse
		if err := c.conn.Invoke(ctx, "/auth.v1.AuthService/Login", &req, &out, grpc.ForceCodec(jsonCodec{})); err != nil {
			return nil, err
		}
		return out.toGateway(), nil
	})
}

// Validate validates a bearer token through auth-service.
func (c *AuthClient) Validate(ctx context.Context, token string) (gateway.Principal, error) {
	req := validateTokenRequest{AccessToken: token}
	return resilience.Do(ctx, c.res, func(ctx context.Context) (gateway.Principal, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		var out validateTokenResponse
		if err := c.conn.Invoke(ctx, "/auth.v1.AuthService/ValidateToken", &req, &out, grpc.ForceCodec(jsonCodec{})); err != nil {
			return gateway.Principal{}, err
		}
		if !out.Valid {
			return gateway.Principal{}, fmt.Errorf("%w: invalid token", gateway.ErrUnauthenticated)
		}
		return out.toPrincipal(), nil
	})
}

// Authorize checks a permission through auth-service.
func (c *AuthClient) Authorize(ctx context.Context, token string, permission string) (gateway.Principal, bool, error) {
	req := authorizeRequest{AccessToken: token, Permission: permission}
	out, err := resilience.Do(ctx, c.res, func(ctx context.Context) (authorizeResult, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		var out authorizeResponse
		if err := c.conn.Invoke(ctx, "/auth.v1.AuthService/Authorize", &req, &out, grpc.ForceCodec(jsonCodec{})); err != nil {
			return authorizeResult{}, err
		}
		return authorizeResult{principal: out.Claims.toPrincipal(), allowed: out.Allowed}, nil
	})
	if err != nil {
		return gateway.Principal{}, false, err
	}
	return out.principal, out.allowed, nil
}

type authorizeResult struct {
	principal gateway.Principal
	allowed   bool
}

type loginRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IP       string `json:"ip"`
}

type registerRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	IP       string `json:"ip"`
}

type validateTokenRequest struct {
	AccessToken string `json:"access_token"`
}

type authorizeRequest struct {
	AccessToken string `json:"access_token"`
	Permission  string `json:"permission"`
}

type authResponse struct {
	User         *authUser `json:"user,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    string    `json:"expires_at"`
}

type authUser struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type validateTokenResponse struct {
	Valid       bool     `json:"valid"`
	Subject     string   `json:"subject"`
	TenantID    string   `json:"tenant_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ExpiresAt   string   `json:"expires_at"`
	TokenID     string   `json:"token_id"`
}

type authorizeResponse struct {
	Allowed bool                  `json:"allowed"`
	Claims  validateTokenResponse `json:"claims"`
}

func (r authResponse) toGateway() *gateway.AuthResponse {
	out := &gateway.AuthResponse{AccessToken: r.AccessToken, RefreshToken: r.RefreshToken, ExpiresAt: r.ExpiresAt}
	if r.User != nil {
		out.User = &gateway.AuthUser{
			ID:        r.User.ID,
			TenantID:  r.User.TenantID,
			Email:     r.User.Email,
			IsActive:  r.User.IsActive,
			CreatedAt: r.User.CreatedAt,
			UpdatedAt: r.User.UpdatedAt,
		}
	}
	return out
}

func (r validateTokenResponse) toPrincipal() gateway.Principal {
	return gateway.Principal{
		ActorID:     r.Subject,
		TenantID:    r.TenantID,
		Email:       r.Email,
		Roles:       append([]string(nil), r.Roles...),
		Permissions: append([]string(nil), r.Permissions...),
		TokenID:     r.TokenID,
	}
}
