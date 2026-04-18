package notification

import "errors"

var (
	ErrNotFound        = errors.New("notification resource not found")
	ErrInvalidArgument = errors.New("invalid argument")
)
