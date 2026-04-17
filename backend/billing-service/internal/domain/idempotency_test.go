package domain_test

import (
	"testing"

	"github.com/petmatch/petmatch/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestHashIdempotencyKeyDoesNotExposeRawKey(t *testing.T) {
	t.Parallel()

	hash, err := domain.HashIdempotencyKey("CreateDonationIntent", "raw-client-key")

	require.NoError(t, err)
	require.NotContains(t, hash, "raw-client-key")
	require.Len(t, hash, 64)
}

func TestHashIdempotencyKeyRequiresScopeAndKey(t *testing.T) {
	t.Parallel()

	_, err := domain.HashIdempotencyKey("", "raw-client-key")
	require.ErrorIs(t, err, domain.ErrValidation)

	_, err = domain.HashIdempotencyKey("CreateDonationIntent", "")
	require.ErrorIs(t, err, domain.ErrValidation)
}
