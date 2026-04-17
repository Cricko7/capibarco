package animal

import (
	"context"
	"fmt"
	"strings"

	domain "github.com/petmatch/petmatch/internal/domain/animal"
)

const defaultPageSize int32 = 50

// Service coordinates animal profile use cases.
type Service struct {
	repo      Repository
	publisher EventPublisher
	ids       IDGenerator
	clock     Clock
}

// NewService builds an animal application service.
func NewService(repo Repository, publisher EventPublisher, ids IDGenerator, clock Clock) *Service {
	return &Service{
		repo:      repo,
		publisher: publisher,
		ids:       ids,
		clock:     clock,
	}
}

// Create creates a draft animal profile and publishes animal.profile_created once.
func (s *Service) Create(ctx context.Context, cmd CreateAnimalCommand) (domain.Profile, error) {
	if err := ctx.Err(); err != nil {
		return domain.Profile{}, err
	}
	if strings.TrimSpace(cmd.ActorID) == "" {
		return domain.Profile{}, fmt.Errorf("%w: actor_id is required", domain.ErrInvalidArgument)
	}

	now := s.clock.Now()
	profile, err := domain.NewProfile(domain.Profile{
		ID:             s.ids.NewID(),
		OwnerProfileID: cmd.OwnerProfileID,
		OwnerType:      cmd.OwnerType,
		Name:           cmd.Name,
		Species:        cmd.Species,
		Breed:          cmd.Breed,
		Sex:            cmd.Sex,
		Size:           cmd.Size,
		AgeMonths:      cmd.AgeMonths,
		Description:    cmd.Description,
		Traits:         cmd.Traits,
		MedicalNotes:   cmd.MedicalNotes,
		Vaccinated:     cmd.Vaccinated,
		Sterilized:     cmd.Sterilized,
		Location:       cmd.Location,
		Photos:         cmd.Photos,
	}, cmd.ActorID, now)
	if err != nil {
		return domain.Profile{}, err
	}

	if cmd.InitialPhotoURL != "" {
		photo := domain.Photo{
			ID:          s.ids.NewID(),
			URL:         cmd.InitialPhotoURL,
			ContentType: "image/jpeg",
			Width:       1,
			Height:      1,
		}
		if err := profile.AddPhoto(photo, cmd.ActorID, now); err != nil {
			return domain.Profile{}, err
		}
	}

	created, err := s.repo.Create(ctx, profile, cmd.IdempotencyKey)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("create animal profile: %w", err)
	}
	if created.ID != profile.ID {
		return created, nil
	}

	event := s.newEvent(domain.EventProfileCreated, created)
	event.IdempotencyKey = cmd.IdempotencyKey
	if err := s.publisher.Publish(ctx, event); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal created event: %w", err)
	}

	return created, nil
}

// Get returns an animal profile by ID.
func (s *Service) Get(ctx context.Context, id string) (domain.Profile, error) {
	if strings.TrimSpace(id) == "" {
		return domain.Profile{}, fmt.Errorf("%w: animal_id is required", domain.ErrInvalidArgument)
	}
	profile, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("get animal %q: %w", id, err)
	}
	return profile, nil
}

// BatchGet returns animals by ID without failing on missing IDs.
func (s *Service) BatchGet(ctx context.Context, ids []string) ([]domain.Profile, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: animal_ids are required", domain.ErrInvalidArgument)
	}
	profiles, err := s.repo.BatchGet(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("batch get animals: %w", err)
	}
	return profiles, nil
}

// Search returns animal profiles matching query filters.
func (s *Service) Search(ctx context.Context, query domain.SearchQuery) (domain.SearchResult, error) {
	if query.PageSize <= 0 {
		query.PageSize = defaultPageSize
	}
	result, err := s.repo.Search(ctx, query)
	if err != nil {
		return domain.SearchResult{}, fmt.Errorf("search animals: %w", err)
	}
	return result, nil
}

