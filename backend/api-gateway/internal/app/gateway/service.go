package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/petmatch/petmatch/internal/domain"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
)

// Dependencies contains application ports.
type Dependencies struct {
	Auth          AuthClient
	Feed          FeedClient
	Animal        AnimalClient
	Matching      MatchingClient
	Chat          ChatClient
	Billing       BillingClient
	Analytics     AnalyticsClient
	Notification  NotificationClient
	GuestSessions *domain.GuestSessionCodec
	Clock         Clock
	Defaults      Defaults
}

// Service orchestrates internal services without owning domain business logic.
type Service struct {
	deps Dependencies
}

// NewService creates a gateway application service.
func NewService(deps Dependencies) *Service {
	if deps.Clock == nil {
		deps.Clock = systemClock{}
	}
	if deps.Defaults.MaxPageSize <= 0 {
		deps.Defaults.MaxPageSize = 50
	}
	if deps.Defaults.FeedPrefetchSize <= 0 {
		deps.Defaults.FeedPrefetchSize = 10
	}
	return &Service{deps: deps}
}

// CreateGuestSession creates a signed anonymous session.
func (s *Service) CreateGuestSession(ctx context.Context, input CreateGuestSessionInput) (CreateGuestSessionOutput, error) {
	if input.DeviceID == "" {
		return CreateGuestSessionOutput{}, fmt.Errorf("%w: device_id is required", ErrInvalidInput)
	}
	token, session, err := s.deps.GuestSessions.Create(input.DeviceID, input.Locale, s.deps.Clock.Now())
	if err != nil {
		return CreateGuestSessionOutput{}, fmt.Errorf("create guest session: %w", err)
	}
	_ = ctx
	return CreateGuestSessionOutput{Token: token, ExpiresAt: session.ExpiresAt, AllowedScopes: session.AllowedScopes}, nil
}

// Register proxies mobile registration to auth-service.
func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	if input.TenantID == "" {
		input.TenantID = s.deps.Defaults.TenantID
	}
	out, err := s.deps.Auth.Register(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("register via auth-service: %w", err)
	}
	return out, nil
}

// Login proxies mobile credentials to auth-service.
func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	if input.TenantID == "" {
		input.TenantID = s.deps.Defaults.TenantID
	}
	out, err := s.deps.Auth.Login(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("login via auth-service: %w", err)
	}
	return out, nil
}

// ValidateBearer validates a JWT through auth-service.
func (s *Service) ValidateBearer(ctx context.Context, token string) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthenticated
	}
	principal, err := s.deps.Auth.Validate(ctx, token)
	if err != nil {
		return Principal{}, fmt.Errorf("validate bearer token: %w", err)
	}
	return principal, nil
}

// ValidateGuest validates a signed guest token.
func (s *Service) ValidateGuest(token string) (Principal, error) {
	session, err := s.deps.GuestSessions.Parse(token, s.deps.Clock.Now())
	if err != nil {
		return Principal{}, err
	}
	return Principal{
		ActorID:     session.ActorID,
		TenantID:    s.deps.Defaults.TenantID,
		Roles:       []string{"guest"},
		Permissions: session.AllowedScopes,
		IsGuest:     true,
	}, nil
}

// GetFeed returns prefetch-sized feed cards for smooth mobile swiping.
func (s *Service) GetFeed(ctx context.Context, input GetFeedInput) (GetFeedOutput, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return GetFeedOutput{}, err
	}
	pageSize := capPageSize(input.PageSize, s.deps.Defaults.FeedPrefetchSize, s.deps.Defaults.MaxPageSize)
	out, err := s.deps.Feed.GetFeed(ctx, &feedv1.GetFeedRequest{
		Principal: toProtoPrincipal(principal),
		Surface:   input.Surface,
		Filter:    input.Filter,
		Page:      &commonv1.PageRequest{PageSize: pageSize, PageToken: input.PageToken},
	})
	if err != nil {
		return GetFeedOutput{}, fmt.Errorf("get feed: %w", err)
	}
	return GetFeedOutput{
		Cards:         out.Cards,
		NextPageToken: out.Page.GetNextPageToken(),
		FeedSessionID: out.FeedSessionId,
	}, nil
}

