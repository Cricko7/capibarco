// Package kafka publishes gateway operational events.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/petmatch/petmatch/internal/config"
	"github.com/segmentio/kafka-go"
)

const (
	TopicRequestRejected       = "gateway.request_rejected"
	TopicWebSocketConnected    = "gateway.websocket_connected"
	TopicWebSocketDisconnected = "gateway.websocket_disconnected"
)

// Publisher emits operational gateway events.
type Publisher interface {
	Publish(ctx context.Context, topic string, key string, payload any) error
	Close() error
}

// NoopPublisher is used when Kafka is disabled.
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, string, string, any) error { return nil }
func (NoopPublisher) Close() error                                       { return nil }

// KafkaPublisher writes JSON payloads to Kafka.
type KafkaPublisher struct {
	writer *kafka.Writer
}

// New creates a Kafka publisher.
func New(cfg config.KafkaConfig) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(cfg.Brokers...),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireOne,
			Async:                  false,
			BatchTimeout:           10 * time.Millisecond,
			Transport:              &kafka.Transport{ClientID: cfg.ClientID},
		},
	}
}

// Publish writes a JSON event.
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal kafka payload: %w", err)
	}
	if key == "" {
		key = uuid.NewString()
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(key), Value: body, Time: time.Now().UTC()}); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

// Close closes the writer.
func (p *KafkaPublisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close kafka writer: %w", err)
	}
	return nil
}
