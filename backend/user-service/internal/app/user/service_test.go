package user

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/user"
)

type repoMock struct{ createErr error }

func (r repoMock) Ping(context.Context) error { return nil }
func (r repoMock) GetProfile(context.Context, string) (domain.Profile, error) {
	return domain.Profile{}, nil
}
func (r repoMock) BatchGetProfiles(context.Context, []string) ([]domain.Profile, error) {
	return nil, nil
}
func (r repoMock) SearchProfiles(context.Context, SearchProfilesQuery) ([]domain.Profile, string, error) {
	return nil, "", nil
}
func (r repoMock) UpdateProfile(context.Context, domain.Profile, []string) (domain.Profile, error) {
	return domain.Profile{ID: "1", DisplayName: "n"}, nil
}
func (r repoMock) CreateReview(context.Context, domain.Review) (domain.Review, error) {
	if r.createErr != nil {
		return domain.Review{}, r.createErr
	}
	return domain.Review{ID: "1", TargetProfileID: "t", AuthorProfileID: "a", Text: "x", Rating: 5}, nil
}
func (r repoMock) UpdateReview(context.Context, domain.Review, []string) (domain.Review, error) {
	return domain.Review{ID: "1"}, nil
}
func (r repoMock) ListReviews(context.Context, string, PageRequest) ([]domain.Review, string, error) {
	return nil, "", nil
}
func (r repoMock) GetReputation(context.Context, string) (domain.Reputation, error) {
	return domain.Reputation{}, nil
}

type pubMock struct{ called bool }

func (p *pubMock) Publish(context.Context, string, string, []byte) error { p.called = true; return nil }

func TestCreateReview(t *testing.T) {
	p := &pubMock{}
	svc := NewService(repoMock{}, p, "user", func() time.Time { return time.Unix(1, 0) })
	_, err := svc.CreateReview(context.Background(), domain.Review{TargetProfileID: "t", AuthorProfileID: "a", Text: "ok", Rating: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !p.called {
		t.Fatal("publisher not called")
	}
}

func TestCreateReviewValidation(t *testing.T) {
	svc := NewService(repoMock{createErr: errors.New("x")}, nil, "user", nil)
	_, err := svc.CreateReview(context.Background(), domain.Review{Rating: 10})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
