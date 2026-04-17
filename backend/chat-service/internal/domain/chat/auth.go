package chat

import "context"

// Principal is an authenticated actor returned by auth-service.
type Principal struct {
	Subject     string
	TenantID    string
	Email       string
	Roles       []string
	Permissions []string
	TokenID     string
}

// TokenValidator validates access tokens against the auth boundary.
type TokenValidator interface {
	ValidateToken(ctx context.Context, accessToken string) (Principal, error)
}
