package gateway

import (
	"time"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
)

// Defaults contains gateway orchestration defaults.
type Defaults struct {
	TenantID         string
	FeedPrefetchSize int32
	MaxPageSize      int32
}

// Principal is an actor identity returned by auth-service or guest-session validation.
type Principal struct {
	ActorID     string
	TenantID    string
	Email       string
	Roles       []string
	Permissions []string
	TokenID     string
	IsGuest     bool
}

// LoginInput is the mobile login request.
type RegisterInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Locale   string `json:"locale"`
	IP       string `json:"-"`
	TenantID string `json:"-"`
}

// LoginInput is the mobile login request.
type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Locale   string `json:"locale"`
	IP       string `json:"-"`
	TenantID string `json:"-"`
}

// AuthResponse mirrors auth-service response shape.
type AuthResponse struct {
	User         *AuthUser `json:"user,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    string    `json:"expires_at"`
}

// AuthUser is returned by auth-service.
type AuthUser struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateGuestSessionInput creates an anonymous browsing session.
type CreateGuestSessionInput struct {
	DeviceID string `json:"device_id" validate:"required"`
	Locale   string `json:"locale"`
}

// CreateGuestSessionOutput is returned to mobile clients.
type CreateGuestSessionOutput struct {
	Token         string    `json:"guest_session_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	AllowedScopes []string  `json:"allowed_scopes"`
}

// GetFeedInput contains mobile feed query parameters.
type GetFeedInput struct {
	Surface   feedv1.FeedSurface
	Filter    *feedv1.FeedFilter
	PageSize  int32
	PageToken string
}

// GetFeedOutput is the mobile feed response.
type GetFeedOutput struct {
	Cards         []*feedv1.FeedCard `json:"cards"`
	NextPageToken string             `json:"next_page_token"`
	FeedSessionID string             `json:"feed_session_id"`
}

// SwipeAnimalInput contains a mobile swipe command.
type SwipeAnimalInput struct {
	AnimalID       string
	OwnerProfileID string
	Direction      matchingv1.SwipeDirection
	FeedCardID     *string
	FeedSessionID  *string
	IdempotencyKey string
}

// SwipeAnimalOutput is returned after recording a swipe.
type SwipeAnimalOutput struct {
	Swipe          *matchingv1.Swipe `json:"swipe,omitempty"`
	Match          *matchingv1.Match `json:"match,omitempty"`
	ConversationID *string           `json:"conversation_id,omitempty"`
}

// CreateAnimalInput wraps animal creation.
type CreateAnimalInput struct {
	Animal         *animalv1.AnimalProfile
	IdempotencyKey string
}

// UploadAnimalPhotoInput describes a multipart animal photo upload.
type UploadAnimalPhotoInput struct {
	AnimalID       string
	Photo          *commonv1.Photo
	IdempotencyKey string
}

// ListConversationsInput contains pagination for chat conversations.
type ListConversationsInput struct {
	PageSize  int32
	PageToken string
}

// ListMessagesInput contains pagination for chat messages.
type ListMessagesInput struct {
	ConversationID string
	PageSize       int32
	PageToken      string
}

// SendMessageInput contains a mobile chat send command.
type SendMessageInput struct {
	ConversationID  string
	Type            chatv1.MessageType
	Text            string
	ClientMessageID string
	IdempotencyKey  string
}

// CreateDonationIntentInput contains a donation checkout command.
type CreateDonationIntentInput struct {
	TargetType     billingv1.DonationTargetType
	TargetID       string
	Amount         *commonv1.MoneyAmount
	Provider       string
	IdempotencyKey string
}

// GetAnimalStatsInput contains animal analytics filters.
type GetAnimalStatsInput struct {
	AnimalID  string
	TimeRange *commonv1.TimeRange
	Bucket    analyticsv1.TimeBucket
}

// RegisterDeviceInput registers a mobile push token.
type RegisterDeviceInput struct {
	Token    string `json:"token" validate:"required"`
	Platform string `json:"platform" validate:"required"`
	Locale   string `json:"locale"`
}

// ListNotificationsInput contains notification inbox pagination.
type ListNotificationsInput struct {
	Statuses  []notificationv1.NotificationStatus
	PageSize  int32
	PageToken string
}