// ListOwnerAnimals returns profiles owned by one profile ID.
func (s *Service) ListOwnerAnimals(ctx context.Context, ownerProfileID string, statuses []domain.Status, pageSize int32, pageToken string) (domain.SearchResult, error) {
	ownerProfileID = strings.TrimSpace(ownerProfileID)
	if ownerProfileID == "" {
		return domain.SearchResult{}, fmt.Errorf("%w: owner_profile_id is required", domain.ErrInvalidArgument)
	}
	return s.Search(ctx, domain.SearchQuery{
		OwnerProfileID: ownerProfileID,
		Statuses:       statuses,
		PageSize:       pageSize,
		PageToken:      pageToken,
	})
}

// Update patches mutable animal profile fields and publishes animal.profile_updated.
func (s *Service) Update(ctx context.Context, cmd UpdateAnimalCommand) (domain.Profile, error) {
	profile, err := s.getOwned(ctx, cmd.AnimalID, cmd.ActorID)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := profile.ApplyPatch(cmd.Patch, cmd.UpdateMask, cmd.ActorID, s.clock.Now()); err != nil {
		return domain.Profile{}, err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("update animal %q: %w", cmd.AnimalID, err)
	}
	if err := s.publisher.Publish(ctx, s.newEvent(domain.EventProfileUpdated, updated)); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal updated event: %w", err)
	}
	return updated, nil
}

// Publish changes a draft profile to available and emits status/published events.
func (s *Service) Publish(ctx context.Context, cmd PublishAnimalCommand) (domain.Profile, error) {
	profile, err := s.getOwned(ctx, cmd.AnimalID, cmd.ActorID)
	if err != nil {
		return domain.Profile{}, err
	}
	oldStatus := profile.Status
	if err := profile.Publish(cmd.ActorID, s.clock.Now()); err != nil {
		return domain.Profile{}, err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal %q: %w", cmd.AnimalID, err)
	}
	statusEvent := s.newEvent(domain.EventStatusChanged, updated)
	statusEvent.OldStatus = oldStatus
	statusEvent.NewStatus = updated.Status
	statusEvent.Reason = "published"
	if err := s.publisher.Publish(ctx, statusEvent); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal status changed event: %w", err)
	}
	if err := s.publisher.Publish(ctx, s.newEvent(domain.EventProfilePublished, updated)); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal published event: %w", err)
	}
	return updated, nil
}

// Archive hides an animal profile and emits archived/status events.
func (s *Service) Archive(ctx context.Context, cmd ArchiveAnimalCommand) (domain.Profile, error) {
	profile, err := s.getOwned(ctx, cmd.AnimalID, cmd.ActorID)
	if err != nil {
		return domain.Profile{}, err
	}
	oldStatus := profile.Status
	if err := profile.Archive(cmd.Reason, cmd.ActorID, s.clock.Now()); err != nil {
		return domain.Profile{}, err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("archive animal %q: %w", cmd.AnimalID, err)
	}
	statusEvent := s.newEvent(domain.EventStatusChanged, updated)
	statusEvent.OldStatus = oldStatus
	statusEvent.NewStatus = updated.Status
	statusEvent.Reason = cmd.Reason
	if err := s.publisher.Publish(ctx, statusEvent); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal status changed event: %w", err)
	}
	archivedEvent := s.newEvent(domain.EventProfileArchived, updated)
	archivedEvent.OldStatus = oldStatus
	archivedEvent.Reason = cmd.Reason
	if err := s.publisher.Publish(ctx, archivedEvent); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal archived event: %w", err)
	}
	return updated, nil
}

// AddPhoto appends a unique photo and publishes animal.photo_added once.
func (s *Service) AddPhoto(ctx context.Context, cmd AddPhotoCommand) (domain.Profile, error) {
	profile, err := s.getOwned(ctx, cmd.AnimalID, cmd.ActorID)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := profile.AddPhoto(cmd.Photo, cmd.ActorID, s.clock.Now()); err != nil {
		return domain.Profile{}, err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("add animal photo: %w", err)
	}
	if err := s.repo.RegisterIdempotency(ctx, cmd.IdempotencyKey, updated.ID); err != nil {
		return domain.Profile{}, fmt.Errorf("register photo idempotency: %w", err)
	}
	event := s.newEvent(domain.EventPhotoAdded, updated)
	event.Photo = &cmd.Photo
	event.IdempotencyKey = cmd.IdempotencyKey
	if err := s.publisher.Publish(ctx, event); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal photo added event: %w", err)
	}
	return updated, nil
}

