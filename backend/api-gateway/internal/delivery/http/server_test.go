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
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	"github.com/petmatch/petmatch/internal/app/gateway"
	"github.com/petmatch/petmatch/internal/config"
	kafkaevents "github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
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
