package animal

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Species identifies the broad animal species.
type Species int32

const (
	SpeciesUnspecified Species = iota
	SpeciesDog
	SpeciesCat
	SpeciesBird
	SpeciesRabbit
	SpeciesRodent
	SpeciesReptile
	SpeciesOther
)

// Sex identifies the biological sex when known.
type Sex int32

const (
	SexUnspecified Sex = iota
	SexMale
	SexFemale
	SexUnknown
)

// Size identifies an animal size bucket.
type Size int32

const (
	SizeUnspecified Size = iota
	SizeSmall
	SizeMedium
	SizeLarge
	SizeExtraLarge
)

// Status identifies an animal profile lifecycle state.
type Status int32

const (
	StatusUnspecified Status = iota
	StatusDraft
	StatusAvailable
	StatusReserved
	StatusAdopted
	StatusArchived
)

// OwnerType identifies who owns an animal profile.
type OwnerType int32

const (
	OwnerTypeUnspecified OwnerType = iota
	OwnerTypeUser
	OwnerTypeShelter
	OwnerTypeKennel
)

// Visibility controls whether an animal profile is discoverable.
type Visibility int32

const (
	VisibilityUnspecified Visibility = iota
	VisibilityPrivate
	VisibilityPublic
	VisibilityUnlisted
	VisibilitySuspended
)

// Address is public location metadata for search and display.
type Address struct {
	CountryCode string   `json:"country_code,omitempty"`
	Region      string   `json:"region,omitempty"`
	City        string   `json:"city,omitempty"`
	District    string   `json:"district,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	Geohash     string   `json:"geohash,omitempty"`
}

// Photo describes an externally stored animal image.
type Photo struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Blurhash    string    `json:"blurhash,omitempty"`
	Width       int32     `json:"width"`
	Height      int32     `json:"height"`
	ContentType string    `json:"content_type"`
	SortOrder   int32     `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuditMetadata captures creation and update identity.
type AuditMetadata struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}

// Profile is the aggregate root for animal profiles.
type Profile struct {
	ID             string
	OwnerProfileID string
	OwnerType      OwnerType
	Name           string
	Species        Species
	Breed          string
	Sex            Sex
	Size           Size
	AgeMonths      *int32
	Description    string
	Traits         []string
	MedicalNotes   []string
	Vaccinated     bool
	Sterilized     bool
	Status         Status
	Location       Address
	Photos         []Photo
	Visibility     Visibility
	Boosted        bool
	BoostExpiresAt *time.Time
	Audit          AuditMetadata
	DonationCount  int64
	InterestCount  int64
}

// Int32 returns a pointer to value and keeps tests and command builders concise.
func Int32(value int32) *int32 {
	return &value
}

// NewProfile creates a draft animal profile and validates invariant fields.
func NewProfile(profile Profile, actorID string, now time.Time) (Profile, error) {
	profile.Name = strings.TrimSpace(profile.Name)
	profile.OwnerProfileID = strings.TrimSpace(profile.OwnerProfileID)
	profile.Breed = strings.TrimSpace(profile.Breed)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.Traits = normalizeStrings(profile.Traits)
	profile.MedicalNotes = normalizeStrings(profile.MedicalNotes)
	profile.Location = normalizeAddress(profile.Location)

	if err := validateBase(profile); err != nil {
		return Profile{}, err
	}
	if profile.Status == StatusUnspecified {
		profile.Status = StatusDraft
	}
	if profile.Status != StatusDraft {
		return Profile{}, fmt.Errorf("%w: new animal must start as draft", ErrInvalidState)
	}
	if profile.Visibility == VisibilityUnspecified {
		profile.Visibility = VisibilityPrivate
	}
	if profile.Visibility != VisibilityPrivate {
		return Profile{}, fmt.Errorf("%w: new animal must start as private", ErrInvalidState)
	}

	for i := range profile.Photos {
		if err := validatePhoto(profile.Photos[i]); err != nil {
			return Profile{}, err
		}
		if profile.Photos[i].CreatedAt.IsZero() {
			profile.Photos[i].CreatedAt = now
		}
	}

	profile.Audit = AuditMetadata{
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: actorID,
		UpdatedBy: actorID,
	}

	return profile, nil
}

// ApplyPatch updates mutable profile fields described by updateMask.
func (p *Profile) ApplyPatch(patch Profile, updateMask []string, actorID string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("%w: nil profile", ErrInvalidArgument)
	}
	if len(updateMask) == 0 {
		return fmt.Errorf("%w: update mask is required", ErrInvalidArgument)
	}

	for _, rawPath := range updateMask {
		path := strings.TrimSpace(rawPath)
		switch path {
		case "name":
			p.Name = strings.TrimSpace(patch.Name)
		case "breed":
			p.Breed = strings.TrimSpace(patch.Breed)
		case "sex":
			p.Sex = patch.Sex
		case "size":
			p.Size = patch.Size
		case "age_months":
			p.AgeMonths = patch.AgeMonths
		case "description":
			p.Description = strings.TrimSpace(patch.Description)
		case "traits":
			p.Traits = normalizeStrings(patch.Traits)
		case "medical_notes":
			p.MedicalNotes = normalizeStrings(patch.MedicalNotes)
		case "vaccinated":
			p.Vaccinated = patch.Vaccinated
		case "sterilized":
			p.Sterilized = patch.Sterilized
		case "location":
			p.Location = normalizeAddress(patch.Location)
		default:
			return fmt.Errorf("%w: unsupported update mask path %q", ErrInvalidArgument, path)
		}
	}

	if err := validateBase(*p); err != nil {
		return err
	}
	p.touch(actorID, now)
	return nil
}