// RemovePhoto removes a photo and publishes animal.profile_updated.
func (s *Service) RemovePhoto(ctx context.Context, cmd RemovePhotoCommand) (domain.Profile, error) {
	profile, err := s.getOwned(ctx, cmd.AnimalID, cmd.ActorID)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := profile.RemovePhoto(cmd.PhotoID, cmd.ActorID, s.clock.Now()); err != nil {
		return domain.Profile{}, err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("remove animal photo: %w", err)
	}
	if err := s.publisher.Publish(ctx, s.newEvent(domain.EventProfileUpdated, updated)); err != nil {
		return domain.Profile{}, fmt.Errorf("publish animal updated event: %w", err)
	}
	return updated, nil
}

// ApplyBoostActivated marks an animal as boosted from a billing event.
func (s *Service) ApplyBoostActivated(ctx context.Context, cmd BoostActivatedCommand) error {
	profile, err := s.repo.Get(ctx, cmd.AnimalID)
	if err != nil {
		return fmt.Errorf("get boosted animal %q: %w", cmd.AnimalID, err)
	}
	if err := profile.MarkBoosted(cmd.ExpiresAt, cmd.ActorID, s.clock.Now()); err != nil {
		return err
	}
	updated, err := s.repo.Update(ctx, profile)
	if err != nil {
		return fmt.Errorf("update boosted animal %q: %w", cmd.AnimalID, err)
	}
	event := s.newEvent(domain.EventStatusChanged, updated)
	event.Reason = "boost_activated"
	if err := s.publisher.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish boost status event: %w", err)
	}
	return nil
}

// ApplyDonationSucceeded increments animal donation counters.
func (s *Service) ApplyDonationSucceeded(ctx context.Context, cmd DonationSucceededCommand) error {
	profile, err := s.repo.Get(ctx, cmd.AnimalID)
	if err != nil {
		return fmt.Errorf("get donated animal %q: %w", cmd.AnimalID, err)
	}
	profile.IncrementDonationCount(cmd.ActorID, s.clock.Now())
	if _, err := s.repo.Update(ctx, profile); err != nil {
		return fmt.Errorf("increment donation count for animal %q: %w", cmd.AnimalID, err)
	}
	return nil
}

// ApplyMatchCreated increments animal adoption interest counters.
func (s *Service) ApplyMatchCreated(ctx context.Context, cmd MatchCreatedCommand) error {
	profile, err := s.repo.Get(ctx, cmd.AnimalID)
	if err != nil {
		return fmt.Errorf("get matched animal %q: %w", cmd.AnimalID, err)
	}
	profile.IncrementInterestCount(cmd.ActorID, s.clock.Now())
	if _, err := s.repo.Update(ctx, profile); err != nil {
		return fmt.Errorf("increment interest count for animal %q: %w", cmd.AnimalID, err)
	}
	return nil
}

func (s *Service) getOwned(ctx context.Context, animalID string, actorID string) (domain.Profile, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return domain.Profile{}, fmt.Errorf("%w: actor_id is required", domain.ErrInvalidArgument)
	}
	profile, err := s.Get(ctx, animalID)
	if err != nil {
		return domain.Profile{}, err
	}
	if profile.OwnerProfileID != actorID {
		return domain.Profile{}, fmt.Errorf("%w: actor %q does not own animal %q", domain.ErrForbidden, actorID, animalID)
	}
	return profile, nil
}

func (s *Service) newEvent(eventType string, profile domain.Profile) domain.Event {
	return domain.Event{
		ID:             s.ids.NewID(),
		Type:           eventType,
		AnimalID:       profile.ID,
		OwnerProfileID: profile.OwnerProfileID,
		Animal:         &profile,
		OccurredAt:     s.clock.Now(),
	}
}
