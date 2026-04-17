package animal

import "time"

const (
	// EventProfileCreated is published after a profile is created.
	EventProfileCreated = "animal.profile_created"
	// EventProfileUpdated is published after a profile is updated.
	EventProfileUpdated = "animal.profile_updated"
	// EventProfilePublished is published after a profile becomes public.
	EventProfilePublished = "animal.profile_published"
	// EventProfileArchived is published after a profile is archived.
	EventProfileArchived = "animal.profile_archived"
	// EventPhotoAdded is published after a photo is added.
	EventPhotoAdded = "animal.photo_added"
	// EventStatusChanged is published after a lifecycle status change.
	EventStatusChanged = "animal.status_changed"
)

// Event is a domain event produced by animal-service.
type Event struct {
	ID             string
	Type           string
	AnimalID       string
	OwnerProfileID string
	Animal         *Profile
	Photo          *Photo
	OldStatus      Status
	NewStatus      Status
	Reason         string
	IdempotencyKey string
	TraceID        string
	CorrelationID  string
	OccurredAt     time.Time
}
