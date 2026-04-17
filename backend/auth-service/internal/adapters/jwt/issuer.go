package jwt

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
	"github.com/hackathon/authsvc/internal/domain"
)

// Ed25519Config configures EdDSA access tokens.
type Ed25519Config struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	Issuer     string
	Audience   string
	AccessTTL  time.Duration
	KeyID      string
}

// Ed25519Issuer issues and validates EdDSA JWT access tokens.
type Ed25519Issuer struct {
	cfg Ed25519Config
}

// NewEd25519Issuer creates an EdDSA issuer.
func NewEd25519Issuer(cfg Ed25519Config) *Ed25519Issuer {
	return &Ed25519Issuer{cfg: cfg}
}

type accessClaims struct {
	TenantID    string   `json:"tenant_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	golangjwt.RegisteredClaims
}

// IssueAccessToken signs an access token.
func (i *Ed25519Issuer) IssueAccessToken(_ context.Context, claims domain.TokenClaims) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(i.cfg.AccessTTL)
	tokenID := claims.TokenID
	if tokenID == "" {
		tokenID = fmt.Sprintf("%d", now.UnixNano())
	}
	token := golangjwt.NewWithClaims(golangjwt.SigningMethodEdDSA, accessClaims{
		TenantID:    claims.TenantID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		RegisteredClaims: golangjwt.RegisteredClaims{
			Issuer:    i.cfg.Issuer,
			Subject:   claims.Subject,
			Audience:  golangjwt.ClaimStrings{i.cfg.Audience},
			ExpiresAt: golangjwt.NewNumericDate(expiresAt),
			IssuedAt:  golangjwt.NewNumericDate(now),
			NotBefore: golangjwt.NewNumericDate(now),
			ID:        tokenID,
		},
	})
	token.Header["kid"] = i.cfg.KeyID
	signed, err := token.SignedString(i.cfg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken validates and returns access token claims.
func (i *Ed25519Issuer) ValidateAccessToken(_ context.Context, rawToken string) (domain.TokenClaims, error) {
	parsed, err := golangjwt.ParseWithClaims(rawToken, &accessClaims{}, func(token *golangjwt.Token) (any, error) {
		if token.Method != golangjwt.SigningMethodEdDSA {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
		}
		return i.cfg.PublicKey, nil
	}, golangjwt.WithIssuer(i.cfg.Issuer), golangjwt.WithAudience(i.cfg.Audience))
	if err != nil {
		if errors.Is(err, golangjwt.ErrTokenExpired) {
			return domain.TokenClaims{}, domain.ErrTokenExpired
		}
		return domain.TokenClaims{}, fmt.Errorf("%w: %v", domain.ErrInvalidToken, err)
	}
	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid {
		return domain.TokenClaims{}, domain.ErrInvalidToken
	}
	return domain.TokenClaims{
		Subject:     claims.Subject,
		TenantID:    claims.TenantID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		ExpiresAt:   claims.ExpiresAt.Time,
		IssuedAt:    claims.IssuedAt.Time,
		TokenID:     claims.ID,
	}, nil
}
