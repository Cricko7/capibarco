package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	domain "github.com/petmatch/petmatch/internal/domain/user"
)

func TestMapErrorMapsForeignKeyViolationToNotFound(t *testing.T) {
	err := mapError(&pgconn.PgError{Code: "23503"})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("mapError(foreign key) = %v, want %v", err, domain.ErrNotFound)
	}
}
