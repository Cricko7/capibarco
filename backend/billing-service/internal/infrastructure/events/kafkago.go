package events

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaGoWriter struct {
	writer *kafka.Writer
}

type KafkaGoConfig struct {
	Brokers      []string
	ClientID     string
	RequiredAcks int
	BatchTimeout time.Duration
	WriteTimeout time.Duration
}

func NewKafkaGoWriter(cfg KafkaGoConfig) (*KafkaGoWriter, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "billing-service"
	}
	if cfg.RequiredAcks == 0 {
		cfg.RequiredAcks = int(kafka.RequireAll)
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 10 * time.Millisecond
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	return &KafkaGoWriter{writer: &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequiredAcks(cfg.RequiredAcks),
		BatchTimeout:           cfg.BatchTimeout,
		WriteTimeout:           cfg.WriteTimeout,
		AllowAutoTopicCreation: false,
		Transport: &kafka.Transport{
			ClientID: cfg.ClientID,
		},
	}}, nil
}

func (w *KafkaGoWriter) WriteMessages(ctx context.Context, messages ...KafkaMessage) error {
	kafkaMessages := make([]kafka.Message, 0, len(messages))
	for _, message := range messages {
		headers := make([]kafka.Header, 0, len(message.Headers))
		for key, value := range message.Headers {
			headers = append(headers, kafka.Header{Key: key, Value: []byte(value)})
		}
		kafkaMessages = append(kafkaMessages, kafka.Message{
			Topic:   message.Topic,
			Key:     message.Key,
			Value:   message.Value,
			Headers: headers,
			Time:    time.Now().UTC(),
		})
	}
	if err := w.writer.WriteMessages(ctx, kafkaMessages...); err != nil {
		return fmt.Errorf("kafka-go write messages: %w", err)
	}
	return nil
}

func (w *KafkaGoWriter) Close() error {
	if w == nil || w.writer == nil {
		return nil
	}
	return w.writer.Close()
}
