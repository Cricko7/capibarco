package ports

import (
	"context"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
)

// Clock allows deterministic time in usecase tests.
type Clock interface {
	Now() time.Time
}

// PasswordHasher hashes and verifies user passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password string, encodedHash string) (bool, error)
}

// UserRepository persists tenant-scoped users.
type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) error
	GetUserByEmail(ctx context.Context, tenantID string, email string) (domain.User, error)
	GetUserByID(ctx context.Context, tenantID string, userID string) (domain.User, error)
	UpdatePassword(ctx context.Context, tenantID string, userID string, passwordHash string, now time.Time) error
}

// RefreshTokenRepository persists hashed refresh tokens.
type RefreshTokenRepository interface {
	CreateRefreshToken(ctx context.Context, token domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string, reason string, now time.Time) error
	RevokeRefreshTokenFamily(ctx context.Context, familyID string, reason string, now time.Time) error
}

// PasswordResetRepository persists one-time reset tokens.
type PasswordResetRepository interface {
	CreatePasswordResetToken(ctx context.Context, token domain.PasswordResetToken) error
	GetPasswordResetTokenByHash(ctx context.Context, hash string) (domain.PasswordResetToken, error)
	ConsumePasswordResetToken(ctx context.Context, id string, now time.Time) error
}

// RBACRepository reads tenant-scoped roles and permissions.
type RBACRepository interface {
	GetUserRoles(ctx context.Context, tenantID string, userID string) ([]domain.Role, error)
	UserHasPermission(ctx context.Context, tenantID string, userID string, permission string) (bool, error)
}

// AccessTokenIssuer issues and validates asymmetric JWT access tokens.
type AccessTokenIssuer interface {
	IssueAccessToken(ctx context.Context, claims domain.TokenClaims) (string, error)
	ValidateAccessToken(ctx context.Context, token string) (domain.TokenClaims, error)
}

// AuditLogger writes security-sensitive audit events.
type AuditLogger interface {
	Log(ctx context.Context, event domain.AuditEvent) error
}

// Mailer sends transactional auth messages.
type Mailer interface {
	SendPasswordReset(ctx context.Context, tenantID string, email string, resetToken string) error
}

// EventPublisher publishes auth domain events to the message bus.
type EventPublisher interface {
	Publish(ctx context.Context, event domain.Event) error
}
