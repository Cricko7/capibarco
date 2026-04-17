package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	"github.com/segmentio/kafka-go"
)

// PublisherConfig configures Kafka event publishing.
type PublisherConfig struct {
	Brokers                []string
	ClientID               string
	AllowAutoTopicCreation bool
}

// Publisher writes auth events to Kafka topics.
type Publisher struct {
	writer *kafka.Writer
}

// NewPublisher creates a Kafka publisher.
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = domain.EventProducerAuthService
	}
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: cfg.AllowAutoTopicCreation,
		BatchTimeout:           10 * time.Millisecond,
		RequiredAcks:           kafka.RequireAll,
		Transport: &kafka.Transport{
			ClientID: clientID,
		},
	}
	return &Publisher{writer: writer}, nil
}

// Publish writes one event to its event_type topic.
func (p *Publisher) Publish(ctx context.Context, event domain.Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal auth event: %w", err)
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: event.Type,
		Key:   []byte(event.Key),
		Value: value,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.Type)},
			{Key: "schema_version", Value: []byte(event.SchemaVersion)},
			{Key: "producer", Value: []byte(event.Producer)},
		},
		Time: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("publish auth event %s: %w", event.Type, err)
	}
	return nil
}

// Close closes the underlying writer.
func (p *Publisher) Close() error {
	return p.writer.Close()
}

// SlogPublisher logs event envelopes instead of sending them to Kafka.
type SlogPublisher struct {
	logger *slog.Logger
}

// NewSlogPublisher creates a development publisher.
func NewSlogPublisher(logger *slog.Logger) *SlogPublisher {
	return &SlogPublisher{logger: logger}
}

// Publish writes the event envelope to structured logs.
func (p *SlogPublisher) Publish(ctx context.Context, event domain.Event) error {
	p.logger.InfoContext(ctx, "auth event",
		slog.String("topic", event.Type),
		slog.String("key", event.Key),
		slog.String("event_id", event.ID),
		slog.Any("payload", event.Payload),
	)
	return nil
}
