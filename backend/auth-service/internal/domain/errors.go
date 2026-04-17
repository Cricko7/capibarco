package domain

import "errors"

var (
	ErrAlreadyExists       = errors.New("already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrTokenReused         = errors.New("refresh token reused")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrNotFound            = errors.New("not found")
	ErrWeakPassword        = errors.New("weak password")
	ErrTenantRequired      = errors.New("tenant id is required")
	ErrValidation          = errors.New("validation failed")
	ErrResetTokenConsumed  = errors.New("reset token already consumed")
	ErrResetTokenNotActive = errors.New("reset token is not active")
)
