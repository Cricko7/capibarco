package postgres

import (
	"errors"
	"testing"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreateUserErrorMapsUnknownTenantToValidation(t *testing.T) {
	err := createUserError(&pgconn.PgError{
		Code:           "23503",
		ConstraintName: "users_tenant_id_fkey",
	})

	if !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("expected validation error for unknown tenant, got %v", err)
	}
}
