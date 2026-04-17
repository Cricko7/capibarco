package kafka

import (
	"context"

	domain "github.com/petmatch/petmatch/internal/domain/animal"
)

// NoopPublisher drops events when Kafka is disabled for local tests.
type NoopPublisher struct{}

// Publish implements the application event publisher port.
func (NoopPublisher) Publish(context.Context, domain.Event) error {
	return nil
}
