package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	notificationv1 "github.com/petmatch/petmatch/gen/go/petmatch/notification/v1"
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	domain "github.com/petmatch/petmatch/internal/domain/notification"
	"github.com/petmatch/petmatch/internal/pkg/requestid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	TopicRequested = "notification.requested"
	TopicDelivered = "notification.delivered"
	TopicFailed    = "notification.failed"
	TopicRead      = "notification.read"
)

type Service struct {
	repo        Repository
	publisher   EventPublisher
	producer    string
	topicPrefix string
	now         func() time.Time
	logger      *slog.Logger
	streams     *streamHub
}

func NewService(repo Repository, publisher EventPublisher, producer, topicPrefix string, now func() time.Time, logger *slog.Logger) *Service {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		repo:        repo,
		publisher:   publisher,
		producer:    producer,
		topicPrefix: topicPrefix,
		now:         now,
		logger:      logger,
		streams:     newStreamHub(),
	}
}

func (s *Service) RegisterDevice(ctx context.Context, profileID, token, platform, locale string) (domain.DeviceToken, error) {
	if strings.TrimSpace(profileID) == "" || strings.TrimSpace(token) == "" || strings.TrimSpace(platform) == "" {
		return domain.DeviceToken{}, domain.ErrInvalidArgument
	}
	now := s.now().UTC()
	return s.repo.RegisterDevice(ctx, domain.DeviceToken{
		ID:        uuid.NewString(),
		ProfileID: profileID,
		Token:     token,
		Platform:  platform,
		Locale:    locale,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *Service) UnregisterDevice(ctx context.Context, deviceTokenID string) (bool, error) {
	if strings.TrimSpace(deviceTokenID) == "" {
		return false, domain.ErrInvalidArgument
	}
	return s.repo.UnregisterDevice(ctx, deviceTokenID)
}

func (s *Service) CreateNotification(ctx context.Context, n domain.Notification) (domain.Notification, error) {
	if err := validateNotification(n); err != nil {
		return domain.Notification{}, err
	}

	if strings.TrimSpace(n.IdempotencyKey) != "" {
		existing, err := s.repo.FindNotificationByIdempotencyKey(ctx, n.RecipientProfileID, n.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.Notification{}, err
		}
	}

	pref, err := s.repo.GetPreference(ctx, n.RecipientProfileID)
	if err != nil {
		return domain.Notification{}, err
	}

	now := s.now().UTC()
	n.ID = uuid.NewString()
	n.CreatedAt = now
	n.Channels = applyPreference(pref, n.Channels, now)
	if len(n.Channels) == 0 {
		n.Status = domain.StatusFailed
	} else {
		n.Status = domain.StatusPending
	}

	stored, err := s.repo.CreateNotification(ctx, n)
	if err != nil {
		return domain.Notification{}, err
	}

	if err := s.publishRequested(ctx, stored); err != nil {
		return domain.Notification{}, err
	}

	if len(stored.Channels) == 0 {
		if err := s.publishFailed(ctx, stored, domain.ChannelUnspecified, "no enabled channels after preferences"); err != nil {
			return domain.Notification{}, err
		}
		s.streams.broadcast(stored.RecipientProfileID, stored)
		return stored, nil
	}

	if err := s.repo.UpdateNotificationStatus(ctx, stored.ID, domain.StatusDelivered); err != nil {
		return domain.Notification{}, err
	}
	stored.Status = domain.StatusDelivered
	if err := s.publishDelivered(ctx, stored); err != nil {
		return domain.Notification{}, err
	}
	s.streams.broadcast(stored.RecipientProfileID, stored)
	return stored, nil
}

func (s *Service) ListNotifications(ctx context.Context, recipientProfileID string, statuses []domain.Status, page domain.PageRequest) ([]domain.Notification, string, error) {
	if strings.TrimSpace(recipientProfileID) == "" {
		return nil, "", domain.ErrInvalidArgument
	}
	return s.repo.ListNotifications(ctx, recipientProfileID, statuses, page)
}

func (s *Service) MarkNotificationRead(ctx context.Context, notificationID, recipientProfileID string) (domain.Notification, error) {
	if strings.TrimSpace(notificationID) == "" || strings.TrimSpace(recipientProfileID) == "" {
		return domain.Notification{}, domain.ErrInvalidArgument
	}
	readAt := s.now().UTC()
	n, err := s.repo.MarkNotificationRead(ctx, notificationID, recipientProfileID, readAt)
	if err != nil {
		return domain.Notification{}, err
	}
	if err := s.publishRead(ctx, n, readAt); err != nil {
		return domain.Notification{}, err
	}
	s.streams.broadcast(n.RecipientProfileID, n)
	return n, nil
}

func (s *Service) Subscribe(recipientProfileID string) (<-chan domain.Notification, func(), error) {
	if strings.TrimSpace(recipientProfileID) == "" {
		return nil, nil, domain.ErrInvalidArgument
	}
	return s.streams.subscribe(recipientProfileID)
}

func (s *Service) HandleMatchCreated(ctx context.Context, event *matchingv1.MatchCreatedEvent) error {
	match := event.GetMatch()
	if match == nil || strings.TrimSpace(match.GetOwnerProfileId()) == "" {
		return domain.ErrInvalidArgument
	}
	_, err := s.CreateNotification(ctx, domain.Notification{
		RecipientProfileID: match.GetOwnerProfileId(),
		Type:               domain.TypeMatchCreated,
		Channels:           []domain.Channel{domain.ChannelPush, domain.ChannelInApp},
		Title:              "New adoption response",
		Body:               "A user responded to one of your animals. Open this notification to start a chat.",
		Data: map[string]string{
			"match_id":           match.GetMatchId(),
			"animal_id":          match.GetAnimalId(),
			"adopter_profile_id": match.GetAdopterProfileId(),
			"conversation_id":    match.GetConversationId(),
		},
		IdempotencyKey: envelopeKey(event.GetEnvelope()),
	})
	return err
}

func (s *Service) HandleMessageSent(ctx context.Context, event *chatv1.MessageSentEvent) error {
	message := event.GetMessage()
	if message == nil {
		return domain.ErrInvalidArgument
	}
	for _, recipient := range recipientsFromMessage(message) {
		if recipient == "" || recipient == message.GetSenderProfileId() {
			continue
		}
		if _, err := s.CreateNotification(ctx, domain.Notification{
			RecipientProfileID: recipient,
			Type:               domain.TypeChatMessage,
			Channels:           []domain.Channel{domain.ChannelPush, domain.ChannelInApp},
			Title:              "New chat message",
			Body:               summarizeText(message.GetText(), "You received a new message."),
			Data: map[string]string{
				"conversation_id":   message.GetConversationId(),
				"message_id":        message.GetMessageId(),
				"sender_profile_id": message.GetSenderProfileId(),
			},
			IdempotencyKey: envelopeKey(event.GetEnvelope()) + ":" + recipient,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) HandleDonationSucceeded(ctx context.Context, event *billingv1.DonationSucceededEvent) error {
	donation := event.GetDonation()
	if donation == nil {
		return domain.ErrInvalidArgument
	}
	recipients := []string{}
	if donor := strings.TrimSpace(donation.GetPayerProfileId()); donor != "" {
		recipients = append(recipients, donor)
	}
	if owner := strings.TrimSpace(event.GetEnvelope().GetPartitionKey()); owner != "" && owner != donation.GetPayerProfileId() {
		recipients = append(recipients, owner)
	}
	for _, recipient := range recipients {
		if _, err := s.CreateNotification(ctx, domain.Notification{
			RecipientProfileID: recipient,
			Type:               domain.TypeDonationSucceeded,
			Channels:           []domain.Channel{domain.ChannelPush, domain.ChannelInApp, domain.ChannelEmail},
			Title:              "Donation processed",
			Body:               "A donation payment has succeeded.",
			Data: map[string]string{
				"donation_id":      donation.GetDonationId(),
				"target_id":        donation.GetTargetId(),
				"payer_profile_id": donation.GetPayerProfileId(),
			},
			IdempotencyKey: envelopeKey(event.GetEnvelope()) + ":" + recipient,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) HandleBoostActivated(ctx context.Context, event *billingv1.BoostActivatedEvent) error {
	boost := event.GetBoost()
	if boost == nil || strings.TrimSpace(boost.GetOwnerProfileId()) == "" {
		return domain.ErrInvalidArgument
	}
	_, err := s.CreateNotification(ctx, domain.Notification{
		RecipientProfileID: boost.GetOwnerProfileId(),
		Type:               domain.TypeBoostActivated,
		Channels:           []domain.Channel{domain.ChannelPush, domain.ChannelInApp},
		Title:              "Boost is active",
		Body:               "Your boost is now active.",
		Data: map[string]string{
			"boost_id":  boost.GetBoostId(),
			"animal_id": boost.GetAnimalId(),
		},
		IdempotencyKey: envelopeKey(event.GetEnvelope()),
	})
	return err
}

func (s *Service) HandleReviewCreated(ctx context.Context, event *userv1.ReviewCreatedEvent) error {
	review := event.GetReview()
	if review == nil || strings.TrimSpace(review.GetTargetProfileId()) == "" {
		return domain.ErrInvalidArgument
	}
	_, err := s.CreateNotification(ctx, domain.Notification{
		RecipientProfileID: review.GetTargetProfileId(),
		Type:               domain.TypeReviewCreated,
		Channels:           []domain.Channel{domain.ChannelPush, domain.ChannelInApp},
		Title:              "New review received",
		Body:               "Someone left a new review on your profile.",
		Data: map[string]string{
			"review_id":         review.GetReviewId(),
			"author_profile_id": review.GetAuthorProfileId(),
			"match_id":          review.GetMatchId(),
		},
		IdempotencyKey: envelopeKey(event.GetEnvelope()),
	})
	return err
}

func (s *Service) publishRequested(ctx context.Context, n domain.Notification) error {
	payload, err := proto.Marshal(&notificationv1.NotificationRequestedEvent{
		Envelope:     s.envelope(ctx, TopicRequested, n.ID, n.IdempotencyKey, n.RecipientProfileID),
		Notification: toProtoNotification(n),
	})
	if err != nil {
		return fmt.Errorf("marshal requested event: %w", err)
	}
	return s.publisher.Publish(ctx, topicName(s.topicPrefix, TopicRequested), n.RecipientProfileID, payload)
}

func (s *Service) publishDelivered(ctx context.Context, n domain.Notification) error {
	payload, err := proto.Marshal(&notificationv1.NotificationDeliveredEvent{
		Envelope:       s.envelope(ctx, TopicDelivered, n.ID, n.IdempotencyKey, n.ID),
		NotificationId: n.ID,
		Channels:       toProtoChannels(n.Channels),
	})
	if err != nil {
		return fmt.Errorf("marshal delivered event: %w", err)
	}
	return s.publisher.Publish(ctx, topicName(s.topicPrefix, TopicDelivered), n.ID, payload)
}

func (s *Service) publishFailed(ctx context.Context, n domain.Notification, channel domain.Channel, reason string) error {
	payload, err := proto.Marshal(&notificationv1.NotificationFailedEvent{
		Envelope:       s.envelope(ctx, TopicFailed, n.ID, n.IdempotencyKey, n.ID),
		NotificationId: n.ID,
		Channel:        notificationv1.NotificationChannel(channel),
		Reason:         reason,
	})
	if err != nil {
		return fmt.Errorf("marshal failed event: %w", err)
	}
	return s.publisher.Publish(ctx, topicName(s.topicPrefix, TopicFailed), n.ID, payload)
}

func (s *Service) publishRead(ctx context.Context, n domain.Notification, readAt time.Time) error {
	payload, err := json.Marshal(map[string]any{
		"envelope": map[string]any{
			"event_id":        uuid.NewString(),
			"event_type":      TopicRead,
			"schema_version":  "v1",
			"producer":        s.producer,
			"occurred_at":     readAt.Format(time.RFC3339Nano),
			"trace_id":        requestid.From(ctx),
			"correlation_id":  n.ID,
			"idempotency_key": n.IdempotencyKey,
			"partition_key":   n.RecipientProfileID,
		},
		"notification_id":      n.ID,
		"recipient_profile_id": n.RecipientProfileID,
		"read_at":              readAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("marshal read event: %w", err)
	}
	return s.publisher.Publish(ctx, topicName(s.topicPrefix, TopicRead), n.RecipientProfileID, payload)
}

func (s *Service) envelope(ctx context.Context, eventType, correlationID, idempotencyKey, partitionKey string) *commonv1.EventEnvelope {
	return &commonv1.EventEnvelope{
		EventId:        uuid.NewString(),
		EventType:      eventType,
		SchemaVersion:  "v1",
		Producer:       s.producer,
		OccurredAt:     timestamppb.New(s.now().UTC()),
		TraceId:        requestid.From(ctx),
		CorrelationId:  correlationID,
		IdempotencyKey: idempotencyKey,
		PartitionKey:   partitionKey,
	}
}

func validateNotification(n domain.Notification) error {
	if strings.TrimSpace(n.RecipientProfileID) == "" || strings.TrimSpace(n.Title) == "" || strings.TrimSpace(n.Body) == "" || n.Type == domain.TypeUnspecified {
		return domain.ErrInvalidArgument
	}
	for _, channel := range n.Channels {
		if channel == domain.ChannelUnspecified {
			return domain.ErrInvalidArgument
		}
	}
	return nil
}

func applyPreference(pref domain.Preference, channels []domain.Channel, now time.Time) []domain.Channel {
	if pref.Muted {
		return nil
	}
	out := make([]domain.Channel, 0, len(channels))
	quiet := inQuietHours(pref, now)
	for _, channel := range channels {
		switch channel {
		case domain.ChannelPush:
			if pref.PushEnabled && !quiet {
				out = append(out, channel)
			}
		case domain.ChannelEmail:
			if pref.EmailEnabled && !quiet {
				out = append(out, channel)
			}
		case domain.ChannelInApp:
			if pref.InAppEnabled {
				out = append(out, channel)
			}
		}
	}
	return dedupeChannels(out)
}

func inQuietHours(pref domain.Preference, now time.Time) bool {
	if !pref.QuietHoursEnabled {
		return false
	}
	start, errStart := time.Parse("15:04", pref.QuietHoursStart)
	end, errEnd := time.Parse("15:04", pref.QuietHoursEnd)
	if errStart != nil || errEnd != nil {
		return false
	}
	current := now.Hour()*60 + now.Minute()
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()
	if startMinutes == endMinutes {
		return false
	}
	if startMinutes < endMinutes {
		return current >= startMinutes && current < endMinutes
	}
	return current >= startMinutes || current < endMinutes
}

func dedupeChannels(values []domain.Channel) []domain.Channel {
	seen := make(map[domain.Channel]struct{}, len(values))
	out := make([]domain.Channel, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func recipientsFromMessage(message *chatv1.Message) []string {
	if message == nil {
		return nil
	}
	meta := message.GetMetadata()
	if recipient := strings.TrimSpace(meta["recipient_profile_id"]); recipient != "" {
		return []string{recipient}
	}
	if recipients := strings.TrimSpace(meta["recipient_profile_ids"]); recipients != "" {
		parts := strings.Split(recipients, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	return nil
}

func summarizeText(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	runes := []rune(text)
	if len(runes) <= 120 {
		return text
	}
	return string(runes[:117]) + "..."
}

func envelopeKey(envelope *commonv1.EventEnvelope) string {
	if envelope == nil {
		return uuid.NewString()
	}
	if key := strings.TrimSpace(envelope.GetIdempotencyKey()); key != "" {
		return key
	}
	if eventID := strings.TrimSpace(envelope.GetEventId()); eventID != "" {
		return eventID
	}
	return uuid.NewString()
}

func topicName(prefix, topic string) string {
	if strings.TrimSpace(prefix) == "" || prefix == "notification" {
		return topic
	}
	return prefix + "." + strings.TrimPrefix(topic, "notification.")
}

func toProtoNotification(n domain.Notification) *notificationv1.Notification {
	out := &notificationv1.Notification{
		NotificationId:     n.ID,
		RecipientProfileId: n.RecipientProfileID,
		Type:               notificationv1.NotificationType(n.Type),
		Title:              n.Title,
		Body:               n.Body,
		Data:               n.Data,
		Status:             notificationv1.NotificationStatus(n.Status),
		CreatedAt:          timestamppb.New(n.CreatedAt),
	}
	for _, channel := range n.Channels {
		out.Channels = append(out.Channels, notificationv1.NotificationChannel(channel))
	}
	if n.ReadAt != nil {
		out.ReadAt = timestamppb.New(*n.ReadAt)
	}
	return out
}

func toProtoChannels(channels []domain.Channel) []notificationv1.NotificationChannel {
	out := make([]notificationv1.NotificationChannel, 0, len(channels))
	for _, channel := range channels {
		out = append(out, notificationv1.NotificationChannel(channel))
	}
	return out
}

type streamHub struct {
	mu   sync.RWMutex
	subs map[string]map[chan domain.Notification]struct{}
}

func newStreamHub() *streamHub {
	return &streamHub{subs: map[string]map[chan domain.Notification]struct{}{}}
}

func (h *streamHub) subscribe(recipient string) (<-chan domain.Notification, func(), error) {
	ch := make(chan domain.Notification, 8)
	h.mu.Lock()
	if _, ok := h.subs[recipient]; !ok {
		h.subs[recipient] = map[chan domain.Notification]struct{}{}
	}
	h.subs[recipient][ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if recipientSubs, ok := h.subs[recipient]; ok {
			delete(recipientSubs, ch)
			if len(recipientSubs) == 0 {
				delete(h.subs, recipient)
			}
		}
		close(ch)
	}
	return ch, cancel, nil
}

func (h *streamHub) broadcast(recipient string, n domain.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subs[recipient] {
		select {
		case ch <- n:
		default:
		}
	}
}
