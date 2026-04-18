package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Publisher struct {
	writer *kafka.Writer
	topic  string
}

var _ application.FeedbackPublisher = (*Publisher)(nil)

func NewPublisher(brokers []string, topic string, clientID string, writeTimeout time.Duration) *Publisher {
	return &Publisher{
		topic: topic,
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
			Balancer:               &kafka.Hash{},
			Transport:              &kafka.Transport{ClientID: clientID},
			WriteTimeout:           writeTimeout,
		},
	}
}

func (p *Publisher) PublishRankingFeedback(ctx context.Context, items []domain.RankingFeedback) error {
	if len(items) == 0 {
		return nil
	}
	messages := make([]kafka.Message, 0, len(items))
	for _, item := range items {
		payload, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal ranking feedback: %w", err)
		}
		messages = append(messages, kafka.Message{Topic: p.topic, Key: []byte(item.ProfileID), Value: payload, Time: time.Now().UTC()})
	}
	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("write ranking feedback to kafka: %w", err)
	}
	return nil
}

func (p *Publisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
