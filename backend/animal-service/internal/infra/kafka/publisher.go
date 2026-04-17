// Package kafka contains Kafka adapters for animal-service.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"github.com/petmatch/petmatch/internal/adapter/pbconv"
	domain "github.com/petmatch/petmatch/internal/domain/animal"
	"github.com/petmatch/petmatch/internal/pkg/resilience"
	"github.com/segmentio/kafka-go"
	"github.com/sony/gobreaker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Publisher writes animal events to Kafka.
type Publisher struct {
	writer  *kafka.Writer
	breaker *gobreaker.CircuitBreaker
}

// NewPublisher creates a Kafka event publisher.
func NewPublisher(brokers []string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.Hash{},
			RequiredAcks: kafka.RequireOne,
			Async:        false,
		},
		breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "kafka-publisher",
			MaxRequests: 2,
			Interval:    time.Minute,
			Timeout:     30 * time.Second,
		}),
	}
}

// Publish publishes one domain event.
func (p *Publisher) Publish(ctx context.Context, event domain.Event) error {
	message, err := p.message(event)
	if err != nil {
		return err
	}
	_, err = p.breaker.Execute(func() (any, error) {
		err := resilience.Retry(ctx, 3, 100*time.Millisecond, func(ctx context.Context) error {
			return p.writer.WriteMessages(ctx, message)
		})
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("write kafka event %s: %w", event.Type, err)
	}
	return nil
}

// Close closes the underlying Kafka writer.
func (p *Publisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close kafka writer: %w", err)
	}
	return nil
}

func (p *Publisher) message(event domain.Event) (kafka.Message, error) {
	payload, contentType, err := marshalEvent(event)
	if err != nil {
		return kafka.Message{}, err
	}
	return kafka.Message{
		Topic: event.Type,
		Key:   []byte(event.AnimalID),
		Value: payload,
		Time:  event.OccurredAt,
		Headers: []kafka.Header{
			{Key: "content-type", Value: []byte(contentType)},
			{Key: "event-type", Value: []byte(event.Type)},
			{Key: "schema-version", Value: []byte("v1")},
		},
	}, nil
}

func marshalEvent(event domain.Event) ([]byte, string, error) {
	envelope := toEnvelope(event)
	switch event.Type {
	case domain.EventProfileCreated:
		payload := &animalv1.AnimalProfileCreatedEvent{
			Envelope: envelope,
			Animal:   pbconv.ToAnimalProfile(derefProfile(event.Animal)),
		}
		data, err := proto.Marshal(payload)
		return data, "application/x-protobuf", err
	case domain.EventProfileUpdated:
		payload := &animalv1.AnimalProfileUpdatedEvent{
			Envelope: envelope,
			Animal:   pbconv.ToAnimalProfile(derefProfile(event.Animal)),
		}
		data, err := proto.Marshal(payload)
		return data, "application/x-protobuf", err
	case domain.EventProfilePublished:
		payload := &animalv1.AnimalPublishedEvent{
			Envelope: envelope,
			Animal:   pbconv.ToAnimalProfile(derefProfile(event.Animal)),
		}
		data, err := proto.Marshal(payload)
		return data, "application/x-protobuf", err
	case domain.EventPhotoAdded:
		payload := &animalv1.AnimalPhotoAddedEvent{
			Envelope: envelope,
			AnimalId: event.AnimalID,
			Photo:    pbconv.ToPhoto(derefPhoto(event.Photo)),
		}
		data, err := proto.Marshal(payload)
		return data, "application/x-protobuf", err
	case domain.EventStatusChanged:
		payload := &animalv1.AnimalStatusChangedEvent{
			Envelope:  envelope,
			AnimalId:  event.AnimalID,
			OldStatus: animalv1.AnimalStatus(event.OldStatus),
			NewStatus: animalv1.AnimalStatus(event.NewStatus),
			Reason:    event.Reason,
		}
		data, err := proto.Marshal(payload)
		return data, "application/x-protobuf", err
	case domain.EventProfileArchived:
		payload := archivedEvent{
			Envelope:       envelopeJSONFromEvent(event),
			AnimalID:       event.AnimalID,
			OwnerProfileID: event.OwnerProfileID,
			PreviousStatus: int32(event.OldStatus),
			Reason:         event.Reason,
		}
		data, err := json.Marshal(payload)
		return data, "application/json", err
	default:
		return nil, "", fmt.Errorf("%w: unsupported event type %q", domain.ErrInvalidArgument, event.Type)
	}
}

func toEnvelope(event domain.Event) *commonv1.EventEnvelope {
	return &commonv1.EventEnvelope{
		EventId:        event.ID,
		EventType:      event.Type,
		SchemaVersion:  "v1",
		Producer:       "animal-service",
		OccurredAt:     timestamppb.New(event.OccurredAt),
		TraceId:        event.TraceID,
		CorrelationId:  event.CorrelationID,
		IdempotencyKey: event.IdempotencyKey,
		PartitionKey:   event.AnimalID,
	}
}

type archivedEvent struct {
	Envelope       envelopeJSON `json:"envelope"`
	AnimalID       string       `json:"animal_id"`
	OwnerProfileID string       `json:"owner_profile_id"`
	PreviousStatus int32        `json:"previous_status"`
	Reason         string       `json:"reason"`
}

type envelopeJSON struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	SchemaVersion  string `json:"schema_version"`
	Producer       string `json:"producer"`
	OccurredAt     string `json:"occurred_at"`
	TraceID        string `json:"trace_id,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	PartitionKey   string `json:"partition_key"`
}

func envelopeJSONFromEvent(event domain.Event) envelopeJSON {
	return envelopeJSON{
		EventID:        event.ID,
		EventType:      event.Type,
		SchemaVersion:  "v1",
		Producer:       "animal-service",
		OccurredAt:     event.OccurredAt.Format(time.RFC3339Nano),
		TraceID:        event.TraceID,
		CorrelationID:  event.CorrelationID,
		IdempotencyKey: event.IdempotencyKey,
		PartitionKey:   event.AnimalID,
	}
}

func derefProfile(profile *domain.Profile) domain.Profile {
	if profile == nil {
		return domain.Profile{}
	}
	return *profile
}

func derefPhoto(photo *domain.Photo) domain.Photo {
	if photo == nil {
		return domain.Photo{}
	}
	return *photo
}