// GetAnimal returns an animal profile.
func (s *Service) GetAnimal(ctx context.Context, animalID string) (*animalv1.AnimalProfile, error) {
	if animalID == "" {
		return nil, fmt.Errorf("%w: animal_id is required", ErrInvalidInput)
	}
	animal, err := s.deps.Animal.GetAnimal(ctx, animalID)
	if err != nil {
		return nil, fmt.Errorf("get animal: %w", err)
	}
	return animal, nil
}

// CreateAnimal proxies animal creation and injects owner identity.
func (s *Service) CreateAnimal(ctx context.Context, input CreateAnimalInput) (*animalv1.AnimalProfile, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input.IdempotencyKey == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	if input.Animal == nil {
		return nil, fmt.Errorf("%w: animal is required", ErrInvalidInput)
	}
	req := &animalv1.CreateAnimalRequest{
		OwnerProfileId: principal.ActorID,
		OwnerType:      commonv1.OwnerType_OWNER_TYPE_USER,
		Animal:         input.Animal,
		IdempotencyKey: input.IdempotencyKey,
	}
	if input.Animal.OwnerProfileId != "" {
		req.OwnerProfileId = input.Animal.OwnerProfileId
	}
	out, err := s.deps.Animal.CreateAnimal(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create animal: %w", err)
	}
	return out, nil
}

// UploadAnimalPhoto stores photo metadata in animal-service after object upload.
func (s *Service) UploadAnimalPhoto(ctx context.Context, input UploadAnimalPhotoInput) (*animalv1.AnimalProfile, error) {
	if input.IdempotencyKey == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	if input.AnimalID == "" || input.Photo == nil {
		return nil, fmt.Errorf("%w: animal_id and photo are required", ErrInvalidInput)
	}
	out, err := s.deps.Animal.AddPhoto(ctx, input.AnimalID, input.Photo, input.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("add animal photo: %w", err)
	}
	return out, nil
}

// SwipeAnimal records a mobile swipe through matching-service.
func (s *Service) SwipeAnimal(ctx context.Context, input SwipeAnimalInput) (SwipeAnimalOutput, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return SwipeAnimalOutput{}, err
	}
	if input.IdempotencyKey == "" {
		return SwipeAnimalOutput{}, ErrIdempotencyKeyRequired
	}
	req := &matchingv1.RecordSwipeRequest{
		Principal:      toProtoPrincipal(principal),
		AnimalId:       input.AnimalID,
		OwnerProfileId: input.OwnerProfileID,
		Direction:      input.Direction,
		FeedCardId:     input.FeedCardID,
		FeedSessionId:  input.FeedSessionID,
		IdempotencyKey: input.IdempotencyKey,
	}
	out, err := s.deps.Matching.RecordSwipe(ctx, req)
	if err != nil {
		return SwipeAnimalOutput{}, fmt.Errorf("record swipe: %w", err)
	}
	return SwipeAnimalOutput{Swipe: out.Swipe, Match: out.Match, ConversationID: out.ConversationId}, nil
}

// ListConversations lists chat conversations for the current actor.
func (s *Service) ListConversations(ctx context.Context, input ListConversationsInput) (*chatv1.ListConversationsResponse, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return s.deps.Chat.ListConversations(ctx, &chatv1.ListConversationsRequest{
		ParticipantProfileId: principal.ActorID,
		Page:                 &commonv1.PageRequest{PageSize: capPageSize(input.PageSize, 20, s.deps.Defaults.MaxPageSize), PageToken: input.PageToken},
	})
}

// ListMessages lists chat messages.
func (s *Service) ListMessages(ctx context.Context, input ListMessagesInput) (*chatv1.ListMessagesResponse, error) {
	if _, err := requiredPrincipal(ctx); err != nil {
		return nil, err
	}
	return s.deps.Chat.ListMessages(ctx, &chatv1.ListMessagesRequest{
		ConversationId: input.ConversationID,
		Page:           &commonv1.PageRequest{PageSize: capPageSize(input.PageSize, 30, s.deps.Defaults.MaxPageSize), PageToken: input.PageToken},
	})
}

// SendMessage sends a chat message as the current actor.
func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*chatv1.SendMessageResponse, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input.IdempotencyKey == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	return s.deps.Chat.SendMessage(ctx, &chatv1.SendMessageRequest{
		ConversationId:  input.ConversationID,
		SenderProfileId: principal.ActorID,
		Type:            input.Type,
		Text:            input.Text,
		ClientMessageId: input.ClientMessageID,
		IdempotencyKey:  input.IdempotencyKey,
	})
}

