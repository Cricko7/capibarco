package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	userv1 "github.com/petmatch/petmatch/gen/go/petmatch/user/v1"
	app "github.com/petmatch/petmatch/internal/app/notification"
	domain "github.com/petmatch/petmatch/internal/domain/notification"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	TopicMatchCreated      = "matching.match_created"
	TopicMessageSent       = "chat.message_sent"
	TopicDonationSucceeded = "billing.donation_succeeded"
	TopicBoostActivated    = "billing.boost_activated"
	TopicReviewCreated     = "user.review_created"
)

var errSkipMessage = errors.New("skip kafka message")

type Consumer struct {
	brokers []string
	groupID string
	service *app.Service
	logger  *slog.Logger
}

func NewConsumer(brokers []string, groupID string, service *app.Service, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{brokers: brokers, groupID: groupID, service: service, logger: logger}
}

func (c *Consumer) Run(ctx context.Context) error {
	if c == nil || c.service == nil {
		return errors.New("kafka consumer is not configured")
	}
	topics := []string{TopicMatchCreated, TopicMessageSent, TopicDonationSucceeded, TopicBoostActivated, TopicReviewCreated}
	errCh := make(chan error, len(topics))
	for _, topic := range topics {
		topic := topic
		safe.Go(ctx, c.logger, "kafka-consumer-"+topic, func(ctx context.Context) {
			errCh <- c.consumeTopic(ctx, topic)
		})
	}
	var joined error
	for range topics {
		err := <-errCh
		if err != nil && !errors.Is(err, context.Canceled) {
			joined = errors.Join(joined, err)
		}
	}
	return joined
}

func (c *Consumer) consumeTopic(ctx context.Context, topic string) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		GroupID:        c.groupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10 << 20,
		CommitInterval: time.Second,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			c.logger.Warn("close kafka reader", "topic", topic, "error", err)
		}
	}()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			c.logger.Error("fetch kafka message", "topic", topic, "error", err)
			continue
		}
		if err := c.handleMessage(ctx, topic, msg.Value); err != nil {
			c.logger.Error("handle kafka message", "topic", topic, "partition", msg.Partition, "offset", msg.Offset, "error", err)
			if errors.Is(err, errSkipMessage) {
				if commitErr := reader.CommitMessages(ctx, msg); commitErr != nil {
					return fmt.Errorf("commit skipped kafka message %s/%d/%d: %w", topic, msg.Partition, msg.Offset, commitErr)
				}
			}
			continue
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit kafka message %s/%d/%d: %w", topic, msg.Partition, msg.Offset, err)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, topic string, payload []byte) error {
	switch topic {
	case TopicMatchCreated:
		var event matchingv1.MatchCreatedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("%w: decode match created: %w", errSkipMessage, err)
		}
		return skipInvalidEvent(c.service.HandleMatchCreated(ctx, &event))
	case TopicMessageSent:
		var event chatv1.MessageSentEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("%w: decode message sent: %w", errSkipMessage, err)
		}
		return skipInvalidEvent(c.service.HandleMessageSent(ctx, &event))
	case TopicDonationSucceeded:
		var event billingv1.DonationSucceededEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("%w: decode donation succeeded: %w", errSkipMessage, err)
		}
		return skipInvalidEvent(c.service.HandleDonationSucceeded(ctx, &event))
	case TopicBoostActivated:
		var event billingv1.BoostActivatedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("%w: decode boost activated: %w", errSkipMessage, err)
		}
		return skipInvalidEvent(c.service.HandleBoostActivated(ctx, &event))
	case TopicReviewCreated:
		var event userv1.ReviewCreatedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("%w: decode review created: %w", errSkipMessage, err)
		}
		return skipInvalidEvent(c.service.HandleReviewCreated(ctx, &event))
	default:
		return nil
	}
}

func skipInvalidEvent(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, domain.ErrInvalidArgument) {
		return fmt.Errorf("%w: %w", errSkipMessage, err)
	}
	return err
}
