package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGuestSessionCodecRoundTrip(t *testing.T) {
	codec := NewGuestSessionCodec([]byte("test-secret"), time.Hour)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	token, session, err := codec.Create("device-1", "ru-RU", now)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, "device-1", session.DeviceID)
	require.Equal(t, "ru-RU", session.Locale)

	parsed, err := codec.Parse(token, now.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, session.ID, parsed.ID)
	require.Equal(t, session.ActorID, parsed.ActorID)
	require.Equal(t, []string{"feed:read", "animal:read", "swipe:create"}, parsed.AllowedScopes)
}

func TestGuestSessionCodecRejectsTamperedToken(t *testing.T) {
	codec := NewGuestSessionCodec([]byte("test-secret"), time.Hour)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	token, _, err := codec.Create("device-1", "ru-RU", now)
	require.NoError(t, err)

	_, err = codec.Parse(token[:len(token)-1]+"x", now)
	require.ErrorIs(t, err, ErrInvalidGuestSession)
}

func TestGuestSessionCodecRejectsExpiredToken(t *testing.T) {
	codec := NewGuestSessionCodec([]byte("test-secret"), time.Minute)
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	token, _, err := codec.Create("device-1", "ru-RU", now)
	require.NoError(t, err)

	_, err = codec.Parse(token, now.Add(2*time.Minute))
	require.ErrorIs(t, err, ErrGuestSessionExpired)
}