// CreateDonationIntent creates a billing donation intent for the current actor.
func (s *Service) CreateDonationIntent(ctx context.Context, input CreateDonationIntentInput) (*billingv1.CreateDonationIntentResponse, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if input.IdempotencyKey == "" {
		return nil, ErrIdempotencyKeyRequired
	}
	return s.deps.Billing.CreateDonationIntent(ctx, &billingv1.CreateDonationIntentRequest{
		PayerProfileId: principal.ActorID,
		TargetType:     input.TargetType,
		TargetId:       input.TargetID,
		Amount:         input.Amount,
		Provider:       input.Provider,
		IdempotencyKey: input.IdempotencyKey,
	})
}

// GetAnimalStats returns animal analytics.
func (s *Service) GetAnimalStats(ctx context.Context, input GetAnimalStatsInput) (*analyticsv1.GetAnimalStatsResponse, error) {
	if _, err := requiredPrincipal(ctx); err != nil {
		return nil, err
	}
	return s.deps.Analytics.GetAnimalStats(ctx, &analyticsv1.GetAnimalStatsRequest{AnimalId: input.AnimalID, TimeRange: input.TimeRange, Bucket: input.Bucket})
}

// RegisterDevice registers a mobile push token with notification-service.
func (s *Service) RegisterDevice(ctx context.Context, input RegisterDeviceInput) (*notificationv1.RegisterDeviceResponse, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if s.deps.Notification == nil {
		return nil, ErrDependencyDisabled
	}
	return s.deps.Notification.RegisterDevice(ctx, &notificationv1.RegisterDeviceRequest{
		ProfileId: principal.ActorID,
		Token:     input.Token,
		Platform:  input.Platform,
		Locale:    input.Locale,
	})
}

// UnregisterDevice removes a mobile push token registration.
func (s *Service) UnregisterDevice(ctx context.Context, deviceTokenID string) (*notificationv1.UnregisterDeviceResponse, error) {
	if _, err := requiredPrincipal(ctx); err != nil {
		return nil, err
	}
	if s.deps.Notification == nil {
		return nil, ErrDependencyDisabled
	}
	return s.deps.Notification.UnregisterDevice(ctx, &notificationv1.UnregisterDeviceRequest{DeviceTokenId: deviceTokenID})
}

// ListNotifications returns the current actor's notification inbox.
func (s *Service) ListNotifications(ctx context.Context, input ListNotificationsInput) (*notificationv1.ListNotificationsResponse, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if s.deps.Notification == nil {
		return nil, ErrDependencyDisabled
	}
	return s.deps.Notification.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{
		RecipientProfileId: principal.ActorID,
		Statuses:           input.Statuses,
		Page:               &commonv1.PageRequest{PageSize: capPageSize(input.PageSize, 30, s.deps.Defaults.MaxPageSize), PageToken: input.PageToken},
	})
}

// MarkNotificationRead marks one notification read for the current actor.
func (s *Service) MarkNotificationRead(ctx context.Context, notificationID string) (*notificationv1.MarkNotificationReadResponse, error) {
	principal, err := requiredPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	if s.deps.Notification == nil {
		return nil, ErrDependencyDisabled
	}
	return s.deps.Notification.MarkNotificationRead(ctx, &notificationv1.MarkNotificationReadRequest{NotificationId: notificationID, RecipientProfileId: principal.ActorID})
}

func requiredPrincipal(ctx context.Context) (Principal, error) {
	principal, ok := PrincipalFromContext(ctx)
	if !ok || principal.ActorID == "" {
		return Principal{}, ErrUnauthenticated
	}
	return principal, nil
}

func toProtoPrincipal(principal Principal) *commonv1.Principal {
	actorType := commonv1.ActorType_ACTOR_TYPE_USER
	if principal.IsGuest {
		actorType = commonv1.ActorType_ACTOR_TYPE_GUEST
	}
	return &commonv1.Principal{
		ActorId:     principal.ActorID,
		ActorType:   actorType,
		TenantId:    principal.TenantID,
		Roles:       principal.Roles,
		Permissions: principal.Permissions,
		IsGuest:     principal.IsGuest,
	}
}

func capPageSize(value int32, fallback int32, max int32) int32 {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
