package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/hackathon/authsvc/internal/ports"
)

// AuthConfig configures auth workflows.
type AuthConfig struct {
	ResetTTL time.Duration
}

// AuthService contains authentication and authorization usecases.
type AuthService struct {
	users   ports.UserRepository
	refresh *TokenService
	resets  ports.PasswordResetRepository
	rbac    ports.RBACRepository
	hasher  ports.PasswordHasher
	issuer  ports.AccessTokenIssuer
	audit   ports.AuditLogger
	mailer  ports.Mailer
	events  ports.EventPublisher
	clock   ports.Clock
	cfg     AuthConfig
	metrics Metrics
}

// Metrics receives auth service counters.
type Metrics interface {
	IncRegister(outcome string)
	IncLogin(outcome string)
	IncRefresh(outcome string)
	IncError(operation string)
}

// NoopMetrics is used when no metrics collector is configured.
type NoopMetrics struct{}

func (NoopMetrics) IncRegister(string) {}
func (NoopMetrics) IncLogin(string)    {}
func (NoopMetrics) IncRefresh(string)  {}
func (NoopMetrics) IncError(string)    {}

// AuthDependencies groups AuthService dependencies.
type AuthDependencies struct {
	Users     ports.UserRepository
	Refresh   *TokenService
	Resets    ports.PasswordResetRepository
	RBAC      ports.RBACRepository
	Hasher    ports.PasswordHasher
	Issuer    ports.AccessTokenIssuer
	Audit     ports.AuditLogger
	Mailer    ports.Mailer
	Publisher ports.EventPublisher
	Clock     ports.Clock
	Metrics   Metrics
	Config    AuthConfig
}

// NewAuthService creates an auth usecase service.
func NewAuthService(deps AuthDependencies) *AuthService {
	metrics := deps.Metrics
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	return &AuthService{
		users:   deps.Users,
		refresh: deps.Refresh,
		resets:  deps.Resets,
		rbac:    deps.RBAC,
		hasher:  deps.Hasher,
		issuer:  deps.Issuer,
		audit:   deps.Audit,
		mailer:  deps.Mailer,
		events:  deps.Publisher,
		clock:   deps.Clock,
		cfg:     deps.Config,
		metrics: metrics,
	}
}

type RegisterInput struct {
	TenantID string
	Email    string
	Password string
	IP       string
}

type LoginInput struct {
	TenantID  string
	Email     string
	Password  string
	IP        string
	UserAgent string
}

type AuthOutput struct {
	User         domain.User
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	TokenID      string
	Roles        []string
}

// Register creates a user and returns tokens.
func (s *AuthService) Register(ctx context.Context, in RegisterInput) (AuthOutput, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return AuthOutput{}, err
	}
	if err := validateTenant(in.TenantID); err != nil {
		return AuthOutput{}, err
	}
	if err := validatePassword(in.Password); err != nil {
		return AuthOutput{}, err
	}
	passwordHash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("hash password: %w", err)
	}
	id, err := newID()
	if err != nil {
		return AuthOutput{}, err
	}
	now := s.clock.Now()
	user := domain.User{ID: id, TenantID: in.TenantID, Email: email, PasswordHash: passwordHash, IsActive: true, CreatedAt: now, UpdatedAt: now}
	if err := s.users.CreateUser(ctx, user); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			s.metrics.IncRegister("duplicate")
			return AuthOutput{}, err
		}
		return AuthOutput{}, fmt.Errorf("create user: %w", err)
	}
	s.metrics.IncRegister("success")
	s.log(ctx, user.TenantID, user.ID, "register", "success", in.IP, nil)
	out, err := s.issueAuthOutput(ctx, user)
	if err != nil {
		return AuthOutput{}, err
	}
	s.publish(ctx, domain.EventUserRegistered, user.ID, domain.UserRegisteredPayload{
		User:           eventUser(user),
		TenantID:       user.TenantID,
		Roles:          out.Roles,
		RegistrationIP: in.IP,
	})
	return out, nil
}

