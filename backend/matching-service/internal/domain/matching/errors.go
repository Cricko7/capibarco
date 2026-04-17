// Package matching contains the core domain model for swipes and matches.
package matching

import "errors"

var (
	// ErrInvalidArgument indicates malformed input that cannot be accepted.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrNotFound indicates that the requested swipe or match does not exist.
	ErrNotFound = errors.New("not found")

	// ErrDuplicateSwipe indicates that an actor has already swiped this animal.
	ErrDuplicateSwipe = errors.New("duplicate swipe")

	// ErrUnavailableAnimal indicates that matching is blocked for this animal.
	ErrUnavailableAnimal = errors.New("animal unavailable")

	// ErrConflict indicates a state conflict that the caller may resolve.
	ErrConflict = errors.New("conflict")
)
