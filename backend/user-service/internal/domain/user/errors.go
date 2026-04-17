package user

import "errors"

var (
	ErrNotFound        = errors.New("user: not found")
	ErrInvalidArgument = errors.New("user: invalid argument")
)
