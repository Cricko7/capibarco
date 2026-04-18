package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
	"github.com/stretchr/testify/require"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
)

func TestServiceGetFeedPropagatesPrincipalAndCapsPageSize(t *testing.T) {
	auth := &fakeAuth{}
	feed := &fakeFeed{}
	svc := NewService(Dependencies{
		Auth:          auth,
		Feed:          feed,
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", FeedPrefetchSize: 7, MaxPageSize: 10},
	})
	ctx := WithPrincipal(context.Background(), Principal{
		ActorID:     "user-1",
		TenantID:    "tenant-1",
		Roles:       []string{"user"},
		Permissions: []string{"feed:read"},
	})

	out, err := svc.GetFeed(ctx, GetFeedInput{Surface: feedv1.FeedSurface_FEED_SURFACE_MAIN, PageSize: 50})
	require.NoError(t, err)
	require.Equal(t, "session-1", out.FeedSessionID)
	require.Equal(t, int32(10), feed.last.Page.PageSize)
	require.Equal(t, "user-1", feed.last.Principal.ActorId)
	require.Equal(t, commonv1.ActorType_ACTOR_TYPE_USER, feed.last.Principal.ActorType)
	require.False(t, feed.last.Principal.IsGuest)
}

func TestServiceSwipeAnimalRequiresIdempotencyKey(t *testing.T) {
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		Matching:      &fakeMatching{},
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})
	ctx := WithPrincipal(context.Background(), Principal{ActorID: "user-1", TenantID: "tenant-1"})

	_, err := svc.SwipeAnimal(ctx, SwipeAnimalInput{AnimalID: "animal-1", OwnerProfileID: "owner-1", Direction: matchingv1.SwipeDirection_SWIPE_DIRECTION_RIGHT})
	require.ErrorIs(t, err, ErrIdempotencyKeyRequired)
}

func TestServiceUploadAnimalPhotoRequiresPrincipal(t *testing.T) {
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		Animal:        &fakeAnimal{},
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})

	_, err := svc.UploadAnimalPhoto(context.Background(), UploadAnimalPhotoInput{
		AnimalID:       "animal-1",
		Photo:          &commonv1.Photo{PhotoId: "photo-1", Url: "https://cdn.example/photo.jpg"},
		IdempotencyKey: "idem-photo",
	})
	require.ErrorIs(t, err, ErrUnauthenticated)
}

func TestServiceCreateGuestSessionReturnsTokenAndScopes(t *testing.T) {
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})

	out, err := svc.CreateGuestSession(context.Background(), CreateGuestSessionInput{DeviceID: "device-1", Locale: "ru-RU"})
	require.NoError(t, err)
	require.NotEmpty(t, out.Token)
	require.Equal(t, []string{"feed:read", "animal:read", "swipe:create"}, out.AllowedScopes)
	require.Equal(t, fixedNow.Add(time.Hour), out.ExpiresAt)
}

func TestServiceCreateAnimalUsesProvidedOwnerType(t *testing.T) {
	animalClient := &fakeAnimal{}
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		Animal:        animalClient,
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})
	ctx := WithPrincipal(context.Background(), Principal{ActorID: "user-1", TenantID: "tenant-1"})

	_, err := svc.CreateAnimal(ctx, CreateAnimalInput{
		IdempotencyKey: "idem-animal",
		Animal: &animalv1.AnimalProfile{
			OwnerProfileId: "profile-kennel-1",
			OwnerType:      commonv1.OwnerType_OWNER_TYPE_KENNEL,
			Name:           "Mila",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, animalClient.lastCreate)
	require.Equal(t, commonv1.OwnerType_OWNER_TYPE_KENNEL, animalClient.lastCreate.OwnerType)
	require.Equal(t, "profile-kennel-1", animalClient.lastCreate.OwnerProfileId)
}

func TestServiceCreateAnimalPublishesWhenRequested(t *testing.T) {
	animalClient := &fakeAnimal{}
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		Animal:        animalClient,
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})
	ctx := WithPrincipal(context.Background(), Principal{ActorID: "profile-1", TenantID: "tenant-1"})

	animal, err := svc.CreateAnimal(ctx, CreateAnimalInput{
		IdempotencyKey: "idem-animal",
		Animal: &animalv1.AnimalProfile{
			OwnerProfileId: "profile-1",
			Name:           "Mila",
			Status:         animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE,
			Visibility:     commonv1.Visibility_VISIBILITY_PUBLIC,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, animal)
	require.Equal(t, animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE, animal.Status)
	require.Equal(t, "animal-1", animalClient.lastPublishAnimalID)
}

func TestServiceCreateConversationUsesCurrentPrincipal(t *testing.T) {
	chatClient := &fakeChat{}
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		Chat:          chatClient,
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})
	ctx := WithPrincipal(context.Background(), Principal{ActorID: "profile-actor", TenantID: "tenant-1"})

	_, err := svc.CreateConversation(ctx, CreateConversationInput{
		TargetProfileID: "profile-target",
		IdempotencyKey:  "idem-chat",
	})
	require.NoError(t, err)
	require.NotNil(t, chatClient.lastCreateConversation)
	require.Equal(t, "profile-actor", chatClient.lastCreateConversation.AdopterProfileId)
	require.Equal(t, "profile-target", chatClient.lastCreateConversation.OwnerProfileId)
	require.Empty(t, chatClient.lastCreateConversation.MatchId)
	require.Empty(t, chatClient.lastCreateConversation.AnimalId)
}

