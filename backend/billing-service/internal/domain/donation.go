package domain

import (
	"fmt"
	"strings"
	"time"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentSucceeded PaymentStatus = "succeeded"
	PaymentFailed    PaymentStatus = "failed"
	PaymentCancelled PaymentStatus = "cancelled"
	PaymentRefunded  PaymentStatus = "refunded"
)

type DonationTargetType string

const (
	DonationTargetUnspecified DonationTargetType = ""
	DonationTargetShelter     DonationTargetType = "shelter"
	DonationTargetAnimal      DonationTargetType = "animal"
)

type Donation struct {
	ID                string
	PayerProfileID    string
	TargetType        DonationTargetType
	TargetID          string
	Amount            Money
	Status            PaymentStatus
	Provider          string
	ProviderPaymentID string
	FailureReason     string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type NewDonationParams struct {
	ID             string
	PayerProfileID string
	TargetType     DonationTargetType
	TargetID       string
	Amount         Money
	Provider       string
	CreatedAt      time.Time
}

func NewDonation(params NewDonationParams) (Donation, error) {
	if strings.TrimSpace(params.ID) == "" {
		return Donation{}, fmt.Errorf("%w: donation id is required", ErrValidation)
	}
	if strings.TrimSpace(params.PayerProfileID) == "" {
		return Donation{}, fmt.Errorf("%w: payer profile id is required", ErrValidation)
	}
	if params.TargetType != DonationTargetShelter && params.TargetType != DonationTargetAnimal {
		return Donation{}, fmt.Errorf("%w: target type is required", ErrValidation)
	}
	if strings.TrimSpace(params.TargetID) == "" {
		return Donation{}, fmt.Errorf("%w: target id is required", ErrValidation)
	}
	if strings.TrimSpace(params.Provider) == "" {
		return Donation{}, fmt.Errorf("%w: provider is required", ErrValidation)
	}
	if params.CreatedAt.IsZero() {
		return Donation{}, fmt.Errorf("%w: created at is required", ErrValidation)
	}

	return Donation{
		ID:             params.ID,
		PayerProfileID: params.PayerProfileID,
		TargetType:     params.TargetType,
		TargetID:       params.TargetID,
		Amount:         params.Amount,
		Status:         PaymentPending,
		Provider:       params.Provider,
		CreatedAt:      params.CreatedAt,
		UpdatedAt:      params.CreatedAt,
	}, nil
}

func (d *Donation) MarkSucceeded(providerPaymentID string, at time.Time) error {
	if d.Status == PaymentSucceeded {
		return nil
	}
	if d.Status != PaymentPending {
		return fmt.Errorf("%w: cannot mark %s donation succeeded", ErrInvalidTransition, d.Status)
	}
	if strings.TrimSpace(providerPaymentID) == "" {
		return fmt.Errorf("%w: provider payment id is required", ErrValidation)
	}
	d.Status = PaymentSucceeded
	d.ProviderPaymentID = providerPaymentID
	d.UpdatedAt = at
	return nil
}

func (d *Donation) MarkFailed(reason string, at time.Time) error {
	if d.Status == PaymentFailed {
		return nil
	}
	if d.Status != PaymentPending {
		return fmt.Errorf("%w: cannot mark %s donation failed", ErrInvalidTransition, d.Status)
	}
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: failure reason is required", ErrValidation)
	}
	d.Status = PaymentFailed
	d.FailureReason = reason
	d.UpdatedAt = at
	return nil
}
