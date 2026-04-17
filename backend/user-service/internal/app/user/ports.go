package user

import (
	"context"

	domain "github.com/petmatch/petmatch/internal/domain/user"
)

type Repository interface {
	Ping(ctx context.Context) error
	GetProfile(ctx context.Context, id string) (domain.Profile, error)
	BatchGetProfiles(ctx context.Context, ids []string) ([]domain.Profile, error)
	SearchProfiles(ctx context.Context, q SearchProfilesQuery) ([]domain.Profile, string, error)
	UpdateProfile(ctx context.Context, p domain.Profile, updateMask []string) (domain.Profile, error)
	CreateReview(ctx context.Context, r domain.Review) (domain.Review, error)
	UpdateReview(ctx context.Context, r domain.Review, updateMask []string) (domain.Review, error)
	ListReviews(ctx context.Context, targetProfileID string, page PageRequest) ([]domain.Review, string, error)
	GetReputation(ctx context.Context, profileID string) (domain.Reputation, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}
