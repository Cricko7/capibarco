package httpserver

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	"github.com/petmatch/petmatch/internal/app/gateway"
	"github.com/petmatch/petmatch/internal/config"
	kafkaevents "github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDecodeUpdateProfileBodyAcceptsFlatSnakeCaseProfile(t *testing.T) {
	profile, mask, err := decodeUpdateProfileBodyBytes([]byte(`{"display_name":"Alice","bio":"Loves cats","update_mask":["display_name","bio"]}`))
	if err != nil {
		t.Fatalf("decodeUpdateProfileBodyBytes() error = %v", err)
	}
	if profile.GetDisplayName() != "Alice" {
		t.Fatalf("display_name decoded as %q, want Alice", profile.GetDisplayName())
	}
	if profile.GetBio() != "Loves cats" {
		t.Fatalf("bio decoded as %q, want Loves cats", profile.GetBio())
	}
	if len(mask) != 2 || mask[0] != "display_name" || mask[1] != "bio" {
		t.Fatalf("update_mask decoded as %v, want [display_name bio]", mask)
	}
}

func TestDecodeUpdateProfileBodyAcceptsWrappedSnakeCaseProfile(t *testing.T) {
	profile, _, err := decodeUpdateProfileBodyBytes([]byte(`{"profile":{"profile_type":1,"display_name":"Alice"}}`))
	if err != nil {
		t.Fatalf("decodeUpdateProfileBodyBytes() error = %v", err)
	}
	if profile.GetProfileType() != userv1.ProfileType_PROFILE_TYPE_USER {
		t.Fatalf("profile_type decoded as %v, want PROFILE_TYPE_USER", profile.GetProfileType())
	}
	if profile.GetDisplayName() != "Alice" {
		t.Fatalf("display_name decoded as %q, want Alice", profile.GetDisplayName())
	}
}

func TestDecodeUpdateProfileBodyAcceptsStringProfileType(t *testing.T) {
	profile, _, err := decodeUpdateProfileBodyBytes([]byte(`{"profile":{"profile_type":"PROFILE_TYPE_KENNEL","display_name":"Alice"}}`))
	if err != nil {
		t.Fatalf("decodeUpdateProfileBodyBytes() error = %v", err)
	}
	if profile.GetProfileType() != userv1.ProfileType_PROFILE_TYPE_KENNEL {
		t.Fatalf("profile_type decoded as %v, want PROFILE_TYPE_KENNEL", profile.GetProfileType())
	}
}

func TestDecodeUpdateAnimalBodyAcceptsWrappedSnakeCaseAnimal(t *testing.T) {
	animal, mask, err := decodeUpdateAnimalBodyBytes([]byte(`{"animal":{"name":"Mila","sex":"ANIMAL_SEX_FEMALE","size":"ANIMAL_SIZE_MEDIUM","species":"SPECIES_DOG","location":{"city":"Moscow"}},"update_mask":["name","location"]}`))
	if err != nil {
		t.Fatalf("decodeUpdateAnimalBodyBytes() error = %v", err)
	}
	if animal.GetName() != "Mila" {
		t.Fatalf("name decoded as %q, want Mila", animal.GetName())
	}
	if animal.GetLocation().GetCity() != "Moscow" {
		t.Fatalf("location.city decoded as %q, want Moscow", animal.GetLocation().GetCity())
	}
	if len(mask) != 2 || mask[0] != "name" || mask[1] != "location" {
		t.Fatalf("update_mask decoded as %v, want [name location]", mask)
	}
}

func TestDecodeSwipeAnimalBodyAcceptsStringDirection(t *testing.T) {
	input, err := decodeSwipeAnimalBodyBytes([]byte(`{"owner_profile_id":"owner-1","direction":"SWIPE_DIRECTION_RIGHT"}`))
	if err != nil {
		t.Fatalf("decodeSwipeAnimalBodyBytes() error = %v", err)
	}
	if input.Direction != 2 {
		t.Fatalf("direction decoded as %v, want 2", input.Direction)
	}
}

func TestDecodeCreateDonationIntentBodyAcceptsStringTargetType(t *testing.T) {
	input, err := decodeCreateDonationIntentBodyBytes([]byte(`{"target_type":"DONATION_TARGET_TYPE_ANIMAL","target_id":"animal-1","amount":{"currency_code":"RUB","units":500,"nanos":0}}`))
	if err != nil {
		t.Fatalf("decodeCreateDonationIntentBodyBytes() error = %v", err)
	}
	if input.TargetType != 2 {
		t.Fatalf("target_type decoded as %v, want 2", input.TargetType)
	}
	if input.Provider != "mock" {
		t.Fatalf("provider decoded as %q, want mock", input.Provider)
	}
}

func TestUserProfileJSONUsesSnakeCaseContract(t *testing.T) {
	payload := userProfileJSON(&userv1.UserProfile{
		ProfileId:   "profile-1",
		AuthUserId:  "auth-1",
		ProfileType: userv1.ProfileType_PROFILE_TYPE_KENNEL,
		DisplayName: "Alice",
		Bio:         "Loves dogs",
		Address:     &commonv1.Address{City: "Moscow"},
		Visibility:  commonv1.Visibility_VISIBILITY_PUBLIC,
	})

	if got := payload["display_name"]; got != "Alice" {
		t.Fatalf("display_name = %v, want Alice", got)
	}
	if got := payload["profile_type"]; got != "PROFILE_TYPE_KENNEL" {
		t.Fatalf("profile_type = %v, want PROFILE_TYPE_KENNEL", got)
	}
	address, ok := payload["address"].(gin.H)
	if !ok {
		t.Fatalf("address payload type = %T, want gin.H", payload["address"])
	}
	if got := address["city"]; got != "Moscow" {
		t.Fatalf("address.city = %v, want Moscow", got)
	}
}

