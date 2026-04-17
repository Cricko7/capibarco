package animal

// SearchQuery contains repository-level search filters.
type SearchQuery struct {
	Species        []Species
	Breeds         []string
	Sexes          []Sex
	Sizes          []Size
	MinAgeMonths   *int32
	MaxAgeMonths   *int32
	City           *string
	NearLatitude   *float64
	NearLongitude  *float64
	RadiusKM       *int32
	Statuses       []Status
	Traits         []string
	Vaccinated     *bool
	Sterilized     *bool
	BoostedOnly    *bool
	OwnerProfileID string
	PageSize       int32
	PageToken      string
}

// SearchResult contains profiles plus cursor metadata.
type SearchResult struct {
	Items         []Profile
	NextPageToken string
	TotalSize     *int32
}
