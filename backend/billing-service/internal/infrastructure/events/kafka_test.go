package events_test

import (
	"context"
	"testing"
	"time"

	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"github.com/petmatch/petmatch/internal/infrastructure/events"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestKafkaPublisherPublishesDonationSucceededProtobuf(t *testing.T) {
	t.Parallel()

	writer := &recordingWriter{}
	publisher := events.NewKafkaPublisher(writer, events.KafkaPublisherConfig{
		ProducerName:  "billing-service",
		SchemaVersion: "1.0.0",
	})
	donation := testDonation(t)

	err := publisher.Publish(context.Background(), application.BillingEvent{
		Topic:          "billing.donation_succeeded",
		PartitionKey:   donation.ID,
		Type:           "DonationSucceededEvent",
		TraceID:        "trace_1",
		CorrelationID:  "corr_1",
		IdempotencyKey: "idem_1",
		Payload:        donation,
	})

	require.NoError(t, err)
	require.Len(t, writer.messages, 1)
	require.Equal(t, "billing.donation_succeeded", writer.messages[0].Topic)
	require.Equal(t, []byte(donation.ID), writer.messages[0].Key)

	var event billingv1.DonationSucceededEvent
	require.NoError(t, proto.Unmarshal(writer.messages[0].Value, &event))
	require.Equal(t, donation.ID, event.GetDonation().GetDonationId())
	require.Equal(t, "billing.donation_succeeded", event.GetEnvelope().GetEventType())
	require.Equal(t, "billing-service", event.GetEnvelope().GetProducer())
	require.Equal(t, "trace_1", event.GetEnvelope().GetTraceId())
	require.Equal(t, "corr_1", event.GetEnvelope().GetCorrelationId())
	require.Equal(t, "idem_1", event.GetEnvelope().GetIdempotencyKey())
}

func TestKafkaPublisherRejectsUnsupportedEventPayload(t *testing.T) {
	t.Parallel()

	writer := &recordingWriter{}
	publisher := events.NewKafkaPublisher(writer, events.KafkaPublisherConfig{
		ProducerName:  "billing-service",
		SchemaVersion: "1.0.0",
	})

	err := publisher.Publish(context.Background(), application.BillingEvent{
		Topic:        "billing.donation_succeeded",
		PartitionKey: "don_1",
		Type:         "DonationSucceededEvent",
		Payload:      "not a donation",
	})

	require.Error(t, err)
	require.Empty(t, writer.messages)
}

type recordingWriter struct {
	messages []events.KafkaMessage
}

func (w *recordingWriter) WriteMessages(_ context.Context, messages ...events.KafkaMessage) error {
	w.messages = append(w.messages, messages...)
	return nil
}

func testDonation(t *testing.T) domain.Donation {
	t.Helper()

	amount, err := domain.NewMoney("USD", 10, 0)
	require.NoError(t, err)
	donation, err := domain.NewDonation(domain.NewDonationParams{
		ID:             "don_1",
		PayerProfileID: "profile_1",
		TargetType:     domain.DonationTargetAnimal,
		TargetID:       "animal_1",
		Amount:         amount,
		Provider:       "mock",
		CreatedAt:      time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.NoError(t, donation.MarkSucceeded("pay_1", donation.CreatedAt.Add(time.Minute)))
	return donation
}
