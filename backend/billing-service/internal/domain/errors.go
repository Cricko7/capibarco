package domain

import "errors"

var (
	ErrValidation          = errors.New("validation failed")
	ErrInvalidMoney        = errors.New("invalid money")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrPaymentNotSucceeded = errors.New("payment is not succeeded")
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("conflict")
	ErrArchivedAnimal      = errors.New("animal profile is archived")
)
