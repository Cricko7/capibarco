// Package kafka contains Kafka adapters for matching-service.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/petmatch/petmatch/internal/infra/postgres"
	"github.com/petmatch/petmatch/internal/pkg/safe"
	"github.com/segmentio/kafka-go"
)

// OutboxStore is the persistence contract for the Kafka outbox publisher.
type OutboxStore interface {
	FetchOutbox(ctx context.Context, limit int) ([]postgres.OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, eventID string) error
	MarkOutboxFailed(ctx context.Context, eventID string, cause error) error
}

// OutboxPublisher publishes transactional outbox events to Kafka.
type OutboxPublisher struct {
	store    OutboxStore
	writer   *kafka.Writer
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

// NewOutboxPublisher creates a Kafka outbox publisher.
func NewOutboxPublisher(store OutboxStore, brokers []string, logger *slog.Logger, clientID string, interval time.Duration, batch int) *OutboxPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	if interval <= 0 {
		interval = time.Second
	}
	if batch <= 0 {
		batch = 50
	}
	return &OutboxPublisher{
		store: store,
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireAll,
			Async:                  false,
			Transport: &kafka.Transport{
				ClientID: clientID,
			},
		},
		logger:   logger,
		interval: interval,
		batch:    batch,
	}
}

// Run publishes outbox events until the context is cancelled.
func (p *OutboxPublisher) Run(ctx context.Context) error {
	if p == nil || p.store == nil || p.writer == nil {
		return errors.New("kafka outbox publisher is not configured")
	}
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	defer func() {
		if err := p.writer.Close(); err != nil {
			p.logger.Warn("close kafka writer", "error", err)
		}
	}()

	for {
		if err := p.flush(ctx); err != nil && !errors.Is(err, context.Canceled) {
			p.logger.Error("flush outbox", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *OutboxPublisher) flush(ctx context.Context) error {
	events, err := p.store.FetchOutbox(ctx, p.batch)
	if err != nil {
		return err
	}
	for _, event := range events {
		msg := kafka.Message{
			Topic: event.Topic,
			Key:   []byte(event.PartitionKey),
			Value: event.Payload,
			Time:  event.CreatedAt,
			Headers: []kafka.Header{
				{Key: "event_id", Value: []byte(event.ID)},
				{Key: "event_type", Value: []byte(event.EventType)},
				{Key: "content_type", Value: []byte("application/protobuf")},
			},
		}
		if err := p.writer.WriteMessages(ctx, msg); err != nil {
			markErr := p.store.MarkOutboxFailed(ctx, event.ID, err)
			return errors.Join(fmt.Errorf("publish outbox event %s: %w", event.ID, err), markErr)
		}
		if err := p.store.MarkOutboxPublished(ctx, event.ID); err != nil {
			return err
		}
	}
	return nil
}

// StartOutboxPublisher starts publisher in a panic-safe goroutine.
func StartOutboxPublisher(ctx context.Context, publisher *OutboxPublisher, logger *slog.Logger) <-chan error {
	errCh := make(chan error, 1)
	safe.Go(ctx, logger, "kafka-outbox-publisher", func(ctx context.Context) {
		err := publisher.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
		close(errCh)
	})
	return errCh
}
