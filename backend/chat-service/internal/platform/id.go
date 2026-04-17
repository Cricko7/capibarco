package platform

import "github.com/google/uuid"

// UUIDGenerator creates UUIDv7 identifiers for sortable records.
type UUIDGenerator struct{}

// NewID returns a UUIDv7 string, falling back to UUIDv4 on entropy failures.
func (UUIDGenerator) NewID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
