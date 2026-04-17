package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	analyticsv1 "github.com/petmatch/petmatch/gen/go/petmatch/analytics/v1"
	animalv1 "github.com/petmatch/petmatch/gen/go/petmatch/animal/v1"
	billingv1 "github.com/petmatch/petmatch/gen/go/petmatch/billing/v1"
	commonv1 "github.com/petmatch/petmatch/gen/go/petmatch/common/v1"
	feedv1 "github.com/petmatch/petmatch/gen/go/petmatch/feed/v1"
	matchingv1 "github.com/petmatch/petmatch/gen/go/petmatch/matching/v1"
	"github.com/petmatch/petmatch/internal/feed"
	"google.golang.org/protobuf/proto"
)

func TestPublisherMarshalsProtoWithEnvelopePartitionKey(t *testing.T) {
	writer := &fakeWriter{}
	publisher := NewPublisher(writer)

	err := publisher.Publish(context.Background(), "feed.card_served", &feedv1.FeedCardServedEvent{
		Envelope:   &commonv1.EventEnvelope{PartitionKey: "animal-1"},
		FeedCardId: "card-1",
		AnimalId:   "animal-1",
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}

	if writer.topic != "feed.card_served" {
		t.Fatalf("topic = %q, want feed.card_served", writer.topic)
	}
	if string(writer.key) != "animal-1" {
		t.Fatalf("key = %q, want animal-1", string(writer.key))
	}
	var event feedv1.FeedCardServedEvent
	if err := proto.Unmarshal(writer.value, &event); err != nil {
		t.Fatalf("unmarshal published value: %v", err)
	}
	if event.FeedCardId != "card-1" {
		t.Fatalf("feed_card_id = %q, want card-1", event.FeedCardId)
	}
}

func TestPublishedFeedEventsUseContractPartitionKeys(t *testing.T) {
	tests := []struct {
		name  string
		topic string
		event proto.Message
		want  string
	}{
		{
			name:  "card served is keyed by feed session",
			topic: "feed.card_served",
			event: &feedv1.FeedCardServedEvent{
				Envelope:      &commonv1.EventEnvelope{PartitionKey: "session-1"},
				FeedSessionId: "session-1",
				AnimalId:      "animal-1",
			},
			want: "session-1",
		},
		{
			name:  "card opened is keyed by animal",
			topic: "feed.card_opened",
			event: &feedv1.FeedCardOpenedEvent{
				Envelope: &commonv1.EventEnvelope{PartitionKey: "animal-1"},
				AnimalId: "animal-1",
			},
			want: "animal-1",
		},
		{
			name:  "filters applied is keyed by actor",
			topic: "feed.filters_applied",
			event: &feedv1.FeedFiltersAppliedEvent{
				Envelope: &commonv1.EventEnvelope{PartitionKey: "actor-1"},
				ActorId:  stringPtr("actor-1"),
			},
			want: "actor-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &fakeWriter{}
			err := NewPublisher(writer).Publish(context.Background(), tt.topic, tt.event)
			if err != nil {
				t.Fatalf("Publish returned error: %v", err)
			}
			if string(writer.key) != tt.want {
				t.Fatalf("partition key = %q, want %q", string(writer.key), tt.want)
			}
		})
	}
}

func TestInboundHandlerAppliesConsumedPayloadContracts(t *testing.T) {
	applier := &recordingApplier{}
	handler := NewInboundHandler(applier, nil)
	published := &animalv1.AnimalPublishedEvent{
		Animal: &animalv1.AnimalProfile{AnimalId: "published-1", OwnerProfileId: "owner-1"},
	}
	archivedPayload, err := json.Marshal(feed.AnimalArchivedPayload{
		AnimalID:       "archived-1",
		OwnerProfileID: "owner-2",
		PreviousStatus: animalv1.AnimalStatus_ANIMAL_STATUS_AVAILABLE.String(),
		Reason:         "owner archived",
	})
	if err != nil {
		t.Fatalf("marshal archive payload: %v", err)
	}
	statusChanged := &animalv1.AnimalStatusChangedEvent{
		AnimalId:  "status-1",
		NewStatus: animalv1.AnimalStatus_ANIMAL_STATUS_ADOPTED,
	}
	swipeRecorded := &matchingv1.SwipeRecordedEvent{
		Swipe: &matchingv1.Swipe{ActorId: "actor-1", AnimalId: "swiped-1"},
	}
	boostActivated := &billingv1.BoostActivatedEvent{
		Boost: &billingv1.Boost{AnimalId: "boosted-1", Active: true},
	}
	entitlementGranted := &billingv1.EntitlementGrantedEvent{
		Entitlement: &billingv1.Entitlement{
			OwnerProfileId: "owner-3",
			Type:           billingv1.EntitlementType_ENTITLEMENT_TYPE_ADVANCED_FILTERS,
			Active:         true,
		},
	}
	statsAggregated := &analyticsv1.AnimalStatsAggregatedEvent{
		Stats: &analyticsv1.AnimalStats{AnimalId: "stats-1", Ctr: 0.42},
	}

	messages := []Message{
		protoMessage(t, "animal.profile_published", published),
		{Topic: "animal.profile_archived", Value: archivedPayload},
		protoMessage(t, "animal.status_changed", statusChanged),
		protoMessage(t, "matching.swipe_recorded", swipeRecorded),
		protoMessage(t, "billing.boost_activated", boostActivated),
		protoMessage(t, "billing.entitlement_granted", entitlementGranted),
		protoMessage(t, "analytics.animal_stats_aggregated", statsAggregated),
	}

	for _, message := range messages {
		if err := handler.Handle(context.Background(), message); err != nil {
			t.Fatalf("Handle(%s) returned error: %v", message.Topic, err)
		}
	}

	assertStrings(t, applier.calls, []string{
		"upsert:published-1",
		"archive:archived-1",
		"status:status-1:ANIMAL_STATUS_ADOPTED",
		"swipe:actor-1:swiped-1",
		"boost:boosted-1:true",
		"entitlement:owner-3:true",
		"stats:stats-1:0.42",
	})
}

