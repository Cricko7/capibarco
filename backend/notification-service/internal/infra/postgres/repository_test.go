package postgres

import (
	"testing"
	"time"

	domain "github.com/petmatch/petmatch/internal/domain/notification"
)

type fakeRow struct{}

func (fakeRow) Scan(dest ...any) error {
	createdAt := time.Date(2026, 4, 18, 20, 0, 0, 0, time.UTC)
	readAt := createdAt.Add(5 * time.Minute)

	*(dest[0].(*string)) = "notification-1"
	*(dest[1].(*string)) = "profile-1"
	*(dest[2].(*int16)) = int16(domain.TypeMatchCreated)
	*(dest[3].(*[]int16)) = []int16{
		int16(domain.ChannelPush),
		int16(domain.ChannelInApp),
	}
	*(dest[4].(*string)) = "New match"
	*(dest[5].(*string)) = "You have a new match"
	*(dest[6].(*[]byte)) = []byte(`{"match_id":"match-1"}`)
	*(dest[7].(*int16)) = int16(domain.StatusDelivered)
	*(dest[8].(**time.Time)) = &readAt
	*(dest[9].(*time.Time)) = createdAt
	*(dest[10].(*string)) = "idem-1"
	return nil
}

func TestScanNotificationConvertsPrimitiveDatabaseValues(t *testing.T) {
	notification, err := scanNotification(fakeRow{})
	if err != nil {
		t.Fatalf("scanNotification() error = %v", err)
	}

	if notification.ID != "notification-1" {
		t.Fatalf("notification.ID = %q, want notification-1", notification.ID)
	}
	if notification.Type != domain.TypeMatchCreated {
		t.Fatalf("notification.Type = %v, want %v", notification.Type, domain.TypeMatchCreated)
	}
	if notification.Status != domain.StatusDelivered {
		t.Fatalf("notification.Status = %v, want %v", notification.Status, domain.StatusDelivered)
	}
	if len(notification.Channels) != 2 || notification.Channels[0] != domain.ChannelPush || notification.Channels[1] != domain.ChannelInApp {
		t.Fatalf("notification.Channels = %v, want [push in_app]", notification.Channels)
	}
	if notification.Data["match_id"] != "match-1" {
		t.Fatalf("notification.Data = %v, want match_id", notification.Data)
	}
	if notification.ReadAt == nil {
		t.Fatalf("notification.ReadAt = nil, want timestamp")
	}
}
