// Package kafka contains Kafka producer and consumer adapters for feed-service.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	"github.com/petmatch/petmatch/internal/feed"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	topicAnimalProfilePublished    = "animal.profile_published"
	topicAnimalProfileArchived     = "animal.profile_archived"
	topicAnimalStatusChanged       = "animal.status_changed"
	topicMatchingSwipeRecorded     = "matching.swipe_recorded"
	topicBillingBoostActivated     = "billing.boost_activated"
	topicBillingEntitlementGranted = "billing.entitlement_granted"
	topicAnimalStatsAggregated     = "analytics.animal_stats_aggregated"
)

// Message is the service-level Kafka message representation.
type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

// Writer writes Kafka messages.
type Writer interface {
	WriteMessages(context.Context, ...Message) error
}

// Reader reads Kafka messages and can be closed on shutdown.
type Reader interface {
	ReadMessage(context.Context) (Message, error)
	Close() error
}

// Handler processes consumed Kafka messages.
type Handler interface {
	Handle(context.Context, Message) error
}

// HandlerFunc adapts a function into a Handler.
type HandlerFunc func(context.Context, Message) error

// Handle processes a Kafka message.
func (f HandlerFunc) Handle(ctx context.Context, message Message) error {
	return f(ctx, message)
}

// Publisher serializes protobuf events and publishes them to Kafka.
type Publisher struct {
	writer Writer
}

// NewPublisher creates a Publisher.
func NewPublisher(writer Writer) *Publisher {
	return &Publisher{writer: writer}
}

// Publish marshals a protobuf event and writes it to the requested topic.
func (p *Publisher) Publish(ctx context.Context, topic string, event proto.Message) error {
	value, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal kafka event: %w", err)
	}
	message := Message{
		Topic: topic,
		Key:   []byte(partitionKey(event)),
		Value: value,
	}
	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

// Consumer reads Kafka messages until the context is cancelled.
type Consumer struct {
	reader  Reader
	handler Handler
}

// NewConsumer creates a Consumer.
func NewConsumer(reader Reader, handler Handler) *Consumer {
	return &Consumer{reader: reader, handler: handler}
}

// Run processes Kafka messages until cancellation or a non-cancellation error.
func (c *Consumer) Run(ctx context.Context) error {
	defer func() {
		_ = c.reader.Close()
	}()
	for {
		message, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("read kafka message: %w", err)
		}
		if err := c.handler.Handle(ctx, message); err != nil {
			return fmt.Errorf("handle kafka message topic %s: %w", message.Topic, err)
		}
	}
}

// InboundHandler handles subscribed domain events that affect feed materialization.
type InboundHandler struct {
	applier feed.EventApplier
	logger  *slog.Logger
}

// NewInboundHandler creates an inbound event handler.
func NewInboundHandler(applier feed.EventApplier, logger *slog.Logger) *InboundHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &InboundHandler{applier: applier, logger: logger}
}

// Handle receives invalidation/update events from upstream services and applies them.
func (h *InboundHandler) Handle(ctx context.Context, message Message) error {
	if h.applier == nil {
		h.logger.Info("received feed dependency event without applier", "topic", message.Topic, "key", string(message.Key), "bytes", len(message.Value))
		return nil
	}
	switch message.Topic {
	case topicAnimalProfilePublished:
		var event animalv1.AnimalPublishedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.UpsertAnimalProfile(ctx, event.Animal)
	case topicAnimalProfileArchived:
		var event feed.AnimalArchivedPayload
		if err := json.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.ArchiveAnimal(ctx, event)
	case topicAnimalStatusChanged:
		var event animalv1.AnimalStatusChangedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.UpdateAnimalStatus(ctx, &event)
	case topicMatchingSwipeRecorded:
		var event matchingv1.SwipeRecordedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.RecordSwipe(ctx, event.Swipe)
	case topicBillingBoostActivated:
		var event billingv1.BoostActivatedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.ActivateBoost(ctx, event.Boost)
	case topicBillingEntitlementGranted:
		var event billingv1.EntitlementGrantedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.GrantEntitlement(ctx, event.Entitlement)
	case topicAnimalStatsAggregated:
		var event analyticsv1.AnimalStatsAggregatedEvent
		if err := proto.Unmarshal(message.Value, &event); err != nil {
			return fmt.Errorf("decode %s: %w", message.Topic, err)
		}
		return h.applier.UpdateAnimalStats(ctx, event.Stats)
	default:
		h.logger.Info("ignored unsupported feed dependency event", "topic", message.Topic)
		return nil
	}
}

// KafkaGoWriter adapts kafka-go Writer to the service Writer interface.
type KafkaGoWriter struct {
	writer *kafka.Writer
}

// NewKafkaGoWriter creates a Kafka writer for dynamic topics.
func NewKafkaGoWriter(brokers []string) *KafkaGoWriter {
	return &KafkaGoWriter{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireAll,
			Async:        false,
		},
	}
}

// WriteMessages writes messages to Kafka.
func (w *KafkaGoWriter) WriteMessages(ctx context.Context, messages ...Message) error {
	kafkaMessages := make([]kafka.Message, 0, len(messages))
	for _, message := range messages {
		kafkaMessages = append(kafkaMessages, kafka.Message{
			Topic: message.Topic,
			Key:   message.Key,
			Value: message.Value,
		})
	}
	return w.writer.WriteMessages(ctx, kafkaMessages...)
}

// Close closes the Kafka writer.
func (w *KafkaGoWriter) Close() error {
	return w.writer.Close()
}

// KafkaGoReader adapts kafka-go Reader to the service Reader interface.
type KafkaGoReader struct {
	reader *kafka.Reader
}

// NewKafkaGoReader creates a Kafka reader subscribed to the provided topics.
func NewKafkaGoReader(brokers []string, groupID string, topics []string) *KafkaGoReader {
	return &KafkaGoReader{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupID:     groupID,
			GroupTopics: topics,
			MinBytes:    1,
			MaxBytes:    10e6,
		}),
	}
}

// ReadMessage reads one Kafka message.
func (r *KafkaGoReader) ReadMessage(ctx context.Context) (Message, error) {
	message, err := r.reader.ReadMessage(ctx)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Topic: message.Topic,
		Key:   message.Key,
		Value: message.Value,
	}, nil
}

// Close closes the Kafka reader.
func (r *KafkaGoReader) Close() error {
	return r.reader.Close()
}

func partitionKey(event proto.Message) string {
	reflectMessage := event.ProtoReflect()
	envelopeField := reflectMessage.Descriptor().Fields().ByName("envelope")
	if envelopeField == nil || !reflectMessage.Has(envelopeField) {
		return ""
	}
	envelope := reflectMessage.Get(envelopeField).Message()
	partitionField := envelope.Descriptor().Fields().ByName("partition_key")
	if partitionField == nil || !envelope.Has(partitionField) {
		return ""
	}
	return envelope.Get(partitionField).String()
}
