package pbconv

import (
	"strings"
	"time"

	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	domain "github.com/petmatch/petmatch/internal/domain/user"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ProfileToProto(p domain.Profile, rep domain.Reputation) *userv1.UserProfile {
	return &userv1.UserProfile{
		ProfileId:   p.ID,
		AuthUserId:  p.AuthUserID,
		ProfileType: userv1.ProfileType(p.ProfileType),
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarUrl:   p.AvatarURL,
		Address:     &commonv1.Address{City: p.City},
		Visibility:  commonv1.Visibility(p.Visibility),
		Reputation:  ReputationToProto(rep),
		Audit:       &commonv1.AuditMetadata{CreatedAt: timestamppb.New(p.CreatedAt), UpdatedAt: timestamppb.New(p.UpdatedAt)},
	}
}

func ReputationToProto(r domain.Reputation) *userv1.ReputationSummary {
	return &userv1.ReputationSummary{ProfileId: r.ProfileID, AverageRating: r.AverageRating, ReviewsCount: r.ReviewsCount, UpdatedAt: timestamppb.New(r.UpdatedAt)}
}

func ReviewToProto(r domain.Review) *userv1.Review {
	resp := &userv1.Review{ReviewId: r.ID, TargetProfileId: r.TargetProfileID, AuthorProfileId: r.AuthorProfileID, Rating: r.Rating, Text: r.Text, Visibility: commonv1.Visibility(r.Visibility), Audit: &commonv1.AuditMetadata{CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt)}}
	if strings.TrimSpace(r.MatchID) != "" {
		resp.MatchId = &r.MatchID
	}
	return resp
}

func ProfileFromProto(p *userv1.UserProfile) domain.Profile {
	if p == nil {
		return domain.Profile{}
	}
	res := domain.Profile{ID: p.GetProfileId(), AuthUserID: p.GetAuthUserId(), ProfileType: domain.ProfileType(p.GetProfileType()), DisplayName: p.GetDisplayName(), Bio: p.GetBio(), AvatarURL: p.GetAvatarUrl(), Visibility: int32(p.GetVisibility())}
	if p.GetAddress() != nil {
		res.City = p.GetAddress().GetCity()
	}
	if p.GetAudit() != nil {
		if ts := p.GetAudit().GetCreatedAt(); ts != nil {
			res.CreatedAt = ts.AsTime()
		}
		if ts := p.GetAudit().GetUpdatedAt(); ts != nil {
			res.UpdatedAt = ts.AsTime()
		}
	}
	if res.CreatedAt.IsZero() {
		res.CreatedAt = time.Now().UTC()
	}
	return res
}

func ReviewFromCreate(target, author string, rating int32, text string, matchID *string) domain.Review {
	res := domain.Review{TargetProfileID: target, AuthorProfileID: author, Rating: rating, Text: text, Visibility: int32(commonv1.Visibility_VISIBILITY_PUBLIC)}
	if matchID != nil {
		res.MatchID = *matchID
	}
	return res
}
