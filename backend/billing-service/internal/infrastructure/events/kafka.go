package events

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type KafkaMessage struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string]string
}

type KafkaWriter interface {
	WriteMessages(ctx context.Context, messages ...KafkaMessage) error
}

type KafkaPublisherConfig struct {
	ProducerName  string
	SchemaVersion string
}

type KafkaPublisher struct {
	writer        KafkaWriter
	producerName  string
	schemaVersion string
}

func NewKafkaPublisher(writer KafkaWriter, cfg KafkaPublisherConfig) *KafkaPublisher {
	if cfg.ProducerName == "" {
		cfg.ProducerName = "billing-service"
	}
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = "1.0.0"
	}
	return &KafkaPublisher{
		writer:        writer,
		producerName:  cfg.ProducerName,
		schemaVersion: cfg.SchemaVersion,
	}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event application.BillingEvent) error {
	payload, err := p.marshal(event)
	if err != nil {
		return err
	}
	message := KafkaMessage{
		Topic: event.Topic,
		Key:   []byte(event.PartitionKey),
		Value: payload,
		Headers: map[string]string{
			"content-type":     "application/x-protobuf",
			"event-type":       event.Topic,
			"schema-version":   p.schemaVersion,
			"trace-id":         event.TraceID,
			"correlation-id":   event.CorrelationID,
			"idempotency-key":  event.IdempotencyKey,
			"producer-service": p.producerName,
		},
	}
	if err := p.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("write kafka event %s: %w", event.Topic, err)
	}
	return nil
}

func (p *KafkaPublisher) marshal(event application.BillingEvent) ([]byte, error) {
	envelope := p.envelope(event)
	var message proto.Message
	switch event.Topic {
	case "billing.donation_succeeded":
		donation, ok := event.Payload.(domain.Donation)
		if !ok {
			return nil, fmt.Errorf("kafka event %s expects domain.Donation payload", event.Topic)
		}
		message = &billingv1.DonationSucceededEvent{Envelope: envelope, Donation: donationToProto(donation)}
	case "billing.donation_failed":
		donation, ok := event.Payload.(domain.Donation)
		if !ok {
			return nil, fmt.Errorf("kafka event %s expects domain.Donation payload", event.Topic)
		}
		message = &billingv1.DonationFailedEvent{
			Envelope:      envelope,
			Donation:      donationToProto(donation),
			FailureReason: donation.FailureReason,
		}
	case "billing.boost_activated":
		boost, ok := event.Payload.(domain.Boost)
		if !ok {
			return nil, fmt.Errorf("kafka event %s expects domain.Boost payload", event.Topic)
		}
		message = &billingv1.BoostActivatedEvent{Envelope: envelope, Boost: boostToProto(boost)}
	case "billing.entitlement_granted":
		entitlement, ok := event.Payload.(domain.Entitlement)
		if !ok {
			return nil, fmt.Errorf("kafka event %s expects domain.Entitlement payload", event.Topic)
		}
		message = &billingv1.EntitlementGrantedEvent{Envelope: envelope, Entitlement: entitlementToProto(entitlement)}
	default:
		return nil, fmt.Errorf("unsupported kafka event topic %q", event.Topic)
	}
	payload, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal kafka event %s: %w", event.Topic, err)
	}
	return payload, nil
}

func (p *KafkaPublisher) envelope(event application.BillingEvent) *commonv1.EventEnvelope {
	return &commonv1.EventEnvelope{
		EventId:        uuid.NewString(),
		EventType:      event.Topic,
		SchemaVersion:  p.schemaVersion,
		Producer:       p.producerName,
		OccurredAt:     timestamppb.New(time.Now().UTC()),
		TraceId:        event.TraceID,
		CorrelationId:  event.CorrelationID,
		IdempotencyKey: event.IdempotencyKey,
		PartitionKey:   event.PartitionKey,
	}
}
