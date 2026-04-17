package domain_test

import (
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestDonationStatusTransitions(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	amount, err := domain.NewMoney("USD", 25, 0)
	require.NoError(t, err)

	donation, err := domain.NewDonation(domain.NewDonationParams{
		ID:             "don_1",
		PayerProfileID: "profile_1",
		TargetType:     domain.DonationTargetAnimal,
		TargetID:       "animal_1",
		Amount:         amount,
		Provider:       "mock",
		CreatedAt:      now,
	})
	require.NoError(t, err)

	require.Equal(t, domain.PaymentPending, donation.Status)

	require.NoError(t, donation.MarkSucceeded("pay_1", now.Add(time.Minute)))
	require.Equal(t, domain.PaymentSucceeded, donation.Status)
	require.Equal(t, "pay_1", donation.ProviderPaymentID)

	err = donation.MarkFailed("late failure", now.Add(2*time.Minute))
	require.ErrorIs(t, err, domain.ErrInvalidTransition)
}

func TestNewDonationRequiresSafeFields(t *testing.T) {
	t.Parallel()

	amount, err := domain.NewMoney("USD", 10, 0)
	require.NoError(t, err)

	tests := []struct {
		name   string
		params domain.NewDonationParams
	}{
		{
			name: "missing payer",
			params: domain.NewDonationParams{
				ID:         "don_1",
				TargetType: domain.DonationTargetShelter,
				TargetID:   "shelter_1",
				Amount:     amount,
				Provider:   "mock",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "unspecified target",
			params: domain.NewDonationParams{
				ID:             "don_1",
				PayerProfileID: "profile_1",
				TargetType:     domain.DonationTargetUnspecified,
				TargetID:       "shelter_1",
				Amount:         amount,
				Provider:       "mock",
				CreatedAt:      time.Now(),
			},
		},
		{
			name: "missing provider",
			params: domain.NewDonationParams{
				ID:             "don_1",
				PayerProfileID: "profile_1",
				TargetType:     domain.DonationTargetShelter,
				TargetID:       "shelter_1",
				Amount:         amount,
				CreatedAt:      time.Now(),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := domain.NewDonation(tt.params)

			require.ErrorIs(t, err, domain.ErrValidation)
		})
	}
}