func TestNotificationJSONUsesSnakeCaseContract(t *testing.T) {
	createdAt := time.Date(2026, 4, 19, 5, 37, 48, 0, time.UTC)
	payload := notificationJSON(&notificationv1.Notification{
		NotificationId:     "notification-1",
		RecipientProfileId: "owner-1",
		Type:               notificationv1.NotificationType_NOTIFICATION_TYPE_MATCH_CREATED,
		Channels: []notificationv1.NotificationChannel{
			notificationv1.NotificationChannel_NOTIFICATION_CHANNEL_IN_APP,
		},
		Title:     "New adoption response",
		Body:      "Open this notification to start a chat.",
		Data:      map[string]string{"match_id": "match-1"},
		Status:    notificationv1.NotificationStatus_NOTIFICATION_STATUS_DELIVERED,
		CreatedAt: timestamppb.New(createdAt),
	})

	if got := payload["notification_id"]; got != "notification-1" {
		t.Fatalf("notification_id = %v, want notification-1", got)
	}
	if got := payload["recipient_profile_id"]; got != "owner-1" {
		t.Fatalf("recipient_profile_id = %v, want owner-1", got)
	}
	if got := payload["type"]; got != "NOTIFICATION_TYPE_MATCH_CREATED" {
		t.Fatalf("type = %v, want NOTIFICATION_TYPE_MATCH_CREATED", got)
	}
	if got := payload["status"]; got != "NOTIFICATION_STATUS_DELIVERED" {
		t.Fatalf("status = %v, want NOTIFICATION_STATUS_DELIVERED", got)
	}
	data, ok := payload["data"].(map[string]string)
	if !ok {
		t.Fatalf("data payload type = %T, want map[string]string", payload["data"])
	}
	if got := data["match_id"]; got != "match-1" {
		t.Fatalf("data.match_id = %v, want match-1", got)
	}
	if got := payload["created_at"]; got != "2026-04-19T05:37:48Z" {
		t.Fatalf("created_at = %v, want RFC3339 timestamp", got)
	}
}

func TestProfileAnimalStatusesForViewerIncludesDraftsForOwnerOnly(t *testing.T) {
	ownerCtx := gateway.WithPrincipal(
		context.Background(),
		gateway.Principal{ActorID: "profile-1"},
	)
	ownerStatuses := profileAnimalStatusesForViewer("profile-1", ownerCtx)
	if len(ownerStatuses) != 2 ||
		ownerStatuses[0] != animalv1.AnimalStatus_ANIMAL_STATUS_DRAFT ||
		ownerStatuses[1] != animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE {
		t.Fatalf("owner statuses = %v, want [DRAFT AVAILABLE]", ownerStatuses)
	}

	viewerCtx := gateway.WithPrincipal(
		context.Background(),
		gateway.Principal{ActorID: "viewer-1"},
	)
	viewerStatuses := profileAnimalStatusesForViewer("profile-1", viewerCtx)
	if len(viewerStatuses) != 1 ||
		viewerStatuses[0] != animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE {
		t.Fatalf("viewer statuses = %v, want [AVAILABLE]", viewerStatuses)
	}
}

func TestImageDimensionsReadsUploadedImageAndRewinds(t *testing.T) {
	var body bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&body, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}

	width, height, err := imageDimensions(bytes.NewReader(body.Bytes()))
	if err != nil {
		t.Fatalf("imageDimensions() error = %v", err)
	}
	if width != 3 || height != 2 {
		t.Fatalf("imageDimensions() = %dx%d, want 3x2", width, height)
	}
}

func TestBeginSSEWritesHeadersBeforeFirstEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	beginSSE(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

func TestBearerTokenAcceptsAuthorizationHeaderAndRawQueryToken(t *testing.T) {
	t.Run("authorization header", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		request := httptest.NewRequest(http.MethodGet, "/ws/chat", nil)
		request.Header.Set("Authorization", "Bearer token-from-header")
		ctx.Request = request

		if got := bearerToken(ctx); got != "token-from-header" {
			t.Fatalf("bearerToken() = %q, want %q", got, "token-from-header")
		}
	})

	t.Run("raw query token", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/ws/chat?access_token=token-from-query", nil)

		if got := bearerToken(ctx); got != "token-from-query" {
			t.Fatalf("bearerToken() = %q, want %q", got, "token-from-query")
		}
	})
}

func TestPublishAnimalRouteIsRegistered(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := New(
		config.Config{
			HTTP: config.HTTPConfig{
				MaxBodyBytes: 1 << 20,
			},
			Rate: config.RateConfig{
				Window: time.Minute,
			},
		},
		nil,
		nil,
		nil,
		nil,
		kafkaevents.NoopPublisher{},
		nil,
		registry,
		metrics.New(registry),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/animals/animal-1/publish", nil)
	recorder := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d to confirm route is registered", recorder.Code, http.StatusUnauthorized)
	}
}

func TestUpdateAnimalRouteIsRegistered(t *testing.T) {
	registry := prometheus.NewRegistry()
	server := New(
		config.Config{
			HTTP: config.HTTPConfig{
				MaxBodyBytes: 1 << 20,
			},
			Rate: config.RateConfig{
				Window: time.Minute,
			},
		},
		nil,
		nil,
		nil,
		nil,
		kafkaevents.NoopPublisher{},
		nil,
		registry,
		metrics.New(registry),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	request := httptest.NewRequest(http.MethodPatch, "/v1/animals/animal-1", bytes.NewBufferString(`{"animal":{"name":"Mila"},"update_mask":["name"]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d to confirm route is registered", recorder.Code, http.StatusUnauthorized)
	}
}
