package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/petmatch/petmatch/internal/pkg/resilience"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	"google.golang.org/grpc"
)

// AnimalClient wraps animal-service.
type AnimalClient struct {
	client  animalv1.AnimalServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewAnimalClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *AnimalClient {
	return &AnimalClient{client: animalv1.NewAnimalServiceClient(conn), timeout: timeout, res: res}
}

func (c *AnimalClient) GetAnimal(ctx context.Context, animalID string) (*animalv1.AnimalProfile, error) {
	out, err := resilience.Do(ctx, c.res, func(ctx context.Context) (*animalv1.GetAnimalResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.GetAnimal(ctx, &animalv1.GetAnimalRequest{AnimalId: animalID})
	})
	if err != nil {
		return nil, err
	}
	return out.Animal, nil
}

func (c *AnimalClient) CreateAnimal(ctx context.Context, req *animalv1.CreateAnimalRequest) (*animalv1.AnimalProfile, error) {
	out, err := resilience.Do(ctx, c.res, func(ctx context.Context) (*animalv1.CreateAnimalResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.CreateAnimal(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return out.Animal, nil
}

func (c *AnimalClient) AddPhoto(ctx context.Context, animalID string, photo *commonv1.Photo, idempotencyKey string) (*animalv1.AnimalProfile, error) {
	out, err := resilience.Do(ctx, c.res, func(ctx context.Context) (*animalv1.AddAnimalPhotoResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.AddAnimalPhoto(ctx, &animalv1.AddAnimalPhotoRequest{AnimalId: animalID, Photo: photo, IdempotencyKey: idempotencyKey})
	})
	if err != nil {
		return nil, err
	}
	return out.Animal, nil
}

// FeedClient wraps feed-service.
type FeedClient struct {
	client  feedv1.FeedServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewFeedClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *FeedClient {
	return &FeedClient{client: feedv1.NewFeedServiceClient(conn), timeout: timeout, res: res}
}

func (c *FeedClient) GetFeed(ctx context.Context, req *feedv1.GetFeedRequest) (*feedv1.GetFeedResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*feedv1.GetFeedResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.GetFeed(ctx, req)
	})
}

// MatchingClient wraps matching-service.
type MatchingClient struct {
	client  matchingv1.MatchingServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewMatchingClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *MatchingClient {
	return &MatchingClient{client: matchingv1.NewMatchingServiceClient(conn), timeout: timeout, res: res}
}

func (c *MatchingClient) RecordSwipe(ctx context.Context, req *matchingv1.RecordSwipeRequest) (*matchingv1.RecordSwipeResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*matchingv1.RecordSwipeResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.RecordSwipe(ctx, req)
	})
}

// ChatClient wraps chat-service.
type ChatClient struct {
	client  chatv1.ChatServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewChatClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *ChatClient {
	return &ChatClient{client: chatv1.NewChatServiceClient(conn), timeout: timeout, res: res}
}

func (c *ChatClient) ListConversations(ctx context.Context, req *chatv1.ListConversationsRequest) (*chatv1.ListConversationsResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*chatv1.ListConversationsResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.ListConversations(ctx, req)
	})
}

func (c *ChatClient) ListMessages(ctx context.Context, req *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*chatv1.ListMessagesResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.ListMessages(ctx, req)
	})
}

func (c *ChatClient) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.SendMessageResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*chatv1.SendMessageResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.SendMessage(ctx, req)
	})
}

// Connect opens a chat bidirectional stream for WebSocket bridging.
func (c *ChatClient) Connect(ctx context.Context) (grpc.BidiStreamingClient[chatv1.ClientChatFrame, chatv1.ServerChatFrame], error) {
	stream, err := c.client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect chat stream: %w", err)
	}
	return stream, nil
}

// BillingClient wraps billing-service.
type BillingClient struct {
	client  billingv1.BillingServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewBillingClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *BillingClient {
	return &BillingClient{client: billingv1.NewBillingServiceClient(conn), timeout: timeout, res: res}
}

func (c *BillingClient) CreateDonationIntent(ctx context.Context, req *billingv1.CreateDonationIntentRequest) (*billingv1.CreateDonationIntentResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*billingv1.CreateDonationIntentResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.CreateDonationIntent(ctx, req)
	})
}

// AnalyticsClient wraps analytics-service.
type AnalyticsClient struct {
	client  analyticsv1.AnalyticsServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewAnalyticsClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *AnalyticsClient {
	return &AnalyticsClient{client: analyticsv1.NewAnalyticsServiceClient(conn), timeout: timeout, res: res}
}

func (c *AnalyticsClient) GetAnimalStats(ctx context.Context, req *analyticsv1.GetAnimalStatsRequest) (*analyticsv1.GetAnimalStatsResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*analyticsv1.GetAnimalStatsResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.GetAnimalStats(ctx, req)
	})
}

