package domain_test

import (
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewBoostRequiresSucceededDonationAndValidDuration(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		status   domain.PaymentStatus
		duration time.Duration
		wantErr  error
	}{
		{name: "succeeded donation", status: domain.PaymentSucceeded, duration: 24 * time.Hour},
		{name: "pending donation", status: domain.PaymentPending, duration: 24 * time.Hour, wantErr: domain.ErrPaymentNotSucceeded},
		{name: "zero duration", status: domain.PaymentSucceeded, duration: 0, wantErr: domain.ErrValidation},
		{name: "too long duration", status: domain.PaymentSucceeded, duration: 91 * 24 * time.Hour, wantErr: domain.ErrValidation},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			boost, err := domain.NewBoost(domain.NewBoostParams{
				ID:             "boost_1",
				AnimalID:       "animal_1",
				OwnerProfileID: "profile_1",
				DonationID:     "don_1",
				DonationStatus: tt.status,
				Duration:       tt.duration,
				StartsAt:       now,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.True(t, boost.Active)
			require.Equal(t, now.Add(tt.duration), boost.ExpiresAt)
		})
	}
}

func TestBoostCancelIsIdempotent(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	boost, err := domain.NewBoost(domain.NewBoostParams{
		ID:             "boost_1",
		AnimalID:       "animal_1",
		OwnerProfileID: "profile_1",
		DonationID:     "don_1",
		DonationStatus: domain.PaymentSucceeded,
		Duration:       time.Hour,
		StartsAt:       now,
	})
	require.NoError(t, err)

	boost.Cancel("archived", now.Add(time.Minute))
	boost.Cancel("duplicate", now.Add(2*time.Minute))

	require.False(t, boost.Active)
	require.Equal(t, "archived", boost.CancelReason)
}
