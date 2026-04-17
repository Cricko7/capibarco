package domain

import (
	"fmt"
	"strings"
	"time"
)

type EntitlementType string

const (
	EntitlementAdvancedFilters     EntitlementType = "advanced_filters"
	EntitlementExtendedAnimalStats EntitlementType = "extended_animal_stats"
	EntitlementAnimalBoost         EntitlementType = "animal_boost"
)

type Entitlement struct {
	ID             string
	OwnerProfileID string
	Type           EntitlementType
	ResourceID     string
	StartsAt       time.Time
	ExpiresAt      time.Time
	Active         bool
}

type NewEntitlementParams struct {
	ID             string
	OwnerProfileID string
	Type           EntitlementType
	ResourceID     string
	Duration       time.Duration
	StartsAt       time.Time
}

func NewEntitlement(params NewEntitlementParams) (Entitlement, error) {
	if strings.TrimSpace(params.ID) == "" || strings.TrimSpace(params.OwnerProfileID) == "" {
		return Entitlement{}, fmt.Errorf("%w: entitlement id and owner profile id are required", ErrValidation)
	}
	if params.Type != EntitlementAdvancedFilters && params.Type != EntitlementExtendedAnimalStats && params.Type != EntitlementAnimalBoost {
		return Entitlement{}, fmt.Errorf("%w: entitlement type is invalid", ErrValidation)
	}
	if params.Duration <= 0 {
		return Entitlement{}, fmt.Errorf("%w: entitlement duration must be positive", ErrValidation)
	}
	if params.StartsAt.IsZero() {
		return Entitlement{}, fmt.Errorf("%w: entitlement start time is required", ErrValidation)
	}
	return Entitlement{
		ID:             params.ID,
		OwnerProfileID: params.OwnerProfileID,
		Type:           params.Type,
		ResourceID:     params.ResourceID,
		StartsAt:       params.StartsAt,
		ExpiresAt:      params.StartsAt.Add(params.Duration),
		Active:         true,
	}, nil
}
