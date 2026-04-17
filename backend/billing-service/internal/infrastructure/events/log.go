package events

import (
	"context"
	"log/slog"

	"github.com/petmatch/petmatch/internal/application"
)

type LoggingPublisher struct {
	logger *slog.Logger
}

func NewLoggingPublisher(logger *slog.Logger) *LoggingPublisher {
	return &LoggingPublisher{logger: logger}
}

func (p *LoggingPublisher) Publish(ctx context.Context, event application.BillingEvent) error {
	p.logger.InfoContext(ctx, "billing event published",
		slog.String("topic", event.Topic),
		slog.String("partition_key", event.PartitionKey),
		slog.String("event_type", event.Type),
		slog.String("trace_id", event.TraceID),
		slog.String("correlation_id", event.CorrelationID),
	)
	return nil
}
