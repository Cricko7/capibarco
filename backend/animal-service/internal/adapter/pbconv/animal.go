// Package pbconv converts between domain entities and protobuf contracts.
package pbconv

import (
	"time"

	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	domain "github.com/petmatch/petmatch/internal/domain/animal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ToAnimalProfile converts a domain profile to protobuf.
func ToAnimalProfile(profile domain.Profile) *animalv1.AnimalProfile {
	var age *int32
	if profile.AgeMonths != nil {
		value := *profile.AgeMonths
		age = &value
	}
	return &animalv1.AnimalProfile{
		AnimalId:       profile.ID,
		OwnerProfileId: profile.OwnerProfileID,
		OwnerType:      commonv1.OwnerType(profile.OwnerType),
		Name:           profile.Name,
		Species:        animalv1.Species(profile.Species),
		Breed:          profile.Breed,
		Sex:            animalv1.AnimalSex(profile.Sex),
		Size:           animalv1.AnimalSize(profile.Size),
		AgeMonths:      age,
		Description:    profile.Description,
		Traits:         append([]string(nil), profile.Traits...),
		MedicalNotes:   append([]string(nil), profile.MedicalNotes...),
		Vaccinated:     profile.Vaccinated,
		Sterilized:     profile.Sterilized,
		Status:         animalv1.AnimalStatus(profile.Status),
		Location:       ToAddress(profile.Location),
		Photos:         ToPhotos(profile.Photos),
		Visibility:     commonv1.Visibility(profile.Visibility),
		Boosted:        profile.Boosted,
		BoostExpiresAt: toTimestampPtr(profile.BoostExpiresAt),
		Audit: &commonv1.AuditMetadata{
			CreatedAt: timestamppb.New(profile.Audit.CreatedAt),
			UpdatedAt: timestamppb.New(profile.Audit.UpdatedAt),
			CreatedBy: profile.Audit.CreatedBy,
			UpdatedBy: profile.Audit.UpdatedBy,
		},
	}
}

// FromAnimalProfile converts protobuf profile fields to a domain profile.
func FromAnimalProfile(profile *animalv1.AnimalProfile) domain.Profile {
	if profile == nil {
		return domain.Profile{}
	}
	return domain.Profile{
		ID:             profile.GetAnimalId(),
		OwnerProfileID: profile.GetOwnerProfileId(),
		OwnerType:      domain.OwnerType(profile.GetOwnerType()),
		Name:           profile.GetName(),
		Species:        domain.Species(profile.GetSpecies()),
		Breed:          profile.GetBreed(),
		Sex:            domain.Sex(profile.GetSex()),
		Size:           domain.Size(profile.GetSize()),
		AgeMonths:      profile.AgeMonths,
		Description:    profile.GetDescription(),
		Traits:         append([]string(nil), profile.GetTraits()...),
		MedicalNotes:   append([]string(nil), profile.GetMedicalNotes()...),
		Vaccinated:     profile.GetVaccinated(),
		Sterilized:     profile.GetSterilized(),
		Status:         domain.Status(profile.GetStatus()),
		Location:       FromAddress(profile.GetLocation()),
		Photos:         FromPhotos(profile.GetPhotos()),
		Visibility:     domain.Visibility(profile.GetVisibility()),
		Boosted:        profile.GetBoosted(),
		BoostExpiresAt: fromTimestamp(profile.GetBoostExpiresAt()),
	}
}

// ToAddress converts a domain address to protobuf.
func ToAddress(address domain.Address) *commonv1.Address {
	result := &commonv1.Address{
		CountryCode: address.CountryCode,
		Region:      address.Region,
		City:        address.City,
		District:    address.District,
	}
	if address.Latitude != nil || address.Longitude != nil || address.Geohash != "" {
		result.Location = &commonv1.GeoPoint{Geohash: address.Geohash}
		if address.Latitude != nil {
			result.Location.Latitude = *address.Latitude
		}
		if address.Longitude != nil {
			result.Location.Longitude = *address.Longitude
		}
	}
	return result
}

// FromAddress converts a protobuf address to domain.
func FromAddress(address *commonv1.Address) domain.Address {
	if address == nil {
		return domain.Address{}
	}
	result := domain.Address{
		CountryCode: address.GetCountryCode(),
		Region:      address.GetRegion(),
		City:        address.GetCity(),
		District:    address.GetDistrict(),
	}
	if address.GetLocation() != nil {
		lat := address.GetLocation().GetLatitude()
		lon := address.GetLocation().GetLongitude()
		result.Latitude = &lat
		result.Longitude = &lon
		result.Geohash = address.GetLocation().GetGeohash()
	}
	return result
}

// ToPhotos converts domain photos to protobuf.
func ToPhotos(photos []domain.Photo) []*commonv1.Photo {
	result := make([]*commonv1.Photo, 0, len(photos))
	for _, photo := range photos {
		result = append(result, ToPhoto(photo))
	}
	return result
}

// ToPhoto converts a domain photo to protobuf.
func ToPhoto(photo domain.Photo) *commonv1.Photo {
	return &commonv1.Photo{
		PhotoId:     photo.ID,
		Url:         photo.URL,
		Blurhash:    photo.Blurhash,
		Width:       photo.Width,
		Height:      photo.Height,
		ContentType: photo.ContentType,
		SortOrder:   photo.SortOrder,
		CreatedAt:   timestamppb.New(photo.CreatedAt),
	}
}

// FromPhotos converts protobuf photos to domain.
func FromPhotos(photos []*commonv1.Photo) []domain.Photo {
	result := make([]domain.Photo, 0, len(photos))
	for _, photo := range photos {
		result = append(result, FromPhoto(photo))
	}
	return result
}

// FromPhoto converts a protobuf photo to domain.
func FromPhoto(photo *commonv1.Photo) domain.Photo {
	if photo == nil {
		return domain.Photo{}
	}
	return domain.Photo{
		ID:          photo.GetPhotoId(),
		URL:         photo.GetUrl(),
		Blurhash:    photo.GetBlurhash(),
		Width:       photo.GetWidth(),
		Height:      photo.GetHeight(),
		ContentType: photo.GetContentType(),
		SortOrder:   photo.GetSortOrder(),
		CreatedAt:   timeFromProto(photo.GetCreatedAt()),
	}
}

// ToPageResponse converts cursor metadata to protobuf.
func ToPageResponse(nextPageToken string, totalSize *int32) *commonv1.PageResponse {
	return &commonv1.PageResponse{
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}
}

func toTimestampPtr(value *time.Time) *timestamppb.Timestamp {
	if value == nil || value.IsZero() {
		return nil
	}
	return timestamppb.New(*value)
}

func fromTimestamp(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	result := value.AsTime()
	return &result
}

func timeFromProto(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime()
}
