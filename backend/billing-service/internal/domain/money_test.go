package domain_test

import (
	"testing"

	"github.com/petmatch/petmatch/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestNewMoneyValidatesPositiveISOAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    string
		units   int64
		nanos   int32
		wantErr error
	}{
		{name: "valid whole amount", code: "USD", units: 10, nanos: 0},
		{name: "valid fractional amount", code: "RUB", units: 1, nanos: 500_000_000},
		{name: "lowercase currency", code: "usd", units: 10, nanos: 0, wantErr: domain.ErrInvalidMoney},
		{name: "empty currency", code: "", units: 10, nanos: 0, wantErr: domain.ErrInvalidMoney},
		{name: "zero amount", code: "USD", units: 0, nanos: 0, wantErr: domain.ErrInvalidMoney},
		{name: "negative units", code: "USD", units: -1, nanos: 0, wantErr: domain.ErrInvalidMoney},
		{name: "negative nanos", code: "USD", units: 1, nanos: -1, wantErr: domain.ErrInvalidMoney},
		{name: "too many nanos", code: "USD", units: 1, nanos: 1_000_000_000, wantErr: domain.ErrInvalidMoney},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.NewMoney(tt.code, tt.units, tt.nanos)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.code, got.CurrencyCode)
			require.Equal(t, tt.units, got.Units)
			require.Equal(t, tt.nanos, got.Nanos)
		})
	}
}
