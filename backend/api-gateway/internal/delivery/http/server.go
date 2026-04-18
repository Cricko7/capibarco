// Package httpserver exposes the mobile REST/WebSocket facade.
package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/petmatch/petmatch/internal/app/gateway"
	"github.com/petmatch/petmatch/internal/config"
	kafkaevents "github.com/petmatch/petmatch/internal/infra/kafka"
	"github.com/petmatch/petmatch/internal/infra/redislimiter"
	"github.com/petmatch/petmatch/internal/infra/storage"
	"github.com/petmatch/petmatch/internal/metrics"
	"github.com/petmatch/petmatch/internal/pkg/problem"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

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

type rateLimiter interface {
	Allow(context.Context, string, int64, time.Duration) (redislimiter.Decision, error)
	Ping(context.Context) error
}

type chatStreamer interface {
	Connect(context.Context) (grpc.BidiStreamingClient[chatv1.ClientChatFrame, chatv1.ServerChatFrame], error)
}

type notificationStreamer interface {
	StreamNotifications(context.Context, *notificationv1.StreamNotificationsRequest) (grpc.ServerStreamingClient[notificationv1.Notification], error)
}

// Server owns the public HTTP server.
type Server struct {
	server        *http.Server
	app           *gateway.Service
	chat          chatStreamer
	notifications notificationStreamer
	limiter       rateLimiter
	publisher     kafkaevents.Publisher
	uploader      storage.Uploader
	cfg           config.Config
	logger        *slog.Logger
	metrics       *metrics.Metrics
	registry      *prometheus.Registry
}

// New creates the API gateway HTTP server.
func New(cfg config.Config, app *gateway.Service, chat chatStreamer, notifications notificationStreamer, limiter rateLimiter, publisher kafkaevents.Publisher, uploader storage.Uploader, registry *prometheus.Registry, metrics *metrics.Metrics, logger *slog.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		app:           app,
		chat:          chat,
		notifications: notifications,
		limiter:       limiter,
		publisher:     publisher,
		uploader:      uploader,
		cfg:           cfg,
		logger:        logger,
		metrics:       metrics,
		registry:      registry,
	}
	router := gin.New()
	router.Use(s.recoverer(), s.requestID(), s.securityHeaders(), s.cors(), s.sizeLimit(), s.metricsMiddleware())

	router.GET("/healthz", s.healthz)
	router.GET("/readyz", s.readyz)
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))
	router.StaticFile("/openapi.yaml", "api/openapi.yaml")

	router.POST("/v1/auth/guest-sessions", s.createGuestSession)
	router.POST("/v1/auth/register", s.register)
	router.POST("/v1/auth/login", s.login)

	protected := router.Group("")
	protected.Use(s.auth(), s.rateLimit())
	protected.GET("/v1/feed", s.getFeed)
	protected.GET("/v1/animals/:animal_id", s.getAnimal)
	protected.POST("/v1/animals", s.createAnimal)
	protected.POST("/v1/animals/:animal_id/photos", s.uploadAnimalPhoto)
	protected.POST("/v1/animals/:animal_id", s.swipeAnimalColon)
	protected.POST("/v1/animals/:animal_id/swipe", s.swipeAnimal)
	protected.GET("/v1/animals/:animal_id/stats", s.getAnimalStats)
	protected.GET("/v1/profiles", s.searchProfiles)
	protected.GET("/v1/profiles/:profile_id", s.getProfile)
	protected.PATCH("/v1/profiles/:profile_id", s.updateProfile)
	protected.GET("/v1/profiles/:profile_id/reviews", s.listReviews)
	protected.POST("/v1/profiles/:profile_id/reviews", s.createReview)
	protected.GET("/v1/profiles/:profile_id/reputation", s.getReputationSummary)
	protected.PATCH("/v1/reviews/:review_id", s.updateReview)
	protected.GET("/v1/chat/conversations", s.listConversations)
	protected.GET("/v1/chat/conversations/:conversation_id/messages", s.listMessages)
	protected.POST("/v1/chat/conversations/:conversation_id/messages", s.sendMessage)
	protected.POST("/v1/billing/donation-intents", s.createDonationIntent)
	protected.POST("/v1/notifications/devices", s.registerDevice)
	protected.DELETE("/v1/notifications/devices/:device_token_id", s.unregisterDevice)
	protected.GET("/v1/notifications", s.listNotifications)
	protected.POST("/v1/notifications/:notification_id/read", s.markNotificationRead)
	protected.GET("/v1/notifications/stream", s.streamNotifications)
	protected.GET("/ws/chat", s.chatWebSocket)
	protected.POST("/v1/notifications/:notification_id", s.markNotificationReadColon)

	s.server = &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http: %w", err)
	}
	return nil
}

