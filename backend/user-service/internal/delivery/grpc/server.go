package grpc

import (
	"context"

	"github.com/go-playground/validator/v10"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	"github.com/petmatch/petmatch/internal/adapter/pbconv"
	app "github.com/petmatch/petmatch/internal/app/user"
	domain "github.com/petmatch/petmatch/internal/domain/user"
)

type Server struct {
	userv1.UnimplementedUserServiceServer
	service  *app.Service
	validate *validator.Validate
}

func NewServer(service *app.Service) *Server {
	return &Server{service: service, validate: validator.New()}
}

func (s *Server) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.GetProfileResponse, error) {
	if err := s.validate.Var(req.GetProfileId(), "required"); err != nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	p, err := s.service.GetProfile(ctx, req.GetProfileId())
	if err != nil {
		return nil, toStatusError(err)
	}
	rep, _ := s.service.GetReputationSummary(ctx, p.ID)
	return &userv1.GetProfileResponse{Profile: pbconv.ProfileToProto(p, rep)}, nil
}

func (s *Server) BatchGetProfiles(ctx context.Context, req *userv1.BatchGetProfilesRequest) (*userv1.BatchGetProfilesResponse, error) {
	profiles, err := s.service.BatchGetProfiles(ctx, req.GetProfileIds())
	if err != nil {
		return nil, toStatusError(err)
	}
	out := make([]*userv1.UserProfile, 0, len(profiles))
	for _, p := range profiles {
		rep, _ := s.service.GetReputationSummary(ctx, p.ID)
		out = append(out, pbconv.ProfileToProto(p, rep))
	}
	return &userv1.BatchGetProfilesResponse{Profiles: out}, nil
}

func (s *Server) SearchProfiles(ctx context.Context, req *userv1.SearchProfilesRequest) (*userv1.SearchProfilesResponse, error) {
	filter := req.GetFilter()
	q := app.SearchProfilesQuery{Page: app.PageRequest{PageSize: req.GetPage().GetPageSize(), PageToken: req.GetPage().GetPageToken()}}
	if filter != nil {
		q.City = filter.GetCity()
		q.Query = filter.GetQuery()
		q.MinAverageRating = filter.GetMinAverageRating()
		q.IncludeSuspended = filter.GetIncludeSuspended()
		for _, pt := range filter.GetProfileTypes() {
			q.ProfileTypes = append(q.ProfileTypes, toDomainType(pt))
		}
	}
	profiles, next, err := s.service.SearchProfiles(ctx, q)
	if err != nil {
		return nil, toStatusError(err)
	}
	out := make([]*userv1.UserProfile, 0, len(profiles))
	for _, p := range profiles {
		rep, _ := s.service.GetReputationSummary(ctx, p.ID)
		out = append(out, pbconv.ProfileToProto(p, rep))
	}
	return &userv1.SearchProfilesResponse{Profiles: out, Page: &commonv1.PageResponse{NextPageToken: next}}, nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	mask := []string{}
	if req.GetUpdateMask() != nil {
		mask = req.GetUpdateMask().GetPaths()
	}
	updated, err := s.service.UpdateProfile(ctx, pbconv.ProfileFromProto(req.GetProfile()), mask)
	if err != nil {
		return nil, toStatusError(err)
	}
	rep, _ := s.service.GetReputationSummary(ctx, updated.ID)
	return &userv1.UpdateProfileResponse{Profile: pbconv.ProfileToProto(updated, rep)}, nil
}

func (s *Server) CreateReview(ctx context.Context, req *userv1.CreateReviewRequest) (*userv1.CreateReviewResponse, error) {
	review, err := s.service.CreateReview(ctx, pbconv.ReviewFromCreate(req.GetTargetProfileId(), req.GetAuthorProfileId(), req.GetRating(), req.GetText(), req.MatchId))
	if err != nil {
		return nil, toStatusError(err)
	}
	return &userv1.CreateReviewResponse{Review: pbconv.ReviewToProto(review)}, nil
}

func (s *Server) UpdateReview(ctx context.Context, req *userv1.UpdateReviewRequest) (*userv1.UpdateReviewResponse, error) {
	in := req.GetReview()
	if in == nil {
		return nil, toStatusError(domain.ErrInvalidArgument)
	}
	r := pbconv.ReviewFromCreate(in.GetTargetProfileId(), in.GetAuthorProfileId(), in.GetRating(), in.GetText(), in.MatchId)
	r.ID = req.GetReviewId()
	r.Visibility = int32(in.GetVisibility())
	mask := []string{}
	if req.GetUpdateMask() != nil {
		mask = req.GetUpdateMask().GetPaths()
	}
	updated, err := s.service.UpdateReview(ctx, r, mask)
	if err != nil {
		return nil, toStatusError(err)
	}
	return &userv1.UpdateReviewResponse{Review: pbconv.ReviewToProto(updated)}, nil
}

func (s *Server) ListReviews(ctx context.Context, req *userv1.ListReviewsRequest) (*userv1.ListReviewsResponse, error) {
	reviews, next, err := s.service.ListReviews(ctx, req.GetTargetProfileId(), app.PageRequest{PageSize: req.GetPage().GetPageSize(), PageToken: req.GetPage().GetPageToken()})
	if err != nil {
		return nil, toStatusError(err)
	}
	out := make([]*userv1.Review, 0, len(reviews))
	for _, rv := range reviews {
		out = append(out, pbconv.ReviewToProto(rv))
	}
	return &userv1.ListReviewsResponse{Reviews: out, Page: &commonv1.PageResponse{NextPageToken: next}}, nil
}

func (s *Server) GetReputationSummary(ctx context.Context, req *userv1.GetReputationSummaryRequest) (*userv1.GetReputationSummaryResponse, error) {
	rep, err := s.service.GetReputationSummary(ctx, req.GetProfileId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &userv1.GetReputationSummaryResponse{Reputation: pbconv.ReputationToProto(rep)}, nil
}

func toDomainType(p userv1.ProfileType) domain.ProfileType { return domain.ProfileType(p) }
