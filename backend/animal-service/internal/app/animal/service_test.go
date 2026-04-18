package animal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appanimal "github.com/petmatch/petmatch/internal/app/animal"
	"github.com/petmatch/petmatch/internal/domain/animal"
	"github.com/stretchr/testify/require"
)

func TestServiceCreateAnimalIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepository()
	publisher := &recordingPublisher{}
	clock := fixedClock(time.Unix(1700000000, 0).UTC())
	ids := &sequenceIDs{values: []string{"animal-1"}}
	service := appanimal.NewService(repo, publisher, ids, clock)

	req := appanimal.CreateAnimalCommand{
		ActorID:         "actor-1",
		OwnerProfileID:  "owner-1",
		OwnerType:       animal.OwnerTypeShelter,
		IdempotencyKey:  "idem-1",
		Name:            "Luna",
		Species:         animal.SpeciesCat,
		Sex:             animal.SexFemale,
		Size:            animal.SizeSmall,
		Description:     "Quiet cat",
		Location:        animal.Address{City: "Moscow"},
		InitialPhotoURL: "https://cdn.example.test/luna.jpg",
	}

	created, err := service.Create(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "animal-1", created.ID)
	require.Equal(t, animal.StatusDraft, created.Status)

	again, err := service.Create(ctx, req)
	require.NoError(t, err)
	require.Equal(t, created.ID, again.ID)
	require.Len(t, repo.animals, 1)
	require.Len(t, publisher.events, 1, "idempotent replay must not publish duplicate events")
	require.Equal(t, animal.EventProfileCreated, publisher.events[0].Type)
}

func TestServiceUpdateAppliesOnlyAllowedFieldMaskPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepository()
	publisher := &recordingPublisher{}
	service := appanimal.NewService(repo, publisher, &sequenceIDs{values: []string{"animal-1"}}, fixedClock(time.Unix(1700000000, 0).UTC()))

	created, err := service.Create(ctx, appanimal.CreateAnimalCommand{
		ActorID:        "owner-1",
		OwnerProfileID: "owner-1",
		OwnerType:      animal.OwnerTypeShelter,
		IdempotencyKey: "idem-1",
		Name:           "Luna",
		Species:        animal.SpeciesCat,
		Sex:            animal.SexFemale,
		Size:           animal.SizeSmall,
	})
	require.NoError(t, err)

	updated, err := service.Update(ctx, appanimal.UpdateAnimalCommand{
		ActorID:  "owner-1",
		AnimalID: created.ID,
		Patch: animal.Profile{
			Name:        "Mila",
			Species:     animal.SpeciesDog,
			Description: "Gentle cat",
			Status:      animal.StatusArchived,
		},
		UpdateMask: []string{"name", "species", "description"},
	})
	require.NoError(t, err)
	require.Equal(t, "Mila", updated.Name)
	require.Equal(t, animal.SpeciesDog, updated.Species)
	require.Equal(t, "Gentle cat", updated.Description)
	require.Equal(t, animal.StatusDraft, updated.Status, "status must not be patched through general update")

	_, err = service.Update(ctx, appanimal.UpdateAnimalCommand{
		ActorID:    "owner-1",
		AnimalID:   created.ID,
		Patch:      animal.Profile{Name: "Bad"},
		UpdateMask: []string{"status"},
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, animal.ErrInvalidArgument))
}

func TestServicePublishEmitsStatusAndPublishedEvents(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepository()
	publisher := &recordingPublisher{}
	service := appanimal.NewService(repo, publisher, &sequenceIDs{values: []string{"animal-1"}}, fixedClock(time.Unix(1700000000, 0).UTC()))

	created, err := service.Create(ctx, appanimal.CreateAnimalCommand{
		ActorID:        "owner-1",
		OwnerProfileID: "owner-1",
		OwnerType:      animal.OwnerTypeShelter,
		IdempotencyKey: "idem-1",
		Name:           "Luna",
		Species:        animal.SpeciesCat,
		Sex:            animal.SexFemale,
		Size:           animal.SizeSmall,
		Description:    "Gentle cat",
		Location:       animal.Address{City: "Moscow"},
	})
	require.NoError(t, err)

	_, err = service.AddPhoto(ctx, appanimal.AddPhotoCommand{
		ActorID:        "owner-1",
		AnimalID:       created.ID,
		IdempotencyKey: "photo-idem-1",
		Photo: animal.Photo{
			ID:          "photo-1",
			URL:         "https://cdn.example.test/photo.jpg",
			ContentType: "image/jpeg",
			Width:       800,
			Height:      600,
		},
	})
	require.NoError(t, err)

	published, err := service.Publish(ctx, appanimal.PublishAnimalCommand{
		ActorID:  "owner-1",
		AnimalID: created.ID,
	})
	require.NoError(t, err)
	require.Equal(t, animal.StatusAvailable, published.Status)

	require.Len(t, publisher.events, 4)
	require.Equal(t, animal.EventProfileCreated, publisher.events[0].Type)
	require.Equal(t, animal.EventPhotoAdded, publisher.events[1].Type)
	require.Equal(t, animal.EventStatusChanged, publisher.events[2].Type)
	require.Equal(t, animal.EventProfilePublished, publisher.events[3].Type)
}

