package animal_test

import (
	"errors"
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/domain/animal"
	"github.com/stretchr/testify/require"
)

func TestNewProfileValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile animal.Profile
		wantErr error
	}{
		{
			name: "missing owner",
			profile: animal.Profile{
				Name:    "Mars",
				Species: animal.SpeciesDog,
				Sex:     animal.SexMale,
				Size:    animal.SizeMedium,
			},
			wantErr: animal.ErrInvalidArgument,
		},
		{
			name: "unspecified species",
			profile: animal.Profile{
				OwnerProfileID: "owner-1",
				Name:           "Mars",
				Sex:            animal.SexMale,
				Size:           animal.SizeMedium,
			},
			wantErr: animal.ErrInvalidArgument,
		},
		{
			name: "negative age",
			profile: animal.Profile{
				OwnerProfileID: "owner-1",
				Name:           "Mars",
				Species:        animal.SpeciesDog,
				Sex:            animal.SexMale,
				Size:           animal.SizeMedium,
				AgeMonths:      animal.Int32(-1),
			},
			wantErr: animal.ErrInvalidArgument,
		},
		{
			name: "valid minimal animal",
			profile: animal.Profile{
				OwnerProfileID: "owner-1",
				OwnerType:      animal.OwnerTypeShelter,
				Name:           "Mars",
				Species:        animal.SpeciesDog,
				Sex:            animal.SexMale,
				Size:           animal.SizeMedium,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := animal.NewProfile(tt.profile, "actor-1", time.Unix(1700000000, 0).UTC())
			if tt.wantErr != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tt.wantErr), "got %v, want %v", err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, animal.StatusDraft, got.Status)
			require.Equal(t, animal.VisibilityPrivate, got.Visibility)
			require.Equal(t, "actor-1", got.Audit.CreatedBy)
			require.Equal(t, got.Audit.CreatedAt, got.Audit.UpdatedAt)
		})
	}
}

func TestPublishRequiresCompleteProfile(t *testing.T) {
	t.Parallel()

	profile, err := animal.NewProfile(animal.Profile{
		ID:             "animal-1",
		OwnerProfileID: "owner-1",
		OwnerType:      animal.OwnerTypeShelter,
		Name:           "Luna",
		Species:        animal.SpeciesCat,
		Sex:            animal.SexFemale,
		Size:           animal.SizeSmall,
	}, "actor-1", time.Unix(1700000000, 0).UTC())
	require.NoError(t, err)

	err = profile.Publish("actor-1", time.Unix(1700000001, 0).UTC())
	require.Error(t, err)
	require.True(t, errors.Is(err, animal.ErrInvalidState))

	profile.Description = "Calm cat looking for a quiet family"
	profile.Location.City = "Moscow"
	profile.AddPhoto(animal.Photo{
		ID:          "photo-1",
		URL:         "https://cdn.example.test/photo.jpg",
		ContentType: "image/jpeg",
		Width:       800,
		Height:      600,
	}, "actor-1", time.Unix(1700000002, 0).UTC())

	require.NoError(t, profile.Publish("actor-1", time.Unix(1700000003, 0).UTC()))
	require.Equal(t, animal.StatusAvailable, profile.Status)
	require.Equal(t, animal.VisibilityPublic, profile.Visibility)
	require.Equal(t, "actor-1", profile.Audit.UpdatedBy)
}

func TestPhotoLifecycleRejectsDuplicatesAndMissingPhotos(t *testing.T) {
	t.Parallel()

	profile, err := animal.NewProfile(animal.Profile{
		ID:             "animal-1",
		OwnerProfileID: "owner-1",
		OwnerType:      animal.OwnerTypeShelter,
		Name:           "Luna",
		Species:        animal.SpeciesCat,
		Sex:            animal.SexFemale,
		Size:           animal.SizeSmall,
	}, "actor-1", time.Unix(1700000000, 0).UTC())
	require.NoError(t, err)

	photo := animal.Photo{
		ID:          "photo-1",
		URL:         "https://cdn.example.test/photo.jpg",
		ContentType: "image/jpeg",
		Width:       800,
		Height:      600,
	}

	require.NoError(t, profile.AddPhoto(photo, "actor-1", time.Unix(1700000001, 0).UTC()))
	require.Len(t, profile.Photos, 1)

	err = profile.AddPhoto(photo, "actor-1", time.Unix(1700000002, 0).UTC())
	require.Error(t, err)
	require.True(t, errors.Is(err, animal.ErrConflict))

	err = profile.RemovePhoto("missing", "actor-1", time.Unix(1700000003, 0).UTC())
	require.Error(t, err)
	require.True(t, errors.Is(err, animal.ErrNotFound))

	require.NoError(t, profile.RemovePhoto("photo-1", "actor-1", time.Unix(1700000004, 0).UTC()))
	require.Empty(t, profile.Photos)
}
