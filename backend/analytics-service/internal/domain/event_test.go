package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid",
			event: Event{EventID: "evt", ProfileID: "p1", ActorID: "a1", Type: EventView, OccurredAt: time.Now()},
		},
		{
			name: "invalid type",
			event: Event{EventID: "evt", ProfileID: "p1", ActorID: "a1", Type: "bad", OccurredAt: time.Now()},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBucketNormalize(t *testing.T) {
	ts := time.Date(2026, 4, 18, 11, 45, 11, 1, time.UTC)
	require.Equal(t, time.Date(2026, 4, 18, 11, 45, 0, 0, time.UTC), BucketMinute.Normalize(ts))
	require.Equal(t, time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC), BucketHour.Normalize(ts))
	require.Equal(t, time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC), BucketDay.Normalize(ts))
}
