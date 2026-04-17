package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	appanimal "github.com/petmatch/petmatch/internal/app/animal"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	topicBoostActivated    = "billing.boost_activated"
	topicDonationSucceeded = "billing.donation_succeeded"
	topicMatchCreated      = "matching.match_created"
)

// ConsumerGroup consumes cross-service events that affect animals.
type ConsumerGroup struct {
	readers []*kafka.Reader
	service *appanimal.Service
	logger  *slog.Logger
}

// NewConsumerGroup creates Kafka readers for animal-service subscriptions.
func NewConsumerGroup(brokers []string, groupID string, service *appanimal.Service, logger *slog.Logger) *ConsumerGroup {
	topics := []string{topicBoostActivated, topicDonationSucceeded, topicMatchCreated}
	readers := make([]*kafka.Reader, 0, len(topics))
	for _, topic := range topics {
		readers = append(readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID,
			Topic:          topic,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}))
	}
	return &ConsumerGroup{readers: readers, service: service, logger: logger}
}

// Run consumes messages until ctx is cancelled.
func (g *ConsumerGroup) Run(ctx context.Context) error {
	errCh := make(chan error, len(g.readers))
	for _, reader := range g.readers {
		reader := reader
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					errCh <- fmt.Errorf("panic in kafka consumer %s: %v", reader.Config().Topic, recovered)
				}
			}()
			errCh <- g.consume(ctx, reader)
		}()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Close closes all Kafka readers.
func (g *ConsumerGroup) Close() error {
	var result error
	for _, reader := range g.readers {
		if err := reader.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close kafka reader %s: %w", reader.Config().Topic, err))
		}
	}
	return result
}

func (g *ConsumerGroup) consume(ctx context.Context, reader *kafka.Reader) error {
	topic := reader.Config().Topic
	for {
		message, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("read kafka topic %s: %w", topic, err)
		}
		if err := g.handle(ctx, topic, message.Value); err != nil {
			g.logger.ErrorContext(ctx, "failed to handle kafka message", "topic", topic, "error", err)
			continue
		}
	}
}

func (g *ConsumerGroup) handle(ctx context.Context, topic string, payload []byte) error {
	switch topic {
	case topicBoostActivated:
		var event billingv1.BoostActivatedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode boost activated event: %w", err)
		}
		return g.service.ApplyBoostActivated(ctx, appanimal.BoostActivatedCommand{
			ActorID:   "billing-service",
			AnimalID:  event.GetBoost().GetAnimalId(),
			ExpiresAt: event.GetBoost().GetExpiresAt().AsTime(),
		})
	case topicDonationSucceeded:
		var event billingv1.DonationSucceededEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode donation succeeded event: %w", err)
		}
		if event.GetDonation().GetTargetType() != billingv1.DonationTargetType_DONATION_TARGET_TYPE_ANIMAL {
			return nil
		}
		return g.service.ApplyDonationSucceeded(ctx, appanimal.DonationSucceededCommand{
			ActorID:  "billing-service",
			AnimalID: event.GetDonation().GetTargetId(),
		})
	case topicMatchCreated:
		var event matchingv1.MatchCreatedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode match created event: %w", err)
		}
		return g.service.ApplyMatchCreated(ctx, appanimal.MatchCreatedCommand{
			ActorID:  "matching-service",
			AnimalID: event.GetMatch().GetAnimalId(),
		})
	default:
		return fmt.Errorf("unsupported topic %q", topic)
	}
}
