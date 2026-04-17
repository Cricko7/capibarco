// Package animal contains the core animal profile domain model.
package animal

import "errors"

var (
	// ErrInvalidArgument is returned when a command or entity violates domain invariants.
	ErrInvalidArgument = errors.New("invalid animal argument")
	// ErrInvalidState is returned when a transition is not allowed from the current state.
	ErrInvalidState = errors.New("invalid animal state")
	// ErrNotFound is returned when an animal resource cannot be found.
	ErrNotFound = errors.New("animal not found")
	// ErrForbidden is returned when an actor does not own a mutable animal profile.
	ErrForbidden = errors.New("animal mutation forbidden")
	// ErrConflict is returned when a command conflicts with existing state.
	ErrConflict = errors.New("animal conflict")
)
