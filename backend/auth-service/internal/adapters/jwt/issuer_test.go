package jwt_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	jwtadapter "github.com/hackathon/authsvc/internal/adapters/jwt"
	"github.com/hackathon/authsvc/internal/domain"
)

func TestEd25519IssuerCreatesAndValidatesAccessToken(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	issuer := jwtadapter.NewEd25519Issuer(jwtadapter.Ed25519Config{
		PrivateKey: priv,
		PublicKey:  pub,
		Issuer:     "authsvc",
		Audience:   "internal-services",
		AccessTTL:  15 * time.Minute,
		KeyID:      "test-key",
	})

	token, err := issuer.IssueAccessToken(context.Background(), domain.TokenClaims{
		Subject:  "user-1",
		TenantID: "tenant-1",
		Email:    "user@example.com",
		Roles:    []string{"admin"},
	})
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	claims, err := issuer.ValidateAccessToken(context.Background(), token)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.Subject != "user-1" || claims.TenantID != "tenant-1" || claims.Email != "user@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Fatalf("unexpected roles: %+v", claims.Roles)
	}
}
