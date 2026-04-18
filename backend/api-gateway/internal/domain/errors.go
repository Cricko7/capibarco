// Package domain contains gateway domain primitives that do not depend on delivery or infra.
package domain

import "errors"

var (
	// ErrInvalidGuestSession is returned when an opaque guest token cannot be verified.
	ErrInvalidGuestSession = errors.New("invalid guest session")
	// ErrGuestSessionExpired is returned when a guest token is valid but expired.
	ErrGuestSessionExpired = errors.New("guest session expired")
)