// Login verifies credentials and returns rotated tokens.
func (s *AuthService) Login(ctx context.Context, in LoginInput) (AuthOutput, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return AuthOutput{}, err
	}
	user, err := s.users.GetUserByEmail(ctx, in.TenantID, email)
	if err != nil {
		s.metrics.IncLogin("invalid")
		return AuthOutput{}, domain.ErrInvalidCredentials
	}
	ok, err := s.hasher.Verify(in.Password, user.PasswordHash)
	if err != nil {
		return AuthOutput{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok || !user.IsActive {
		s.metrics.IncLogin("invalid")
		s.log(ctx, in.TenantID, user.ID, "login", "failure", in.IP, nil)
		return AuthOutput{}, domain.ErrInvalidCredentials
	}
	s.metrics.IncLogin("success")
	s.log(ctx, user.TenantID, user.ID, "login", "success", in.IP, nil)
	out, err := s.issueAuthOutput(ctx, user)
	if err != nil {
		return AuthOutput{}, err
	}
	s.publish(ctx, domain.EventUserLoggedIn, user.ID, domain.UserLoggedInPayload{
		UserID:    user.ID,
		TenantID:  user.TenantID,
		Email:     user.Email,
		TokenID:   out.TokenID,
		Roles:     out.Roles,
		IP:        in.IP,
		UserAgent: in.UserAgent,
	})
	return out, nil
}

// RefreshToken rotates a refresh token and returns a new token pair.
func (s *AuthService) RefreshToken(ctx context.Context, rawRefreshToken string) (AuthOutput, error) {
	rotated, err := s.refresh.RotateRefreshToken(ctx, rawRefreshToken)
	if err != nil {
		s.metrics.IncRefresh("failure")
		return AuthOutput{}, err
	}
	user, err := s.users.GetUserByID(ctx, rotated.Token.TenantID, rotated.Token.UserID)
	if err != nil {
		return AuthOutput{}, err
	}
	out, err := s.issueAccessForUser(ctx, user)
	if err != nil {
		return AuthOutput{}, err
	}
	out.RefreshToken = rotated.Raw
	s.metrics.IncRefresh("success")
	s.log(ctx, user.TenantID, user.ID, "refresh", "success", "", nil)
	s.publish(ctx, domain.EventTokenRefreshed, user.ID, domain.TokenRefreshedPayload{
		UserID:     user.ID,
		TenantID:   user.TenantID,
		OldTokenID: rotated.PreviousTokenID,
		NewTokenID: rotated.Token.ID,
		ExpiresAt:  out.ExpiresAt.Format(time.RFC3339Nano),
	})
	return out, nil
}

// ForgotPassword creates a one-time reset token and sends it through the mailer.
func (s *AuthService) ForgotPassword(ctx context.Context, tenantID string, emailRaw string, ip string) error {
	email, err := normalizeEmail(emailRaw)
	if err != nil {
		return err
	}
	user, err := s.users.GetUserByEmail(ctx, tenantID, email)
	if err != nil {
		return nil
	}
	raw, err := randomToken(32)
	if err != nil {
		return err
	}
	id, err := newID()
	if err != nil {
		return err
	}
	now := s.clock.Now()
	token := domain.PasswordResetToken{ID: id, TenantID: tenantID, UserID: user.ID, Hash: hashToken(raw), ExpiresAt: now.Add(s.cfg.ResetTTL), CreatedAt: now}
	if err := s.resets.CreatePasswordResetToken(ctx, token); err != nil {
		return err
	}
	if s.mailer != nil {
		if err := s.mailer.SendPasswordReset(ctx, tenantID, email, raw); err != nil {
			return err
		}
	}
	s.log(ctx, tenantID, user.ID, "forgot_password", "success", ip, nil)
	s.publish(ctx, domain.EventPasswordResetRequested, email, domain.PasswordResetRequestedPayload{
		TenantID:     tenantID,
		Email:        email,
		ResetTokenID: token.ID,
		ExpiresAt:    token.ExpiresAt.Format(time.RFC3339Nano),
		IP:           ip,
	})
	return nil
}

// ResetPassword consumes a reset token and sets a new password.
func (s *AuthService) ResetPassword(ctx context.Context, tenantID string, resetToken string, newPassword string, ip string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	token, err := s.resets.GetPasswordResetTokenByHash(ctx, hashToken(resetToken))
	if err != nil || token.TenantID != tenantID {
		return domain.ErrInvalidToken
	}
	now := s.clock.Now()
	if token.ConsumedAt != nil {
		return domain.ErrResetTokenConsumed
	}
	if token.ExpiresAt.Before(now) {
		return domain.ErrTokenExpired
	}
	passwordHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := s.users.UpdatePassword(ctx, tenantID, token.UserID, passwordHash, now); err != nil {
		return err
	}
	if err := s.resets.ConsumePasswordResetToken(ctx, token.ID, now); err != nil {
		return err
	}
	s.log(ctx, tenantID, token.UserID, "reset_password", "success", ip, nil)
	user, err := s.users.GetUserByID(ctx, tenantID, token.UserID)
	if err != nil {
		return err
	}
	s.publish(ctx, domain.EventPasswordResetCompleted, token.UserID, domain.PasswordResetCompletedPayload{
		UserID:       token.UserID,
		TenantID:     tenantID,
		Email:        user.Email,
		ResetTokenID: token.ID,
		IP:           ip,
	})
	return nil
}

// ValidateToken validates an access token.
func (s *AuthService) ValidateToken(ctx context.Context, accessToken string) (domain.TokenClaims, error) {
	return s.issuer.ValidateAccessToken(ctx, accessToken)
}

// GetUserInfo returns a user after token validation.
func (s *AuthService) GetUserInfo(ctx context.Context, accessToken string) (domain.User, error) {
	claims, err := s.ValidateToken(ctx, accessToken)
	if err != nil {
		return domain.User{}, err
	}
	return s.users.GetUserByID(ctx, claims.TenantID, claims.Subject)
}

// Authorize checks a tenant-scoped permission.
func (s *AuthService) Authorize(ctx context.Context, accessToken string, permission string) (domain.TokenClaims, bool, error) {
	claims, err := s.ValidateToken(ctx, accessToken)
	if err != nil {
		return domain.TokenClaims{}, false, err
	}
	ok, err := s.rbac.UserHasPermission(ctx, claims.TenantID, claims.Subject, permission)
	if err != nil {
		return domain.TokenClaims{}, false, err
	}
	if !ok {
		s.log(ctx, claims.TenantID, claims.Subject, "authorize", "denied", "", map[string]string{"permission": permission})
		s.publish(ctx, domain.EventPermissionDenied, claims.Subject, domain.PermissionDeniedPayload{
			Subject:    claims.Subject,
			TenantID:   claims.TenantID,
			Permission: permission,
			Roles:      claims.Roles,
			TokenID:    claims.TokenID,
		})
		return claims, false, domain.ErrPermissionDenied
	}
	return claims, true, nil
}

func (s *AuthService) issueAuthOutput(ctx context.Context, user domain.User) (AuthOutput, error) {
	out, err := s.issueAccessForUser(ctx, user)
	if err != nil {
		return AuthOutput{}, err
	}
	refresh, err := s.refresh.IssueRefreshToken(ctx, user)
	if err != nil {
		return AuthOutput{}, err
	}
	out.RefreshToken = refresh.Raw
	return out, nil
}

func (s *AuthService) issueAccessForUser(ctx context.Context, user domain.User) (AuthOutput, error) {
	roles, err := s.rbac.GetUserRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return AuthOutput{}, err
	}
	roleNames := make([]string, 0, len(roles))
	permissions := map[string]struct{}{}
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
		for _, permission := range role.Permissions {
			permissions[permission.Value] = struct{}{}
		}
	}
	permissionValues := make([]string, 0, len(permissions))
	for permission := range permissions {
		permissionValues = append(permissionValues, permission)
	}
	access, err := s.issuer.IssueAccessToken(ctx, domain.TokenClaims{
		Subject:     user.ID,
		TenantID:    user.TenantID,
		Email:       user.Email,
		Roles:       roleNames,
		Permissions: permissionValues,
	})
	if err != nil {
		return AuthOutput{}, err
	}
	claims, err := s.issuer.ValidateAccessToken(ctx, access)
	if err != nil {
		return AuthOutput{}, err
	}
	return AuthOutput{User: user, AccessToken: access, ExpiresAt: claims.ExpiresAt, TokenID: claims.TokenID, Roles: roleNames}, nil
}

func (s *AuthService) log(ctx context.Context, tenantID string, userID string, action string, outcome string, ip string, metadata map[string]string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Log(ctx, domain.AuditEvent{TenantID: tenantID, UserID: userID, Action: action, Outcome: outcome, IP: ip, Metadata: metadata, CreatedAt: s.clock.Now()})
}

func (s *AuthService) publish(ctx context.Context, eventType string, key string, payload any) {
	if s.events == nil {
		return
	}
	id, err := newID()
	if err != nil {
		s.metrics.IncError("event_id")
		return
	}
	event := domain.NewEvent(id, eventType, key, s.clock.Now(), domain.EventMetaFromContext(ctx), payload)
	if err := s.events.Publish(ctx, event); err != nil {
		s.metrics.IncError("event_publish")
	}
}

func eventUser(user domain.User) domain.EventUser {
	return domain.EventUser{
		ID:        user.ID,
		TenantID:  user.TenantID,
		Email:     user.Email,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if _, err := mail.ParseAddress(email); err != nil {
		return "", fmt.Errorf("%w: invalid email", domain.ErrValidation)
	}
	return email, nil
}

func validateTenant(tenantID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return domain.ErrTenantRequired
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < 12 {
		return domain.ErrWeakPassword
	}
	return nil
}
