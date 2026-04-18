// Package gateway contains application orchestration for api-gateway.
package gateway

import "errors"

var (
	// ErrUnauthenticated means the request does not have a valid actor.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrPermissionDenied means the actor is valid but cannot perform an operation.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrIdempotencyKeyRequired means a mutating request omitted its idempotency key.
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	// ErrInvalidInput means a request failed application validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrDependencyDisabled means a future downstream service is not enabled in this environment.
	ErrDependencyDisabled = errors.New("dependency disabled")
	// ErrRateLimited means traffic exceeded gateway limits.
	ErrRateLimited = errors.New("rate limit exceeded")
)
