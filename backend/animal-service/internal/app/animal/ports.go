// Package animal contains application use cases for animal profiles.
package animal

import (
	"context"
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/animal"
)

// Repository stores animal profiles.
type Repository interface {
	Create(ctx context.Context, profile domain.Profile, idempotencyKey string) (domain.Profile, error)
	Get(ctx context.Context, id string) (domain.Profile, error)
	BatchGet(ctx context.Context, ids []string) ([]domain.Profile, error)
	Search(ctx context.Context, query domain.SearchQuery) (domain.SearchResult, error)
	Update(ctx context.Context, profile domain.Profile) (domain.Profile, error)
	RegisterIdempotency(ctx context.Context, key string, animalID string) error
}

// EventPublisher publishes animal events to the outside world.
type EventPublisher interface {
	Publish(ctx context.Context, event domain.Event) error
}

// IDGenerator creates unique identifiers.
type IDGenerator interface {
	NewID() string
}

// Clock provides the current time.
type Clock interface {
	Now() time.Time
}