func TestServiceListOwnerAnimalsUsesRequestedProfile(t *testing.T) {
	animalClient := &fakeAnimal{}
	svc := NewService(Dependencies{
		Auth:          &fakeAuth{},
		Animal:        animalClient,
		GuestSessions: domain.NewGuestSessionCodec([]byte("secret"), time.Hour),
		Clock:         fixedClock{},
		Defaults:      Defaults{TenantID: "petmatch", MaxPageSize: 10},
	})
	ctx := WithPrincipal(context.Background(), Principal{ActorID: "viewer-1", TenantID: "tenant-1"})

	_, err := svc.ListOwnerAnimals(ctx, ListOwnerAnimalsInput{
		OwnerProfileID: "owner-1",
		PageSize:       99,
	})
	require.NoError(t, err)
	require.NotNil(t, animalClient.lastListOwner)
	require.Equal(t, "owner-1", animalClient.lastListOwner.OwnerProfileId)
	require.Equal(t, int32(10), animalClient.lastListOwner.Page.PageSize)
}

type fixedClock struct{}

var fixedNow = time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

func (fixedClock) Now() time.Time { return fixedNow }

type fakeAuth struct{}

func (fakeAuth) Register(context.Context, RegisterInput) (*AuthResponse, error) {
	return &AuthResponse{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: fixedNow.Add(time.Hour).Format(time.RFC3339)}, nil
}

func (fakeAuth) Login(context.Context, LoginInput) (*AuthResponse, error) {
	return &AuthResponse{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: fixedNow.Add(time.Hour).Format(time.RFC3339)}, nil
}

func (fakeAuth) Refresh(context.Context, RefreshInput) (*AuthResponse, error) {
	return &AuthResponse{AccessToken: "access", RefreshToken: "refresh-2", ExpiresAt: fixedNow.Add(time.Hour).Format(time.RFC3339)}, nil
}

func (fakeAuth) Validate(context.Context, string) (Principal, error) {
	return Principal{ActorID: "user-1", TenantID: "tenant-1"}, nil
}

func (fakeAuth) Authorize(context.Context, string, string) (Principal, bool, error) {
	return Principal{ActorID: "user-1", TenantID: "tenant-1"}, true, nil
}

type fakeFeed struct {
	last *feedv1.GetFeedRequest
}

func (f *fakeFeed) GetFeed(_ context.Context, req *feedv1.GetFeedRequest) (*feedv1.GetFeedResponse, error) {
	f.last = req
	return &feedv1.GetFeedResponse{FeedSessionId: "session-1"}, nil
}

type fakeMatching struct{}

func (fakeMatching) RecordSwipe(context.Context, *matchingv1.RecordSwipeRequest) (*matchingv1.RecordSwipeResponse, error) {
	return nil, errors.New("should not be called")
}

type fakeAnimal struct {
	lastCreate          *animalv1.CreateAnimalRequest
	lastListOwner       *animalv1.ListOwnerAnimalsRequest
	lastPublishAnimalID string
}

func (fakeAnimal) GetAnimal(context.Context, string) (*animalv1.AnimalProfile, error) {
	return nil, nil
}
func (f *fakeAnimal) CreateAnimal(_ context.Context, req *animalv1.CreateAnimalRequest) (*animalv1.AnimalProfile, error) {
	f.lastCreate = req
	if req.Animal != nil && req.Animal.AnimalId == "" {
		req.Animal.AnimalId = "animal-1"
	}
	return req.Animal, nil
}
func (f *fakeAnimal) PublishAnimal(_ context.Context, animalID string) (*animalv1.AnimalProfile, error) {
	f.lastPublishAnimalID = animalID
	return &animalv1.AnimalProfile{
		AnimalId: animalID,
		Status:   animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE,
	}, nil
}
func (fakeAnimal) AddPhoto(context.Context, string, *commonv1.Photo, string) (*animalv1.AnimalProfile, error) {
	return nil, nil
}
func (f *fakeAnimal) ListOwnerAnimals(_ context.Context, req *animalv1.ListOwnerAnimalsRequest) (*animalv1.ListOwnerAnimalsResponse, error) {
	f.lastListOwner = req
	return &animalv1.ListOwnerAnimalsResponse{}, nil
}

type fakeChat struct {
	lastCreateConversation *chatv1.CreateConversationRequest
}

func (f *fakeChat) CreateConversation(_ context.Context, req *chatv1.CreateConversationRequest) (*chatv1.CreateConversationResponse, error) {
	f.lastCreateConversation = req
	return &chatv1.CreateConversationResponse{Conversation: &chatv1.Conversation{ConversationId: "conversation-1"}}, nil
}
func (fakeChat) ListConversations(context.Context, *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error) {
	return nil, nil
}
func (fakeChat) ListMessages(context.Context, *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	return nil, nil
}
func (fakeChat) SendMessage(context.Context, *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	return nil, nil
}

type fakeBilling struct{}

func (fakeBilling) CreateDonationIntent(context.Context, *billingv1.CreateDonationIntentRequest) (*billingv1.CreateDonationIntentResponse, error) {
	return nil, nil
}

type fakeAnalytics struct{}

func (fakeAnalytics) GetAnimalStats(context.Context, *analyticsv1.GetAnimalStatsRequest) (*analyticsv1.GetAnimalStatsResponse, error) {
	return nil, nil
}