// Publish makes a complete draft profile searchable.
func (p *Profile) Publish(actorID string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("%w: nil profile", ErrInvalidArgument)
	}
	if p.Status != StatusDraft {
		return fmt.Errorf("%w: only draft animals can be published", ErrInvalidState)
	}
	if strings.TrimSpace(p.Description) == "" {
		return fmt.Errorf("%w: description is required to publish", ErrInvalidState)
	}
	if strings.TrimSpace(p.Location.City) == "" {
		return fmt.Errorf("%w: city is required to publish", ErrInvalidState)
	}
	if len(p.Photos) == 0 {
		return fmt.Errorf("%w: at least one photo is required to publish", ErrInvalidState)
	}
	p.Status = StatusAvailable
	p.Visibility = VisibilityPublic
	p.touch(actorID, now)
	return nil
}

// Archive hides an animal profile from discovery.
func (p *Profile) Archive(reason string, actorID string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("%w: nil profile", ErrInvalidArgument)
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: archive reason is required", ErrInvalidArgument)
	}
	if p.Status == StatusArchived {
		return fmt.Errorf("%w: animal is already archived", ErrInvalidState)
	}
	p.Status = StatusArchived
	p.Visibility = VisibilityPrivate
	p.touch(actorID, now)
	return nil
}

// AddPhoto adds a unique photo to the profile.
func (p *Profile) AddPhoto(photo Photo, actorID string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("%w: nil profile", ErrInvalidArgument)
	}
	if err := validatePhoto(photo); err != nil {
		return err
	}
	if slices.ContainsFunc(p.Photos, func(existing Photo) bool {
		return existing.ID == photo.ID
	}) {
		return fmt.Errorf("%w: photo %q already exists", ErrConflict, photo.ID)
	}
	if photo.CreatedAt.IsZero() {
		photo.CreatedAt = now
	}
	p.Photos = append(p.Photos, photo)
	p.touch(actorID, now)
	return nil
}

// RemovePhoto removes a photo by ID.
func (p *Profile) RemovePhoto(photoID string, actorID string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("%w: nil profile", ErrInvalidArgument)
	}
	photoID = strings.TrimSpace(photoID)
	if photoID == "" {
		return fmt.Errorf("%w: photo id is required", ErrInvalidArgument)
	}
	index := slices.IndexFunc(p.Photos, func(photo Photo) bool {
		return photo.ID == photoID
	})
	if index < 0 {
		return fmt.Errorf("%w: photo %q", ErrNotFound, photoID)
	}
	p.Photos = slices.Delete(p.Photos, index, index+1)
	p.touch(actorID, now)
	return nil
}

// MarkBoosted records the latest boost state for search ranking.
func (p *Profile) MarkBoosted(expiresAt time.Time, actorID string, now time.Time) error {
	if p == nil {
		return fmt.Errorf("%w: nil profile", ErrInvalidArgument)
	}
	if expiresAt.IsZero() || !expiresAt.After(now) {
		return fmt.Errorf("%w: boost expiration must be in the future", ErrInvalidArgument)
	}
	p.Boosted = true
	p.BoostExpiresAt = &expiresAt
	p.touch(actorID, now)
	return nil
}

// IncrementDonationCount tracks successful animal-targeted donations.
func (p *Profile) IncrementDonationCount(actorID string, now time.Time) {
	p.DonationCount++
	p.touch(actorID, now)
}

// IncrementInterestCount tracks adoption interest from match events.
func (p *Profile) IncrementInterestCount(actorID string, now time.Time) {
	p.InterestCount++
	p.touch(actorID, now)
}

func (p *Profile) touch(actorID string, now time.Time) {
	p.Audit.UpdatedAt = now
	p.Audit.UpdatedBy = actorID
}

func validateBase(profile Profile) error {
	if profile.OwnerProfileID == "" {
		return fmt.Errorf("%w: owner_profile_id is required", ErrInvalidArgument)
	}
	if profile.OwnerType == OwnerTypeUnspecified {
		return fmt.Errorf("%w: owner_type is required", ErrInvalidArgument)
	}
	if profile.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidArgument)
	}
	if profile.Species == SpeciesUnspecified {
		return fmt.Errorf("%w: species is required", ErrInvalidArgument)
	}
	if profile.Sex == SexUnspecified {
		return fmt.Errorf("%w: sex is required", ErrInvalidArgument)
	}
	if profile.Size == SizeUnspecified {
		return fmt.Errorf("%w: size is required", ErrInvalidArgument)
	}
	if profile.AgeMonths != nil && *profile.AgeMonths < 0 {
		return fmt.Errorf("%w: age_months must be non-negative", ErrInvalidArgument)
	}
	return nil
}

func validatePhoto(photo Photo) error {
	if strings.TrimSpace(photo.ID) == "" {
		return fmt.Errorf("%w: photo_id is required", ErrInvalidArgument)
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(photo.URL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%w: photo url must be absolute", ErrInvalidArgument)
	}
	if photo.Width <= 0 || photo.Height <= 0 {
		return fmt.Errorf("%w: photo dimensions must be positive", ErrInvalidArgument)
	}
	if strings.TrimSpace(photo.ContentType) == "" {
		return fmt.Errorf("%w: photo content_type is required", ErrInvalidArgument)
	}
	return nil
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func normalizeAddress(address Address) Address {
	address.CountryCode = strings.ToUpper(strings.TrimSpace(address.CountryCode))
	address.Region = strings.TrimSpace(address.Region)
	address.City = strings.TrimSpace(address.City)
	address.District = strings.TrimSpace(address.District)
	address.Geohash = strings.TrimSpace(address.Geohash)
	return address
}