func (s *Server) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), s.cfg.Redis.Timeout)
	defer cancel()
	if err := s.limiter.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) createGuestSession(c *gin.Context) {
	var input gateway.CreateGuestSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	out, err := s.app.CreateGuestSession(c.Request.Context(), input)
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (s *Server) register(c *gin.Context) {
	var input gateway.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	input.IP = c.ClientIP()
	input.TenantID = s.cfg.Auth.TenantID
	out, err := s.app.Register(c.Request.Context(), input)
	if err != nil {
		s.publishRejected(c, http.StatusBadRequest, err.Error())
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (s *Server) login(c *gin.Context) {
	var input gateway.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	input.IP = c.ClientIP()
	input.TenantID = s.cfg.Auth.TenantID
	out, err := s.app.Login(c.Request.Context(), input)
	if err != nil {
		s.publishRejected(c, http.StatusUnauthorized, err.Error())
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) getFeed(c *gin.Context) {
	out, err := s.app.GetFeed(c.Request.Context(), gateway.GetFeedInput{
		Surface:   feedv1.FeedSurface(queryInt32(c, "surface")),
		PageSize:  queryInt32(c, "page_size"),
		PageToken: c.Query("page_token"),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) getAnimal(c *gin.Context) {
	out, err := s.app.GetAnimal(c.Request.Context(), c.Param("animal_id"))
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, &animalv1.GetAnimalResponse{Animal: out})
}

func (s *Server) createAnimal(c *gin.Context) {
	var animal animalv1.AnimalProfile
	if err := decodeProtoBody(c, &animal); err != nil {
		problem.Abort(c, err)
		return
	}
	out, err := s.app.CreateAnimal(c.Request.Context(), gateway.CreateAnimalInput{Animal: &animal, IdempotencyKey: idempotencyKey(c)})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusCreated, &animalv1.CreateAnimalResponse{Animal: out})
}

func (s *Server) uploadAnimalPhoto(c *gin.Context) {
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		problem.Abort(c, fmt.Errorf("%w: photo is required", gateway.ErrInvalidInput))
		return
	}
	defer closeFile(file)
	width, height, err := imageDimensions(file)
	if err != nil {
		problem.Abort(c, fmt.Errorf("%w: decode photo dimensions: %v", gateway.ErrInvalidInput, err))
		return
	}
	url, err := s.uploader.Upload(c.Request.Context(), objectName(c.Param("animal_id"), header), file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		problem.Abort(c, err)
		return
	}
	photo := &commonv1.Photo{
		PhotoId:     uuid.NewString(),
		Url:         url,
		Width:       width,
		Height:      height,
		ContentType: header.Header.Get("Content-Type"),
		SortOrder:   int32(formInt(c, "sort_order")),
		CreatedAt:   timestamppb.Now(),
	}
	out, err := s.app.UploadAnimalPhoto(c.Request.Context(), gateway.UploadAnimalPhotoInput{AnimalID: c.Param("animal_id"), Photo: photo, IdempotencyKey: idempotencyKey(c)})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusCreated, &animalv1.AddAnimalPhotoResponse{Animal: out})
}

func (s *Server) swipeAnimal(c *gin.Context) {
	s.swipeAnimalWithID(c, c.Param("animal_id"))
}

func (s *Server) swipeAnimalColon(c *gin.Context) {
	animalID, ok := strings.CutSuffix(c.Param("animal_id"), ":swipe")
	if !ok || animalID == "" {
		c.Status(http.StatusNotFound)
		return
	}
	s.swipeAnimalWithID(c, animalID)
}

func (s *Server) swipeAnimalWithID(c *gin.Context, animalID string) {
	input, err := decodeSwipeAnimalBody(c)
	if err != nil {
		problem.Abort(c, err)
		return
	}
	out, err := s.app.SwipeAnimal(c.Request.Context(), gateway.SwipeAnimalInput{
		AnimalID:       animalID,
		OwnerProfileID: input.OwnerProfileID,
		Direction:      matchingv1.SwipeDirection(input.Direction),
		FeedCardID:     input.FeedCardID,
		FeedSessionID:  input.FeedSessionID,
		IdempotencyKey: idempotencyKey(c),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) getAnimalStats(c *gin.Context) {
	out, err := s.app.GetAnimalStats(c.Request.Context(), gateway.GetAnimalStatsInput{
		AnimalID: c.Param("animal_id"),
		Bucket:   analyticsv1.TimeBucket(queryInt32(c, "bucket")),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func (s *Server) getProfile(c *gin.Context) {
	out, err := s.app.GetProfile(c.Request.Context(), c.Param("profile_id"))
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) searchProfiles(c *gin.Context) {
	var profileTypes []userv1.ProfileType
	for _, raw := range c.QueryArray("profile_type") {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			problem.Abort(c, fmt.Errorf("%w: invalid profile_type", gateway.ErrInvalidInput))
			return
		}
		profileTypes = append(profileTypes, userv1.ProfileType(value))
	}
	var city *string
	if value := c.Query("city"); value != "" {
		city = &value
	}
	var minAverageRating *float64
	if value := c.Query("min_average_rating"); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			problem.Abort(c, fmt.Errorf("%w: invalid min_average_rating", gateway.ErrInvalidInput))
			return
		}
		minAverageRating = &parsed
	}
	var query *string
	if value := c.Query("query"); value != "" {
		query = &value
	}
	out, err := s.app.SearchProfiles(c.Request.Context(), gateway.SearchProfilesInput{
		ProfileTypes:     profileTypes,
		City:             city,
		MinAverageRating: minAverageRating,
		Query:            query,
		IncludeSuspended: c.Query("include_suspended") == "true",
		PageSize:         queryInt32(c, "page_size"),
		PageToken:        c.Query("page_token"),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) updateProfile(c *gin.Context) {
	profile, updateMask, err := decodeUpdateProfileBody(c)
	if err != nil {
		problem.Abort(c, err)
		return
	}
	out, err := s.app.UpdateProfile(c.Request.Context(), gateway.UpdateProfileInput{
		ProfileID:  c.Param("profile_id"),
		Profile:    profile,
		UpdateMask: updateMask,
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) createReview(c *gin.Context) {
	var input struct {
		Rating  int32   `json:"rating"`
		Text    string  `json:"text"`
		MatchID *string `json:"match_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	out, err := s.app.CreateReview(c.Request.Context(), gateway.CreateReviewInput{
		TargetProfileID: c.Param("profile_id"),
		Rating:          input.Rating,
		Text:            input.Text,
		MatchID:         input.MatchID,
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (s *Server) updateReview(c *gin.Context) {
	var input struct {
		Review     *userv1.Review `json:"review"`
		UpdateMask []string       `json:"update_mask"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	out, err := s.app.UpdateReview(c.Request.Context(), gateway.UpdateReviewInput{
		ReviewID:   c.Param("review_id"),
		Review:     input.Review,
		UpdateMask: input.UpdateMask,
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) listReviews(c *gin.Context) {
	out, err := s.app.ListReviews(c.Request.Context(), gateway.ListReviewsInput{
		TargetProfileID: c.Param("profile_id"),
		PageSize:        queryInt32(c, "page_size"),
		PageToken:       c.Query("page_token"),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) getReputationSummary(c *gin.Context) {
	out, err := s.app.GetReputationSummary(c.Request.Context(), c.Param("profile_id"))
	if err != nil {
		problem.Abort(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) listConversations(c *gin.Context) {
	out, err := s.app.ListConversations(c.Request.Context(), gateway.ListConversationsInput{PageSize: queryInt32(c, "page_size"), PageToken: c.Query("page_token")})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func (s *Server) listMessages(c *gin.Context) {
	out, err := s.app.ListMessages(c.Request.Context(), gateway.ListMessagesInput{ConversationID: c.Param("conversation_id"), PageSize: queryInt32(c, "page_size"), PageToken: c.Query("page_token")})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func (s *Server) sendMessage(c *gin.Context) {
	var input struct {
		Type            int32  `json:"type"`
		Text            string `json:"text"`
		ClientMessageID string `json:"client_message_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	out, err := s.app.SendMessage(c.Request.Context(), gateway.SendMessageInput{
		ConversationID:  c.Param("conversation_id"),
		Type:            chatv1.MessageType(input.Type),
		Text:            input.Text,
		ClientMessageID: input.ClientMessageID,
		IdempotencyKey:  idempotencyKey(c),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusCreated, out)
}

func (s *Server) createDonationIntent(c *gin.Context) {
	var input struct {
		TargetType int32                 `json:"target_type"`
		TargetID   string                `json:"target_id"`
		Amount     *commonv1.MoneyAmount `json:"amount"`
		Provider   string                `json:"provider"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	out, err := s.app.CreateDonationIntent(c.Request.Context(), gateway.CreateDonationIntentInput{
		TargetType:     billingv1.DonationTargetType(input.TargetType),
		TargetID:       input.TargetID,
		Amount:         input.Amount,
		Provider:       input.Provider,
		IdempotencyKey: idempotencyKey(c),
	})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusCreated, out)
}

func (s *Server) registerDevice(c *gin.Context) {
	var input gateway.RegisterDeviceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		problem.Abort(c, fmt.Errorf("%w: %v", gateway.ErrInvalidInput, err))
		return
	}
	out, err := s.app.RegisterDevice(c.Request.Context(), input)
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusCreated, out)
}

func (s *Server) unregisterDevice(c *gin.Context) {
	out, err := s.app.UnregisterDevice(c.Request.Context(), c.Param("device_token_id"))
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func (s *Server) listNotifications(c *gin.Context) {
	out, err := s.app.ListNotifications(c.Request.Context(), gateway.ListNotificationsInput{PageSize: queryInt32(c, "page_size"), PageToken: c.Query("page_token")})
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func (s *Server) markNotificationRead(c *gin.Context) {
	out, err := s.app.MarkNotificationRead(c.Request.Context(), c.Param("notification_id"))
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func (s *Server) markNotificationReadColon(c *gin.Context) {
	notificationID, ok := strings.CutSuffix(c.Param("notification_id"), ":read")
	if !ok || notificationID == "" {
		c.Status(http.StatusNotFound)
		return
	}
	out, err := s.app.MarkNotificationRead(c.Request.Context(), notificationID)
	if err != nil {
		problem.Abort(c, err)
		return
	}
	writeProto(c, http.StatusOK, out)
}

func decodeProtoBody(c *gin.Context, msg proto.Message) error {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("%w: read body: %v", gateway.ErrInvalidInput, err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, msg); err != nil {
		return fmt.Errorf("%w: decode json: %v", gateway.ErrInvalidInput, err)
	}
	return nil
}

func decodeUpdateProfileBody(c *gin.Context) (*userv1.UserProfile, []string, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: read body: %v", gateway.ErrInvalidInput, err)
	}
	profile, updateMask, err := decodeUpdateProfileBodyBytes(body)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: decode json: %v", gateway.ErrInvalidInput, err)
	}
	return profile, updateMask, nil
}

func decodeSwipeAnimalBody(c *gin.Context) (swipeAnimalJSON, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return swipeAnimalJSON{}, fmt.Errorf("%w: read body: %v", gateway.ErrInvalidInput, err)
	}
	input, err := decodeSwipeAnimalBodyBytes(body)
	if err != nil {
		return swipeAnimalJSON{}, fmt.Errorf("%w: decode json: %v", gateway.ErrInvalidInput, err)
	}
	return input, nil
}

func decodeSwipeAnimalBodyBytes(body []byte) (swipeAnimalJSON, error) {
	var input swipeAnimalJSON
	if err := json.Unmarshal(body, &input); err != nil {
		return swipeAnimalJSON{}, err
	}
	return input, nil
}

type swipeAnimalJSON struct {
	OwnerProfileID string         `json:"owner_profile_id"`
	Direction      swipeDirection `json:"direction"`
	FeedCardID     *string        `json:"feed_card_id"`
	FeedSessionID  *string        `json:"feed_session_id"`
}

type swipeDirection int32

func (d *swipeDirection) UnmarshalJSON(data []byte) error {
	var number int32
	if err := json.Unmarshal(data, &number); err == nil {
		*d = swipeDirection(number)
		return nil
	}
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	value, ok := matchingv1.SwipeDirection_value[name]
	if !ok {
		return fmt.Errorf("unknown swipe direction %q", name)
	}
	*d = swipeDirection(value)
	return nil
}

func decodeUpdateProfileBodyBytes(body []byte) (*userv1.UserProfile, []string, error) {
	var input updateProfileJSON
	if err := json.Unmarshal(body, &input); err != nil {
		return nil, nil, err
	}
	if input.Profile == nil {
		var profile profileJSON
		if err := json.Unmarshal(body, &profile); err != nil {
			return nil, nil, err
		}
		input.Profile = &profile
	}
	return input.Profile.toProto(), input.UpdateMask, nil
}

type updateProfileJSON struct {
	Profile    *profileJSON `json:"profile"`
	UpdateMask []string     `json:"update_mask"`
}

type profileJSON struct {
	ProfileID   string              `json:"profile_id"`
	AuthUserID  string              `json:"auth_user_id"`
	ProfileType userv1.ProfileType  `json:"profile_type"`
	DisplayName string              `json:"display_name"`
	Bio         string              `json:"bio"`
	AvatarURL   string              `json:"avatar_url"`
	Address     *commonv1.Address   `json:"address"`
	Visibility  commonv1.Visibility `json:"visibility"`
	UpdateMask  []string            `json:"update_mask"`
}

func (p profileJSON) toProto() *userv1.UserProfile {
	return &userv1.UserProfile{
		ProfileId:   p.ProfileID,
		AuthUserId:  p.AuthUserID,
		ProfileType: p.ProfileType,
		DisplayName: p.DisplayName,
		Bio:         p.Bio,
		AvatarUrl:   p.AvatarURL,
		Address:     p.Address,
		Visibility:  p.Visibility,
	}
}

func writeProto(c *gin.Context, status int, msg proto.Message) {
	body, err := (protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}).Marshal(msg)
	if err != nil {
		problem.Abort(c, fmt.Errorf("marshal proto response: %w", err))
		return
	}
	c.Data(status, "application/json", body)
}

func queryInt32(c *gin.Context, key string) int32 {
	value, _ := strconv.ParseInt(c.Query(key), 10, 32)
	return int32(value)
}

func formInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.PostForm(key))
	return value
}

func idempotencyKey(c *gin.Context) string {
	if value := c.GetHeader("Idempotency-Key"); value != "" {
		return value
	}
	return c.GetHeader("X-Idempotency-Key")
}

func objectName(animalID string, header *multipart.FileHeader) string {
	name := strings.ReplaceAll(filepath.Base(header.Filename), " ", "_")
	return fmt.Sprintf("animals/%s/%s-%s", animalID, uuid.NewString(), name)
}

func imageDimensions(file io.ReadSeeker) (int32, int32, error) {
	cfg, _, err := image.DecodeConfig(file)
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil && err == nil {
		err = seekErr
	}
	if err != nil {
		return 0, 0, err
	}
	return int32(cfg.Width), int32(cfg.Height), nil
}

func closeFile(file multipart.File) {
	if closer, ok := file.(io.Closer); ok {
		_ = closer.Close()
	}
}
