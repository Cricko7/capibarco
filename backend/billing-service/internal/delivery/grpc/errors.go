package grpcdelivery

import (
	"errors"

	"github.com/petmatch/petmatch/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toStatus(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrInvalidMoney), errors.Is(err, domain.ErrInvalidTransition),
		errors.Is(err, domain.ErrPaymentNotSucceeded), errors.Is(err, domain.ErrArchivedAnimal):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal billing error")
	}
}
