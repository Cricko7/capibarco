package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	chatv1 "github.com/petmatch/petmatch/gen/go/petmatch/chat/v1"
	app "github.com/petmatch/petmatch/internal/app/matching"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	TopicAnimalProfileArchived  = "animal.profile_archived"
	TopicAnimalStatusChanged    = "animal.status_changed"
	TopicChatConversationCreate = "chat.conversation_created"
)

// Consumer handles Kafka events required by matching-service.
type Consumer struct {
	brokers []string
	groupID string
	service *app.Service
	logger  *slog.Logger
}

// NewConsumer creates a Kafka consumer group adapter.
func NewConsumer(brokers []string, groupID string, service *app.Service, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{brokers: brokers, groupID: groupID, service: service, logger: logger}
}

// Run starts topic readers until the context is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	if c == nil || c.service == nil {
		return errors.New("kafka consumer is not configured")
	}
	topics := []string{TopicAnimalProfileArchived, TopicAnimalStatusChanged, TopicChatConversationCreate}
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
			continue
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit kafka message %s/%d/%d: %w", topic, msg.Partition, msg.Offset, err)
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, topic string, payload []byte) error {
	switch topic {
	case TopicAnimalStatusChanged:
		var event animalv1.AnimalStatusChangedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode animal status changed: %w", err)
		}
		available := event.NewStatus == animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE
		_, err := c.service.HandleAnimalStatusChanged(ctx, event.GetAnimalId(), "", available, event.GetReason())
		return err
	case TopicAnimalProfileArchived:
		var event animalProfileArchivedEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode animal profile archived: %w", err)
		}
		if strings.TrimSpace(event.AnimalID) == "" {
			return errors.New("animal profile archived event missing animal_id")
		}
		_, err := c.service.HandleAnimalArchived(ctx, event.AnimalID, event.Reason)
		return err
	case TopicChatConversationCreate:
		var event chatv1.ConversationCreatedEvent
		if err := proto.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("decode chat conversation created: %w", err)
		}
		conversation := event.GetConversation()
		if conversation == nil {
			return errors.New("chat conversation created event missing conversation")
		}
		return c.service.HandleConversationCreated(ctx, conversation.GetMatchId(), conversation.GetConversationId())
	default:
		return nil
	}
}

type animalProfileArchivedEvent struct {
	AnimalID       string `json:"animal_id"`
	OwnerProfileID string `json:"owner_profile_id"`
	PreviousStatus string `json:"previous_status"`
	Reason         string `json:"reason"`
}
