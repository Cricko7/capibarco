package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/petmatch/chat-service/internal/domain/chat"
)

func TestMapErrorMapsInvalidUUIDToNotFound(t *testing.T) {
	err := mapError(&pgconn.PgError{Code: "22P02"})
	if !errors.Is(err, chat.ErrNotFound) {
		t.Fatalf("mapError(invalid uuid) = %v, want %v", err, chat.ErrNotFound)
	}
}
