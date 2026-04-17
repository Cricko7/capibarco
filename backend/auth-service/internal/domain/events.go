package domain

import "time"

const (
	EventUserRegistered          = "auth.user_registered"
	EventUserLoggedIn            = "auth.user_logged_in"
	EventTokenRefreshed          = "auth.token_refreshed"
	EventPasswordResetRequested  = "auth.password_reset_requested"
	EventPasswordResetCompleted  = "auth.password_reset_completed"
	EventPermissionDenied        = "auth.permission_denied"
	EventProducerAuthService     = "auth-service"
	EventSchemaVersionAuthEvents = "1"
)

// Event is the JSON-compatible Kafka event envelope used by auth events.
type Event struct {
	ID             string `json:"event_id"`
	Type           string `json:"event_type"`
	SchemaVersion  string `json:"schema_version"`
	OccurredAt     string `json:"occurred_at"`
	Producer       string `json:"producer"`
	TraceID        string `json:"trace_id"`
	CorrelationID  string `json:"correlation_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Key            string `json:"-"`
	Payload        any    `json:"payload"`
}

// EventMeta contains cross-service tracing metadata.
type EventMeta struct {
	TraceID        string
	CorrelationID  string
	IdempotencyKey string
}

// NewEvent builds a Kafka event envelope.
func NewEvent(id string, eventType string, key string, occurredAt time.Time, meta EventMeta, payload any) Event {
	return Event{
		ID:             id,
		Type:           eventType,
		SchemaVersion:  EventSchemaVersionAuthEvents,
		OccurredAt:     occurredAt.UTC().Format(time.RFC3339Nano),
		Producer:       EventProducerAuthService,
		TraceID:        meta.TraceID,
		CorrelationID:  meta.CorrelationID,
		IdempotencyKey: meta.IdempotencyKey,
		Key:            key,
		Payload:        payload,
	}
}

// UserRegisteredPayload is published to auth.user_registered.
type UserRegisteredPayload struct {
	User           EventUser `json:"user"`
	TenantID       string    `json:"tenant_id"`
	Roles          []string  `json:"roles"`
	RegistrationIP string    `json:"registration_ip"`
}

// EventUser matches auth.v1.User JSON shape for auth events.
type EventUser struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// UserLoggedInPayload is published to auth.user_logged_in.
type UserLoggedInPayload struct {
	UserID    string   `json:"user_id"`
	TenantID  string   `json:"tenant_id"`
	Email     string   `json:"email"`
	TokenID   string   `json:"token_id"`
	Roles     []string `json:"roles"`
	IP        string   `json:"ip"`
	UserAgent string   `json:"user_agent"`
}

// TokenRefreshedPayload is published to auth.token_refreshed.
type TokenRefreshedPayload struct {
	UserID     string `json:"user_id"`
	TenantID   string `json:"tenant_id"`
	OldTokenID string `json:"old_token_id"`
	NewTokenID string `json:"new_token_id"`
	ExpiresAt  string `json:"expires_at"`
}

// PasswordResetRequestedPayload is published to auth.password_reset_requested.
type PasswordResetRequestedPayload struct {
	TenantID     string `json:"tenant_id"`
	Email        string `json:"email"`
	ResetTokenID string `json:"reset_token_id"`
	ExpiresAt    string `json:"expires_at"`
	IP           string `json:"ip"`
}

// PasswordResetCompletedPayload is published to auth.password_reset_completed.
type PasswordResetCompletedPayload struct {
	UserID       string `json:"user_id"`
	TenantID     string `json:"tenant_id"`
	Email        string `json:"email"`
	ResetTokenID string `json:"reset_token_id"`
	IP           string `json:"ip"`
}

// PermissionDeniedPayload is published to auth.permission_denied.
type PermissionDeniedPayload struct {
	Subject    string   `json:"subject"`
	TenantID   string   `json:"tenant_id"`
	Permission string   `json:"permission"`
	Roles      []string `json:"roles"`
	TokenID    string   `json:"token_id"`
}
