package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/hackathon/authsvc/internal/usecase"
)

func TestAuthServicePublishesContractEvents(t *testing.T) {
	ctx := context.Background()
	clock := &fakeClock{now: time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)}
	users := newMemoryUserRepo()
	refreshRepo := newMemoryRefreshRepo()
	resetRepo := newMemoryResetRepo()
	rbac := &fakeRBAC{roles: []domain.Role{{Name: "User"}}}
	publisher := &recordingPublisher{}
	auth := usecase.NewAuthService(usecase.AuthDependencies{
		Users:     users,
		Refresh:   usecase.NewTokenService(refreshRepo, clock, usecase.TokenConfig{RefreshTTL: time.Hour}),
		Resets:    resetRepo,
		RBAC:      rbac,
		Hasher:    fakeHasher{},
		Issuer:    &fakeIssuer{clock: clock},
		Audit:     noopAudit{},
		Mailer:    noopMailer{},
		Clock:     clock,
		Publisher: publisher,
		Config:    usecase.AuthConfig{ResetTTL: 15 * time.Minute},
	})

	registered, err := auth.Register(ctx, usecase.RegisterInput{
		TenantID: "default",
		Email:    "alice@example.com",
		Password: "CorrectHorseBatteryStaple!",
		IP:       "203.0.113.10",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if publisher.lastEvent().Type != domain.EventUserRegistered {
		t.Fatalf("expected %s, got %s", domain.EventUserRegistered, publisher.lastEvent().Type)
	}
	if publisher.lastEvent().Key != registered.User.ID {
		t.Fatalf("expected user id partition key, got %s", publisher.lastEvent().Key)
	}

	if _, err := auth.Login(ctx, usecase.LoginInput{
		TenantID: "default",
		Email:    "alice@example.com",
		Password: "CorrectHorseBatteryStaple!",
		IP:       "203.0.113.10",
	}); err != nil {
		t.Fatalf("login: %v", err)
	}
	if publisher.lastEvent().Type != domain.EventUserLoggedIn {
		t.Fatalf("expected %s, got %s", domain.EventUserLoggedIn, publisher.lastEvent().Type)
	}

	if _, err := auth.RefreshToken(ctx, registered.RefreshToken); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if publisher.lastEvent().Type != domain.EventTokenRefreshed {
		t.Fatalf("expected %s, got %s", domain.EventTokenRefreshed, publisher.lastEvent().Type)
	}

	if err := auth.ForgotPassword(ctx, "default", "alice@example.com", "203.0.113.10"); err != nil {
		t.Fatalf("forgot password: %v", err)
	}
	if publisher.lastEvent().Type != domain.EventPasswordResetRequested {
		t.Fatalf("expected %s, got %s", domain.EventPasswordResetRequested, publisher.lastEvent().Type)
	}

	resetRaw := resetRepo.lastRaw
	if err := auth.ResetPassword(ctx, "default", resetRaw, "AnotherCorrectHorseBatteryStaple!", "203.0.113.10"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if publisher.lastEvent().Type != domain.EventPasswordResetCompleted {
		t.Fatalf("expected %s, got %s", domain.EventPasswordResetCompleted, publisher.lastEvent().Type)
	}

	rbac.allow = false
	_, _, err = auth.Authorize(ctx, "default|"+registered.User.ID+"|alice@example.com|User|token-1", "billing:invoice:read")
	if err != domain.ErrPermissionDenied {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if publisher.lastEvent().Type != domain.EventPermissionDenied {
		t.Fatalf("expected %s, got %s", domain.EventPermissionDenied, publisher.lastEvent().Type)
	}
}

type recordingPublisher struct {
	events []domain.Event
}

func (p *recordingPublisher) Publish(_ context.Context, event domain.Event) error {
	p.events = append(p.events, event)
	return nil
}

func (p *recordingPublisher) lastEvent() domain.Event {
	if len(p.events) == 0 {
		return domain.Event{}
	}
	return p.events[len(p.events)-1]
}

type memoryUserRepo struct {
	users map[string]domain.User
}

func newMemoryUserRepo() *memoryUserRepo {
	return &memoryUserRepo{users: map[string]domain.User{}}
}

func (r *memoryUserRepo) CreateUser(_ context.Context, user domain.User) error {
	key := user.TenantID + ":" + user.Email
	if _, ok := r.users[key]; ok {
		return domain.ErrAlreadyExists
	}
	r.users[key] = user
	r.users[user.TenantID+":"+user.ID] = user
	return nil
}

func (r *memoryUserRepo) GetUserByEmail(_ context.Context, tenantID string, email string) (domain.User, error) {
	user, ok := r.users[tenantID+":"+email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func (r *memoryUserRepo) GetUserByID(_ context.Context, tenantID string, userID string) (domain.User, error) {
	user, ok := r.users[tenantID+":"+userID]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return user, nil
}

func (r *memoryUserRepo) UpdatePassword(_ context.Context, tenantID string, userID string, passwordHash string, now time.Time) error {
	user, ok := r.users[tenantID+":"+userID]
	if !ok {
		return domain.ErrNotFound
	}
	user.PasswordHash = passwordHash
	user.UpdatedAt = now
	r.users[tenantID+":"+userID] = user
	r.users[tenantID+":"+user.Email] = user
	return nil
}

type memoryResetRepo struct {
	tokens  map[string]domain.PasswordResetToken
	lastRaw string
}

func newMemoryResetRepo() *memoryResetRepo {
	return &memoryResetRepo{tokens: map[string]domain.PasswordResetToken{}}
}

func (r *memoryResetRepo) CreatePasswordResetToken(_ context.Context, token domain.PasswordResetToken) error {
	r.tokens[token.Hash] = token
	r.lastRaw = token.ID
	r.tokens[token.ID] = token
	return nil
}

func (r *memoryResetRepo) GetPasswordResetTokenByHash(_ context.Context, hash string) (domain.PasswordResetToken, error) {
	for _, token := range r.tokens {
		if token.ID == r.lastRaw {
			return token, nil
		}
	}
	return domain.PasswordResetToken{}, domain.ErrNotFound
}

func (r *memoryResetRepo) ConsumePasswordResetToken(_ context.Context, id string, now time.Time) error {
	for hash, token := range r.tokens {
		if token.ID == id {
			token.ConsumedAt = &now
			r.tokens[hash] = token
		}
	}
	return nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) { return "hash:" + password, nil }
func (fakeHasher) Verify(password string, encodedHash string) (bool, error) {
	return encodedHash == "hash:"+password, nil
}

type fakeIssuer struct {
	clock *fakeClock
}

func (i *fakeIssuer) IssueAccessToken(_ context.Context, claims domain.TokenClaims) (string, error) {
	return claims.TenantID + "|" + claims.Subject + "|" + claims.Email + "|" + joinRoles(claims.Roles) + "|token-1", nil
}

func (i *fakeIssuer) ValidateAccessToken(_ context.Context, token string) (domain.TokenClaims, error) {
	parts := splitToken(token)
	if len(parts) != 5 {
		return domain.TokenClaims{}, domain.ErrInvalidToken
	}
	return domain.TokenClaims{
		TenantID:  parts[0],
		Subject:   parts[1],
		Email:     parts[2],
		Roles:     []string{parts[3]},
		TokenID:   parts[4],
		ExpiresAt: i.clock.Now().Add(15 * time.Minute),
		IssuedAt:  i.clock.Now(),
	}, nil
}

func joinRoles(roles []string) string {
	if len(roles) == 0 {
		return ""
	}
	return roles[0]
}

func splitToken(token string) []string {
	var parts []string
	start := 0
	for i := range token {
		if token[i] == '|' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	return append(parts, token[start:])
}

type fakeRBAC struct {
	roles []domain.Role
	allow bool
}

func (r *fakeRBAC) GetUserRoles(context.Context, string, string) ([]domain.Role, error) {
	return r.roles, nil
}

func (r *fakeRBAC) UserHasPermission(context.Context, string, string, string) (bool, error) {
	return r.allow, nil
}

type noopAudit struct{}

func (noopAudit) Log(context.Context, domain.AuditEvent) error { return nil }

type noopMailer struct{}

func (noopMailer) SendPasswordReset(context.Context, string, string, string) error { return nil }
