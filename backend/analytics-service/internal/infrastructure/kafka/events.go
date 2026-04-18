package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
)

type EventConsumer struct {
	reader  *kafka.Reader
	service *application.Service
}

type EventMessage struct {
	EventID    string            `json:"event_id"`
	ProfileID  string            `json:"profile_id"`
	ActorID    string            `json:"actor_id"`
	Type       string            `json:"type"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata"`
}

func NewEventConsumer(brokers []string, groupID, topic string, service *application.Service) *EventConsumer {
	return &EventConsumer{reader: kafka.NewReader(kafka.ReaderConfig{Brokers: brokers, GroupID: groupID, Topic: topic}), service: service}
}

func (c *EventConsumer) Close() error {
	return c.reader.Close()
}

func (c *EventConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch kafka message: %w", err)
		}
		var eventMsg EventMessage
		if err := json.Unmarshal(msg.Value, &eventMsg); err != nil {
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}
		event := domain.Event{EventID: eventMsg.EventID, ProfileID: eventMsg.ProfileID, ActorID: eventMsg.ActorID, Type: domain.EventType(eventMsg.Type), OccurredAt: eventMsg.OccurredAt, Metadata: eventMsg.Metadata}
		if err := c.service.IngestEvent(ctx, event); err != nil {
			if err == application.ErrDuplicateEvent {
				_ = c.reader.CommitMessages(ctx, msg)
				continue
			}
			return fmt.Errorf("ingest from kafka: %w", err)
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return fmt.Errorf("commit kafka message: %w", err)
		}
	}
}

type RankingPublisher struct {
	writer *kafka.Writer
}

func NewRankingPublisher(brokers []string, topic string) *RankingPublisher {
	return &RankingPublisher{writer: &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: topic, Balancer: &kafka.LeastBytes{}}}
}

func (p *RankingPublisher) Close() error { return p.writer.Close() }

func (p *RankingPublisher) PublishRankingFeedback(ctx context.Context, items []domain.RankingFeedback) error {
	payload, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal ranking feedback: %w", err)
	}
	if err := p.writer.WriteMessages(ctx, kafka.Message{Value: payload}); err != nil {
		return fmt.Errorf("publish ranking feedback kafka: %w", err)
	}
	return nil
}