func TestServiceRejectsNonOwnerMutations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryRepository()
	service := appanimal.NewService(repo, &recordingPublisher{}, &sequenceIDs{values: []string{"animal-1"}}, fixedClock(time.Unix(1700000000, 0).UTC()))

	created, err := service.Create(ctx, appanimal.CreateAnimalCommand{
		ActorID:        "owner-1",
		OwnerProfileID: "owner-1",
		OwnerType:      animal.OwnerTypeShelter,
		IdempotencyKey: "idem-1",
		Name:           "Luna",
		Species:        animal.SpeciesCat,
		Sex:            animal.SexFemale,
		Size:           animal.SizeSmall,
	})
	require.NoError(t, err)

	_, err = service.Archive(ctx, appanimal.ArchiveAnimalCommand{
		ActorID:  "intruder",
		AnimalID: created.ID,
		Reason:   "test",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, animal.ErrForbidden))
}

type fixedClock time.Time

func (c fixedClock) Now() time.Time {
	return time.Time(c)
}

type sequenceIDs struct {
	values []string
	next   int
}

func (s *sequenceIDs) NewID() string {
	if s.next >= len(s.values) {
		return "extra-id"
	}
	value := s.values[s.next]
	s.next++
	return value
}

type recordingPublisher struct {
	events []animal.Event
}

func (p *recordingPublisher) Publish(ctx context.Context, event animal.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.events = append(p.events, event)
	return nil
}

type memoryRepository struct {
	animals         map[string]animal.Profile
	idempotencyKeys map[string]string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		animals:         make(map[string]animal.Profile),
		idempotencyKeys: make(map[string]string),
	}
}

func (r *memoryRepository) Create(ctx context.Context, profile animal.Profile, idempotencyKey string) (animal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return animal.Profile{}, err
	}
	if existingID, ok := r.idempotencyKeys[idempotencyKey]; ok {
		return r.animals[existingID], nil
	}
	r.animals[profile.ID] = profile
	if idempotencyKey != "" {
		r.idempotencyKeys[idempotencyKey] = profile.ID
	}
	return profile, nil
}

func (r *memoryRepository) Get(ctx context.Context, id string) (animal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return animal.Profile{}, err
	}
	profile, ok := r.animals[id]
	if !ok {
		return animal.Profile{}, animal.ErrNotFound
	}
	return profile, nil
}

func (r *memoryRepository) BatchGet(ctx context.Context, ids []string) ([]animal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]animal.Profile, 0, len(ids))
	for _, id := range ids {
		if profile, ok := r.animals[id]; ok {
			result = append(result, profile)
		}
	}
	return result, nil
}

func (r *memoryRepository) Search(ctx context.Context, query animal.SearchQuery) (animal.SearchResult, error) {
	if err := ctx.Err(); err != nil {
		return animal.SearchResult{}, err
	}
	result := animal.SearchResult{Items: make([]animal.Profile, 0, len(r.animals))}
	for _, profile := range r.animals {
		if query.OwnerProfileID != "" && profile.OwnerProfileID != query.OwnerProfileID {
			continue
		}
		result.Items = append(result.Items, profile)
	}
	return result, nil
}

func (r *memoryRepository) Update(ctx context.Context, profile animal.Profile) (animal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return animal.Profile{}, err
	}
	if _, ok := r.animals[profile.ID]; !ok {
		return animal.Profile{}, animal.ErrNotFound
	}
	r.animals[profile.ID] = profile
	return profile, nil
}

func (r *memoryRepository) RegisterIdempotency(ctx context.Context, key string, animalID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if key != "" {
		r.idempotencyKeys[key] = animalID
	}
	return nil
}
