package grpc

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/petmatch/chat-service/internal/domain/chat"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatusError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request cancelled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request deadline exceeded")
	case errors.Is(err, chat.ErrNotFound):
		return status.Error(codes.NotFound, "chat resource not found")
	case errors.Is(err, chat.ErrForbidden):
		return status.Error(codes.PermissionDenied, "chat operation forbidden")
	case errors.Is(err, chat.ErrInvalidParticipant), errors.Is(err, chat.ErrInvalidMessage), errors.Is(err, chat.ErrMissingIdempotencyKey):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, chat.ErrConversationClosed):
		return status.Error(codes.FailedPrecondition, "conversation is not active")
	default:
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Error(codes.Internal, "internal chat service error")
	}
}
