package kafka

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/petmatch/petmatch/internal/pkg/resilience"
	"github.com/segmentio/kafka-go"
	"github.com/sony/gobreaker"
)

type Publisher struct {
	writer     *kafka.Writer
	logger     *slog.Logger
	retryCount int
	backoff    time.Duration
	breaker    *gobreaker.CircuitBreaker
}

func NewPublisher(brokers []string, clientID string, retryCount int, backoff time.Duration, failThreshold uint32, logger *slog.Logger) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.Hash{},
			Transport:              &kafka.Transport{ClientID: clientID},
		},
		logger:     logger,
		retryCount: retryCount,
		backoff:    backoff,
		breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "kafka-publisher",
			MaxRequests: 3,
			Interval:    30 * time.Second,
			Timeout:     15 * time.Second,
			ReadyToTrip: func(c gobreaker.Counts) bool { return c.ConsecutiveFailures >= failThreshold },
		}),
	}
}

func (p *Publisher) Publish(ctx context.Context, topic, key string, payload []byte) error {
	_, err := p.breaker.Execute(func() (any, error) {
		return nil, resilience.Retry(ctx, p.retryCount, p.backoff, func(ctx context.Context) error {
			return p.writer.WriteMessages(ctx, kafka.Message{Topic: topic, Key: []byte(key), Value: payload, Time: time.Now().UTC()})
		})
	})
	if err != nil {
		return fmt.Errorf("publish kafka: %w", err)
	}
	return nil
}

func (p *Publisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
