package audit

import (
	"context"
	"log/slog"

	"github.com/hackathon/authsvc/internal/domain"
)

// SlogLogger writes audit events as structured logs.
type SlogLogger struct {
	logger *slog.Logger
}

// NewSlogLogger creates an audit logger.
func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{logger: logger}
}

// Log writes an audit event.
func (l *SlogLogger) Log(ctx context.Context, event domain.AuditEvent) error {
	l.logger.InfoContext(ctx, "audit",
		slog.String("tenant_id", event.TenantID),
		slog.String("user_id", event.UserID),
		slog.String("action", event.Action),
		slog.String("outcome", event.Outcome),
		slog.String("ip", event.IP),
		slog.Any("metadata", event.Metadata),
	)
	return nil
}
