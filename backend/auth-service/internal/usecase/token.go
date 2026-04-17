package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/hackathon/authsvc/internal/ports"
)

// TokenConfig controls refresh token behavior.
type TokenConfig struct {
	RefreshTTL time.Duration
}

// IssuedRefreshToken returns the raw token once and the persisted metadata.
type IssuedRefreshToken struct {
	Raw             string
	Token           domain.RefreshToken
	PreviousTokenID string
}

// TokenService handles opaque refresh token rotation and reuse detection.
type TokenService struct {
	repo  ports.RefreshTokenRepository
	clock ports.Clock
	cfg   TokenConfig
}

// NewTokenService creates a token service.
func NewTokenService(repo ports.RefreshTokenRepository, clock ports.Clock, cfg TokenConfig) *TokenService {
	return &TokenService{repo: repo, clock: clock, cfg: cfg}
}

// IssueRefreshToken creates a first token in a new family.
func (s *TokenService) IssueRefreshToken(ctx context.Context, user domain.User) (IssuedRefreshToken, error) {
	familyID, err := newID()
	if err != nil {
		return IssuedRefreshToken{}, err
	}
	return s.issue(ctx, user.TenantID, user.ID, familyID)
}

// RotateRefreshToken revokes the current token and creates a replacement.
func (s *TokenService) RotateRefreshToken(ctx context.Context, raw string) (IssuedRefreshToken, error) {
	now := s.clock.Now()
	current, err := s.repo.GetRefreshTokenByHash(ctx, hashToken(raw))
	if err != nil {
		return IssuedRefreshToken{}, domain.ErrInvalidToken
	}
	if current.ExpiresAt.Before(now) {
		return IssuedRefreshToken{}, domain.ErrTokenExpired
	}
	if current.RevokedAt != nil {
		if err := s.repo.RevokeRefreshTokenFamily(ctx, current.FamilyID, "reuse_detected", now); err != nil {
			return IssuedRefreshToken{}, fmt.Errorf("revoke refresh token family: %w", err)
		}
		return IssuedRefreshToken{}, domain.ErrTokenReused
	}
	user := domain.User{ID: current.UserID, TenantID: current.TenantID}
	next, err := s.issue(ctx, user.TenantID, user.ID, current.FamilyID)
	if err != nil {
		return IssuedRefreshToken{}, err
	}
	if err := s.repo.RevokeRefreshToken(ctx, current.ID, "rotated", now); err != nil {
		return IssuedRefreshToken{}, fmt.Errorf("revoke refresh token: %w", err)
	}
	next.PreviousTokenID = current.ID
	return next, nil
}

func (s *TokenService) issue(ctx context.Context, tenantID string, userID string, familyID string) (IssuedRefreshToken, error) {
	id, err := newID()
	if err != nil {
		return IssuedRefreshToken{}, err
	}
	raw, err := randomToken(32)
	if err != nil {
		return IssuedRefreshToken{}, err
	}
	now := s.clock.Now()
	token := domain.RefreshToken{
		ID:        id,
		TenantID:  tenantID,
		UserID:    userID,
		Hash:      hashToken(raw),
		FamilyID:  familyID,
		ExpiresAt: now.Add(s.cfg.RefreshTTL),
		CreatedAt: now,
	}
	if err := s.repo.CreateRefreshToken(ctx, token); err != nil {
		return IssuedRefreshToken{}, fmt.Errorf("create refresh token: %w", err)
	}
	return IssuedRefreshToken{Raw: raw, Token: token}, nil
}
