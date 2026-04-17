package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/petmatch/petmatch/internal/domain/user"
)

type Service struct {
	repo        Repository
	publisher   EventPublisher
	now         func() time.Time
	topicPrefix string
}

type PageRequest struct {
	PageSize  int32
	PageToken string
}

type SearchProfilesQuery struct {
	ProfileTypes     []domain.ProfileType
	City             string
	Query            string
	MinAverageRating float64
	IncludeSuspended bool
	Page             PageRequest
}

func NewService(repo Repository, pub EventPublisher, topicPrefix string, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, publisher: pub, topicPrefix: topicPrefix, now: now}
}

func (s *Service) GetProfile(ctx context.Context, id string) (domain.Profile, error) {
	if id == "" {
		return domain.Profile{}, domain.ErrInvalidArgument
	}
	return s.repo.GetProfile(ctx, id)
}

func (s *Service) BatchGetProfiles(ctx context.Context, ids []string) ([]domain.Profile, error) {
	if len(ids) == 0 {
		return nil, domain.ErrInvalidArgument
	}
	return s.repo.BatchGetProfiles(ctx, ids)
}

func (s *Service) SearchProfiles(ctx context.Context, q SearchProfilesQuery) ([]domain.Profile, string, error) {
	return s.repo.SearchProfiles(ctx, q)
}

func (s *Service) UpdateProfile(ctx context.Context, p domain.Profile, mask []string) (domain.Profile, error) {
	if err := p.Validate(); err != nil {
		return domain.Profile{}, err
	}
	p.UpdatedAt = s.now().UTC()
	updated, err := s.repo.UpdateProfile(ctx, p, mask)
	if err != nil {
		return domain.Profile{}, err
	}
	_ = s.publish(ctx, "profile_updated", updated.ID, updated)
	return updated, nil
}

func (s *Service) CreateReview(ctx context.Context, r domain.Review) (domain.Review, error) {
	if err := r.Validate(); err != nil {
		return domain.Review{}, err
	}
	r.ID = uuid.NewString()
	r.CreatedAt = s.now().UTC()
	r.UpdatedAt = r.CreatedAt
	created, err := s.repo.CreateReview(ctx, r)
	if err != nil {
		return domain.Review{}, err
	}
	_ = s.publish(ctx, "review_created", created.TargetProfileID, created)
	return created, nil
}

func (s *Service) UpdateReview(ctx context.Context, r domain.Review, mask []string) (domain.Review, error) {
	if r.ID == "" {
		return domain.Review{}, domain.ErrInvalidArgument
	}
	r.UpdatedAt = s.now().UTC()
	updated, err := s.repo.UpdateReview(ctx, r, mask)
	if err != nil {
		return domain.Review{}, err
	}
	return updated, nil
}

func (s *Service) ListReviews(ctx context.Context, targetProfileID string, page PageRequest) ([]domain.Review, string, error) {
	if targetProfileID == "" {
		return nil, "", domain.ErrInvalidArgument
	}
	return s.repo.ListReviews(ctx, targetProfileID, page)
}

func (s *Service) GetReputationSummary(ctx context.Context, profileID string) (domain.Reputation, error) {
	if profileID == "" {
		return domain.Reputation{}, domain.ErrInvalidArgument
	}
	return s.repo.GetReputation(ctx, profileID)
}

func (s *Service) Ping(ctx context.Context) error { return s.repo.Ping(ctx) }

func (s *Service) publish(ctx context.Context, topicSuffix, key string, payload any) error {
	if s.publisher == nil {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := s.publisher.Publish(ctx, s.topicPrefix+"."+topicSuffix, key, body); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil
}

var errPublisherUnavailable = errors.New("publisher unavailable")

func IsPublisherUnavailable(err error) bool {
	return errors.Is(err, errPublisherUnavailable)
}
