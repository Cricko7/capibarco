package chat

import "errors"

var (
	// ErrNotFound means the requested chat resource does not exist.
	ErrNotFound = errors.New("chat resource not found")
	// ErrForbidden means the actor is not allowed to perform the operation.
	ErrForbidden = errors.New("chat operation forbidden")
	// ErrInvalidParticipant means conversation participants are invalid.
	ErrInvalidParticipant = errors.New("invalid chat participant")
	// ErrMissingIdempotencyKey means a mutating request lacks an idempotency key.
	ErrMissingIdempotencyKey = errors.New("missing idempotency key")
	// ErrInvalidMessage means message content does not satisfy domain rules.
	ErrInvalidMessage = errors.New("invalid chat message")
	// ErrConversationClosed means messages cannot be sent to the conversation.
	ErrConversationClosed = errors.New("conversation is not active")
)
