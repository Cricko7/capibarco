package gateway

import (
	"context"
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

// Clock gives the application deterministic time in tests.
type Clock interface {
	Now() time.Time
}

// AuthClient hides auth-service transport details.
type AuthClient interface {
	Register(context.Context, RegisterInput) (*AuthResponse, error)
	Login(context.Context, LoginInput) (*AuthResponse, error)
	Validate(context.Context, string) (Principal, error)
	Authorize(context.Context, string, string) (Principal, bool, error)
}

// FeedClient exposes feed-service calls used by the gateway.
type FeedClient interface {
	GetFeed(context.Context, *feedv1.GetFeedRequest) (*feedv1.GetFeedResponse, error)
}

// AnimalClient exposes animal-service calls used by the gateway.
type AnimalClient interface {
	GetAnimal(context.Context, string) (*animalv1.AnimalProfile, error)
	CreateAnimal(context.Context, *animalv1.CreateAnimalRequest) (*animalv1.AnimalProfile, error)
	AddPhoto(context.Context, string, *commonv1.Photo, string) (*animalv1.AnimalProfile, error)
}

// MatchingClient exposes matching-service calls used by the gateway.
type MatchingClient interface {
	RecordSwipe(context.Context, *matchingv1.RecordSwipeRequest) (*matchingv1.RecordSwipeResponse, error)
}

// ChatClient exposes chat-service unary calls used by REST handlers.
type ChatClient interface {
	ListConversations(context.Context, *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error)
	ListMessages(context.Context, *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error)
	SendMessage(context.Context, *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error)
}

// BillingClient exposes billing-service calls used by the gateway.
type BillingClient interface {
	CreateDonationIntent(context.Context, *billingv1.CreateDonationIntentRequest) (*billingv1.CreateDonationIntentResponse, error)
}

// AnalyticsClient exposes analytics-service calls used by the gateway.
type AnalyticsClient interface {
	GetAnimalStats(context.Context, *analyticsv1.GetAnimalStatsRequest) (*analyticsv1.GetAnimalStatsResponse, error)
}

// NotificationClient exposes notification-service calls used by the gateway.
type NotificationClient interface {
	RegisterDevice(context.Context, *notificationv1.RegisterDeviceRequest) (*notificationv1.RegisterDeviceResponse, error)
	UnregisterDevice(context.Context, *notificationv1.UnregisterDeviceRequest) (*notificationv1.UnregisterDeviceResponse, error)
	ListNotifications(context.Context, *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error)
	MarkNotificationRead(context.Context, *notificationv1.MarkNotificationReadRequest) (*notificationv1.MarkNotificationReadResponse, error)
}
