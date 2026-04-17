package domain

import "time"

// User is a tenant-scoped principal.
type User struct {
	ID           string
	TenantID     string
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// RefreshToken is stored hashed and rotated on every use.
type RefreshToken struct {
	ID            string
	TenantID      string
	UserID        string
	Hash          string
	FamilyID      string
	ExpiresAt     time.Time
	CreatedAt     time.Time
	RevokedAt     *time.Time
	RevokedReason string
	ReplacedByID  string
}

// PasswordResetToken is a one-time opaque reset token stored by hash.
type PasswordResetToken struct {
	ID         string
	TenantID   string
	UserID     string
	Hash       string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	ConsumedAt *time.Time
}

// Role grants permissions within one tenant.
type Role struct {
	ID          string
	TenantID    string
	Name        string
	Permissions []Permission
}

// Permission is formatted as system:resource:action.
type Permission struct {
	ID       string
	TenantID string
	Value    string
}

// TokenClaims is the internal representation returned by access-token validation.
type TokenClaims struct {
	Subject     string
	TenantID    string
	Email       string
	Roles       []string
	Permissions []string
	ExpiresAt   time.Time
	IssuedAt    time.Time
	TokenID     string
}

// AuditEvent records security-sensitive operations.
type AuditEvent struct {
	ID        string
	TenantID  string
	UserID    string
	Action    string
	Outcome   string
	IP        string
	UserAgent string
	Metadata  map[string]string
	CreatedAt time.Time
}
