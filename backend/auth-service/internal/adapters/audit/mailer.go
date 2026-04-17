package audit

import (
	"context"
	"log/slog"
)

// LogMailer logs reset tokens for development; replace with SMTP/provider adapter in production.
type LogMailer struct {
	logger *slog.Logger
}

// NewLogMailer creates a development mailer.
func NewLogMailer(logger *slog.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

// SendPasswordReset logs the reset token.
func (m *LogMailer) SendPasswordReset(ctx context.Context, tenantID string, email string, resetToken string) error {
	m.logger.WarnContext(ctx, "password reset token generated",
		slog.String("tenant_id", tenantID),
		slog.String("email", email),
		slog.String("reset_token", resetToken),
	)
	return nil
}
