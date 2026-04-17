package animal

import (
	"time"

	"github.com/google/uuid"
)

// UUIDGenerator creates UUIDv7 identifiers when possible.
type UUIDGenerator struct{}

// NewID returns a unique identifier.
func (UUIDGenerator) NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

// SystemClock reads wall-clock time in UTC.
type SystemClock struct{}

// Now returns current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
