package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/hackathon/authsvc/internal/usecase"
)

func TestRefreshTokenRotationRejectsReuse(t *testing.T) {
	ctx := context.Background()
	clock := &fakeClock{now: time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC)}
	repo := newMemoryRefreshRepo()
	tokens := usecase.NewTokenService(repo, clock, usecase.TokenConfig{RefreshTTL: 30 * 24 * time.Hour})

	issued, err := tokens.IssueRefreshToken(ctx, domain.User{ID: "user-1", TenantID: "tenant-1"})
	if err != nil {
		t.Fatalf("issue refresh token: %v", err)
	}

	rotated, err := tokens.RotateRefreshToken(ctx, issued.Raw)
	if err != nil {
		t.Fatalf("rotate refresh token: %v", err)
	}
	if rotated.Raw == issued.Raw {
		t.Fatal("expected a new refresh token value")
	}

	_, err = tokens.RotateRefreshToken(ctx, issued.Raw)
	if err == nil {
		t.Fatal("expected reused refresh token to be rejected")
	}
	if err != domain.ErrTokenReused {
		t.Fatalf("expected ErrTokenReused, got %v", err)
	}
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

type memoryRefreshRepo struct {
	tokens map[string]domain.RefreshToken
}

func newMemoryRefreshRepo() *memoryRefreshRepo {
	return &memoryRefreshRepo{tokens: map[string]domain.RefreshToken{}}
}

func (r *memoryRefreshRepo) CreateRefreshToken(_ context.Context, token domain.RefreshToken) error {
	r.tokens[token.Hash] = token
	return nil
}

func (r *memoryRefreshRepo) GetRefreshTokenByHash(_ context.Context, hash string) (domain.RefreshToken, error) {
	token, ok := r.tokens[hash]
	if !ok {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	return token, nil
}

func (r *memoryRefreshRepo) RevokeRefreshToken(_ context.Context, id string, reason string, now time.Time) error {
	for hash, token := range r.tokens {
		if token.ID == id {
			token.RevokedAt = &now
			token.RevokedReason = reason
			r.tokens[hash] = token
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *memoryRefreshRepo) RevokeRefreshTokenFamily(_ context.Context, familyID string, reason string, now time.Time) error {
	for hash, token := range r.tokens {
		if token.FamilyID == familyID {
			token.RevokedAt = &now
			token.RevokedReason = reason
			r.tokens[hash] = token
		}
	}
	return nil
}