func TestConsumerRoutesMessagesAndStopsOnContextCancel(t *testing.T) {
	reader := &fakeReader{
		messages: []Message{
			{Topic: "animal.status_changed", Key: []byte("animal-1"), Value: []byte("payload")},
		},
	}
	handler := &recordingHandler{}
	consumer := NewConsumer(reader, handler)

	ctx, cancel := context.WithCancel(context.Background())
	err := consumer.Run(ctx)
	cancel()
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if handler.topic != "animal.status_changed" {
		t.Fatalf("handled topic = %q, want animal.status_changed", handler.topic)
	}
	if string(handler.value) != "payload" {
		t.Fatalf("handled value = %q, want payload", string(handler.value))
	}
}

func TestConsumerReturnsHandlerErrors(t *testing.T) {
	wantErr := errors.New("boom")
	reader := &fakeReader{messages: []Message{{Topic: "billing.boost_activated"}}}
	consumer := NewConsumer(reader, HandlerFunc(func(context.Context, Message) error {
		return wantErr
	}))

	err := consumer.Run(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error = %v, want %v", err, wantErr)
	}
}

type fakeWriter struct {
	topic string
	key   []byte
	value []byte
}

func (w *fakeWriter) WriteMessages(_ context.Context, messages ...Message) error {
	if len(messages) != 1 {
		return errors.New("expected one message")
	}
	w.topic = messages[0].Topic
	w.key = append([]byte(nil), messages[0].Key...)
	w.value = append([]byte(nil), messages[0].Value...)
	return nil
}

type fakeReader struct {
	messages []Message
	next     int
}

func (r *fakeReader) ReadMessage(ctx context.Context) (Message, error) {
	if r.next >= len(r.messages) {
		return Message{}, context.Canceled
	}
	message := r.messages[r.next]
	r.next++
	return message, ctx.Err()
}

func (r *fakeReader) Close() error {
	return nil
}

type recordingHandler struct {
	topic string
	value []byte
}

func (h *recordingHandler) Handle(_ context.Context, message Message) error {
	h.topic = message.Topic
	h.value = append([]byte(nil), message.Value...)
	return nil
}

type recordingApplier struct {
	calls []string
}

func (a *recordingApplier) UpsertAnimalProfile(_ context.Context, animal *animalv1.AnimalProfile) error {
	a.calls = append(a.calls, "upsert:"+animal.AnimalId)
	return nil
}

func (a *recordingApplier) ArchiveAnimal(_ context.Context, event feed.AnimalArchivedPayload) error {
	a.calls = append(a.calls, "archive:"+event.AnimalID)
	return nil
}

func (a *recordingApplier) UpdateAnimalStatus(_ context.Context, event *animalv1.AnimalStatusChangedEvent) error {
	a.calls = append(a.calls, "status:"+event.AnimalId+":"+event.NewStatus.String())
	return nil
}

func (a *recordingApplier) RecordSwipe(_ context.Context, swipe *matchingv1.Swipe) error {
	a.calls = append(a.calls, "swipe:"+swipe.ActorId+":"+swipe.AnimalId)
	return nil
}

func (a *recordingApplier) ActivateBoost(_ context.Context, boost *billingv1.Boost) error {
	a.calls = append(a.calls, "boost:"+boost.AnimalId+":true")
	return nil
}

func (a *recordingApplier) GrantEntitlement(_ context.Context, entitlement *billingv1.Entitlement) error {
	a.calls = append(a.calls, "entitlement:"+entitlement.OwnerProfileId+":true")
	return nil
}

func (a *recordingApplier) UpdateAnimalStats(_ context.Context, stats *analyticsv1.AnimalStats) error {
	a.calls = append(a.calls, "stats:"+stats.AnimalId+":0.42")
	return nil
}

func protoMessage(t *testing.T, topic string, message proto.Message) Message {
	t.Helper()
	value, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal %s: %v", topic, err)
	}
	return Message{Topic: topic, Value: value}
}

func stringPtr(value string) *string {
	return &value
}

func assertStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d = %q, want %q; full got %v", i, got[i], want[i], got)
		}
	}
}
