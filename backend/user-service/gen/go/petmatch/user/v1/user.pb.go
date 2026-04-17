package userv1

import (
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProfileType int32

const (
	ProfileType_PROFILE_TYPE_UNSPECIFIED ProfileType = 0
	ProfileType_PROFILE_TYPE_USER        ProfileType = 1
	ProfileType_PROFILE_TYPE_SHELTER     ProfileType = 2
	ProfileType_PROFILE_TYPE_KENNEL      ProfileType = 3
)

type UserProfile struct {
	ProfileId, AuthUserId, DisplayName, Bio, AvatarUrl string
	ProfileType                                        ProfileType
	Address                                            *commonv1.Address
	Visibility                                         commonv1.Visibility
	Reputation                                         *ReputationSummary
	Audit                                              *commonv1.AuditMetadata
}

func (m *UserProfile) GetProfileId() string {
	if m == nil {
		return ""
	}
	return m.ProfileId
}
func (m *UserProfile) GetAuthUserId() string {
	if m == nil {
		return ""
	}
	return m.AuthUserId
}
func (m *UserProfile) GetProfileType() ProfileType {
	if m == nil {
		return 0
	}
	return m.ProfileType
}
func (m *UserProfile) GetDisplayName() string {
	if m == nil {
		return ""
	}
	return m.DisplayName
}
func (m *UserProfile) GetBio() string {
	if m == nil {
		return ""
	}
	return m.Bio
}
func (m *UserProfile) GetAvatarUrl() string {
	if m == nil {
		return ""
	}
	return m.AvatarUrl
}
func (m *UserProfile) GetAddress() *commonv1.Address {
	if m == nil {
		return nil
	}
	return m.Address
}
func (m *UserProfile) GetVisibility() commonv1.Visibility {
	if m == nil {
		return 0
	}
	return m.Visibility
}
func (m *UserProfile) GetReputation() *ReputationSummary {
	if m == nil {
		return nil
	}
	return m.Reputation
}
func (m *UserProfile) GetAudit() *commonv1.AuditMetadata {
	if m == nil {
		return nil
	}
	return m.Audit
}

type ReputationSummary struct {
	ProfileId                             string
	AverageRating                         float64
	ReviewsCount, CompletedAdoptionsCount int32
	UpdatedAt                             *timestamppb.Timestamp
}

func (m *ReputationSummary) GetProfileId() string {
	if m == nil {
		return ""
	}
	return m.ProfileId
}

type Review struct {
	ReviewId, TargetProfileId, AuthorProfileId string
	Rating                                     int32
	Text                                       string
	MatchId                                    *string
	Visibility                                 commonv1.Visibility
	Audit                                      *commonv1.AuditMetadata
}

func (m *Review) GetReviewId() string {
	if m == nil {
		return ""
	}
	return m.ReviewId
}
func (m *Review) GetTargetProfileId() string {
	if m == nil {
		return ""
	}
	return m.TargetProfileId
}
func (m *Review) GetAuthorProfileId() string {
	if m == nil {
		return ""
	}
	return m.AuthorProfileId
}
func (m *Review) GetRating() int32 {
	if m == nil {
		return 0
	}
	return m.Rating
}
func (m *Review) GetText() string {
	if m == nil {
		return ""
	}
	return m.Text
}
func (m *Review) GetVisibility() commonv1.Visibility {
	if m == nil {
		return 0
	}
	return m.Visibility
}

type ProfileFilter struct {
	ProfileTypes     []ProfileType
	City             *string
	MinAverageRating *float64
	Query            *string
	IncludeSuspended bool
}

func (m *ProfileFilter) GetProfileTypes() []ProfileType {
	if m == nil {
		return nil
	}
	return m.ProfileTypes
}
func (m *ProfileFilter) GetCity() string {
	if m == nil || m.City == nil {
		return ""
	}
	return *m.City
}
func (m *ProfileFilter) GetMinAverageRating() float64 {
	if m == nil || m.MinAverageRating == nil {
		return 0
	}
	return *m.MinAverageRating
}
func (m *ProfileFilter) GetQuery() string {
	if m == nil || m.Query == nil {
		return ""
	}
	return *m.Query
}
func (m *ProfileFilter) GetIncludeSuspended() bool {
	if m == nil {
		return false
	}
	return m.IncludeSuspended
}

type GetProfileRequest struct{ ProfileId string }

func (m *GetProfileRequest) GetProfileId() string {
	if m == nil {
		return ""
	}
	return m.ProfileId
}

type GetProfileResponse struct{ Profile *UserProfile }

type BatchGetProfilesRequest struct{ ProfileIds []string }

func (m *BatchGetProfilesRequest) GetProfileIds() []string {
	if m == nil {
		return nil
	}
	return m.ProfileIds
}

type BatchGetProfilesResponse struct{ Profiles []*UserProfile }

type SearchProfilesRequest struct {
	Filter *ProfileFilter
	Page   *commonv1.PageRequest
}

func (m *SearchProfilesRequest) GetFilter() *ProfileFilter {
	if m == nil {
		return nil
	}
	return m.Filter
}
func (m *SearchProfilesRequest) GetPage() *commonv1.PageRequest {
	if m == nil {
		return nil
	}
	return m.Page
}

type SearchProfilesResponse struct {
	Profiles []*UserProfile
	Page     *commonv1.PageResponse
}

type UpdateProfileRequest struct {
	ProfileId  string
	Profile    *UserProfile
	UpdateMask *fieldmaskpb.FieldMask
}

func (m *UpdateProfileRequest) GetProfile() *UserProfile {
	if m == nil {
		return nil
	}
	return m.Profile
}
func (m *UpdateProfileRequest) GetUpdateMask() *fieldmaskpb.FieldMask {
	if m == nil {
		return nil
	}
	return m.UpdateMask
}

type UpdateProfileResponse struct{ Profile *UserProfile }

type CreateReviewRequest struct {
	TargetProfileId, AuthorProfileId string
	Rating                           int32
	Text                             string
	MatchId                          *string
}

func (m *CreateReviewRequest) GetTargetProfileId() string {
	if m == nil {
		return ""
	}
	return m.TargetProfileId
}
func (m *CreateReviewRequest) GetAuthorProfileId() string {
	if m == nil {
		return ""
	}
	return m.AuthorProfileId
}
func (m *CreateReviewRequest) GetRating() int32 {
	if m == nil {
		return 0
	}
	return m.Rating
}
func (m *CreateReviewRequest) GetText() string {
	if m == nil {
		return ""
	}
	return m.Text
}

type CreateReviewResponse struct{ Review *Review }

type UpdateReviewRequest struct {
	ReviewId   string
	Review     *Review
	UpdateMask *fieldmaskpb.FieldMask
}

func (m *UpdateReviewRequest) GetReviewId() string {
	if m == nil {
		return ""
	}
	return m.ReviewId
}
func (m *UpdateReviewRequest) GetReview() *Review {
	if m == nil {
		return nil
	}
	return m.Review
}
func (m *UpdateReviewRequest) GetUpdateMask() *fieldmaskpb.FieldMask {
	if m == nil {
		return nil
	}
	return m.UpdateMask
}

type UpdateReviewResponse struct{ Review *Review }

type ListReviewsRequest struct {
	TargetProfileId string
	Page            *commonv1.PageRequest
}

func (m *ListReviewsRequest) GetTargetProfileId() string {
	if m == nil {
		return ""
	}
	return m.TargetProfileId
}
func (m *ListReviewsRequest) GetPage() *commonv1.PageRequest {
	if m == nil {
		return nil
	}
	return m.Page
}

type ListReviewsResponse struct {
	Reviews []*Review
	Page    *commonv1.PageResponse
}

type GetReputationSummaryRequest struct{ ProfileId string }

func (m *GetReputationSummaryRequest) GetProfileId() string {
	if m == nil {
		return ""
	}
	return m.ProfileId
}

type GetReputationSummaryResponse struct{ Reputation *ReputationSummary }
