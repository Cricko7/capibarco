package animal

import (
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/animal"
)

// CreateAnimalCommand creates a draft animal profile.
type CreateAnimalCommand struct {
	ActorID         string
	OwnerProfileID  string
	OwnerType       domain.OwnerType
	IdempotencyKey  string
	Name            string
	Species         domain.Species
	Breed           string
	Sex             domain.Sex
	Size            domain.Size
	AgeMonths       *int32
	Description     string
	Traits          []string
	MedicalNotes    []string
	Vaccinated      bool
	Sterilized      bool
	Location        domain.Address
	Photos          []domain.Photo
	InitialPhotoURL string
}

// UpdateAnimalCommand updates mutable fields described by UpdateMask.
type UpdateAnimalCommand struct {
	ActorID    string
	AnimalID   string
	Patch      domain.Profile
	UpdateMask []string
}

// PublishAnimalCommand publishes an animal profile.
type PublishAnimalCommand struct {
	ActorID  string
	AnimalID string
}

// ArchiveAnimalCommand archives an animal profile.
type ArchiveAnimalCommand struct {
	ActorID  string
	AnimalID string
	Reason   string
}

// AddPhotoCommand adds an animal photo.
type AddPhotoCommand struct {
	ActorID        string
	AnimalID       string
	IdempotencyKey string
	Photo          domain.Photo
}

// RemovePhotoCommand removes an animal photo.
type RemovePhotoCommand struct {
	ActorID  string
	AnimalID string
	PhotoID  string
}

// BoostActivatedCommand applies billing boost events.
type BoostActivatedCommand struct {
	ActorID   string
	AnimalID  string
	ExpiresAt time.Time
}

// DonationSucceededCommand applies successful donation events.
type DonationSucceededCommand struct {
	ActorID  string
	AnimalID string
}

// MatchCreatedCommand applies successful match events.
type MatchCreatedCommand struct {
	ActorID  string
	AnimalID string
}