// UserClient wraps user-service.
type UserClient struct {
	conn    *grpc.ClientConn
	timeout time.Duration
	res     *resilience.Client
}

func NewUserClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *UserClient {
	return &UserClient{conn: conn, timeout: timeout, res: res}
}

func (c *UserClient) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.GetProfileResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/GetProfile", req, func() *userv1.GetProfileResponse { return &userv1.GetProfileResponse{} })
}

func (c *UserClient) SearchProfiles(ctx context.Context, req *userv1.SearchProfilesRequest) (*userv1.SearchProfilesResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/SearchProfiles", req, func() *userv1.SearchProfilesResponse { return &userv1.SearchProfilesResponse{} })
}

func (c *UserClient) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/UpdateProfile", req, func() *userv1.UpdateProfileResponse { return &userv1.UpdateProfileResponse{} })
}

func (c *UserClient) CreateReview(ctx context.Context, req *userv1.CreateReviewRequest) (*userv1.CreateReviewResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/CreateReview", req, func() *userv1.CreateReviewResponse { return &userv1.CreateReviewResponse{} })
}

func (c *UserClient) UpdateReview(ctx context.Context, req *userv1.UpdateReviewRequest) (*userv1.UpdateReviewResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/UpdateReview", req, func() *userv1.UpdateReviewResponse { return &userv1.UpdateReviewResponse{} })
}

func (c *UserClient) ListReviews(ctx context.Context, req *userv1.ListReviewsRequest) (*userv1.ListReviewsResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/ListReviews", req, func() *userv1.ListReviewsResponse { return &userv1.ListReviewsResponse{} })
}

func (c *UserClient) GetReputationSummary(ctx context.Context, req *userv1.GetReputationSummaryRequest) (*userv1.GetReputationSummaryResponse, error) {
	return invokeUser(ctx, c, "/petmatch.user.v1.UserService/GetReputationSummary", req, func() *userv1.GetReputationSummaryResponse { return &userv1.GetReputationSummaryResponse{} })
}

func invokeUser[Req any, Resp any](ctx context.Context, c *UserClient, method string, req Req, newResp func() Resp) (Resp, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (Resp, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		resp := newResp()
		if err := c.conn.Invoke(ctx, method, req, resp, grpc.ForceCodec(userCodec{})); err != nil {
			var zero Resp
			return zero, err
		}
		return resp, nil
	})
}

type userCodec struct{}

func (userCodec) Name() string { return "json" }

func (userCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

func (userCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// NotificationClient wraps future notification-service.
type NotificationClient struct {
	client  notificationv1.NotificationServiceClient
	timeout time.Duration
	res     *resilience.Client
}

func NewNotificationClient(conn *grpc.ClientConn, timeout time.Duration, res *resilience.Client) *NotificationClient {
	return &NotificationClient{client: notificationv1.NewNotificationServiceClient(conn), timeout: timeout, res: res}
}

func (c *NotificationClient) RegisterDevice(ctx context.Context, req *notificationv1.RegisterDeviceRequest) (*notificationv1.RegisterDeviceResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*notificationv1.RegisterDeviceResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.RegisterDevice(ctx, req)
	})
}

func (c *NotificationClient) UnregisterDevice(ctx context.Context, req *notificationv1.UnregisterDeviceRequest) (*notificationv1.UnregisterDeviceResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*notificationv1.UnregisterDeviceResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.UnregisterDevice(ctx, req)
	})
}

func (c *NotificationClient) ListNotifications(ctx context.Context, req *notificationv1.ListNotificationsRequest) (*notificationv1.ListNotificationsResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*notificationv1.ListNotificationsResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.ListNotifications(ctx, req)
	})
}

func (c *NotificationClient) MarkNotificationRead(ctx context.Context, req *notificationv1.MarkNotificationReadRequest) (*notificationv1.MarkNotificationReadResponse, error) {
	return resilience.Do(ctx, c.res, func(ctx context.Context) (*notificationv1.MarkNotificationReadResponse, error) {
		ctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		return c.client.MarkNotificationRead(ctx, req)
	})
}

// StreamNotifications opens a server stream for SSE bridging.
func (c *NotificationClient) StreamNotifications(ctx context.Context, req *notificationv1.StreamNotificationsRequest) (grpc.ServerStreamingClient[notificationv1.Notification], error) {
	stream, err := c.client.StreamNotifications(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("stream notifications: %w", err)
	}
	return stream, nil
}
