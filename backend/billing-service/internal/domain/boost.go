package domain

import (
	"fmt"
	"strings"
	"time"
)

const MaxBoostDuration = 90 * 24 * time.Hour

type Boost struct {
	ID             string
	AnimalID       string
	OwnerProfileID string
	DonationID     string
	StartsAt       time.Time
	ExpiresAt      time.Time
	Active         bool
	CancelReason   string
	CancelledAt    time.Time
}

type NewBoostParams struct {
	ID             string
	AnimalID       string
	OwnerProfileID string
	DonationID     string
	DonationStatus PaymentStatus
	Duration       time.Duration
	StartsAt       time.Time
}

func NewBoost(params NewBoostParams) (Boost, error) {
	if params.DonationStatus != PaymentSucceeded {
		return Boost{}, fmt.Errorf("%w: boost requires succeeded donation", ErrPaymentNotSucceeded)
	}
	if strings.TrimSpace(params.ID) == "" || strings.TrimSpace(params.AnimalID) == "" ||
		strings.TrimSpace(params.OwnerProfileID) == "" || strings.TrimSpace(params.DonationID) == "" {
		return Boost{}, fmt.Errorf("%w: boost id, animal id, owner profile id, and donation id are required", ErrValidation)
	}
	if params.Duration <= 0 || params.Duration > MaxBoostDuration {
		return Boost{}, fmt.Errorf("%w: boost duration must be between 1ns and 90 days", ErrValidation)
	}
	if params.StartsAt.IsZero() {
		return Boost{}, fmt.Errorf("%w: boost start time is required", ErrValidation)
	}
	return Boost{
		ID:             params.ID,
		AnimalID:       params.AnimalID,
		OwnerProfileID: params.OwnerProfileID,
		DonationID:     params.DonationID,
		StartsAt:       params.StartsAt,
		ExpiresAt:      params.StartsAt.Add(params.Duration),
		Active:         true,
	}, nil
}

func (b *Boost) Cancel(reason string, at time.Time) {
	if !b.Active {
		return
	}
	b.Active = false
	b.CancelReason = reason
	b.CancelledAt = at
}
