package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidEventType = errors.New("invalid event type")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrForbidden        = errors.New("forbidden")
)

type EventType string

const (
	EventView          EventType = "view"
	EventImpression    EventType = "feed_impression"
	EventSwipeLeft     EventType = "swipe_left"
	EventSwipeRight    EventType = "swipe_right"
	EventCardOpen      EventType = "card_open"
	EventCTRClick      EventType = "ctr_click"
	EventChatStart     EventType = "chat_start"
	EventDonation      EventType = "donation"
	EventBoost         EventType = "boost"
	EventProfileChange EventType = "profile_change"
)

type BucketSize string

const (
	BucketMinute BucketSize = "minute"
	BucketHour   BucketSize = "hour"
	BucketDay    BucketSize = "day"
)

type Event struct {
	EventID    string
	ProfileID  string
	ActorID    string
	Type       EventType
	OccurredAt time.Time
	Metadata   map[string]string
}

func (e Event) Validate() error {
	if e.EventID == "" || e.ProfileID == "" || e.ActorID == "" {
		return fmt.Errorf("missing required ids: %w", ErrInvalidEventType)
	}
	if _, ok := validEventTypes[e.Type]; !ok {
		return fmt.Errorf("type %q: %w", e.Type, ErrInvalidEventType)
	}
	if e.OccurredAt.IsZero() {
		return fmt.Errorf("occurred_at missing: %w", ErrInvalidTimestamp)
	}
	return nil
}

func (b BucketSize) Normalize(ts time.Time) time.Time {
	switch b {
	case BucketMinute:
		return ts.UTC().Truncate(time.Minute)
	case BucketHour:
		return ts.UTC().Truncate(time.Hour)
	default:
		utc := ts.UTC()
		return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func (b BucketSize) IsValid() bool {
	_, ok := validBucketSizes[b]
	return ok
}

type ExtendedStatsRole string

const (
	RoleOwner   ExtendedStatsRole = "owner"
	RoleShelter ExtendedStatsRole = "shelter"
)

func (r ExtendedStatsRole) IsEntitled() bool {
	return r == RoleOwner || r == RoleShelter
}

var validEventTypes = map[EventType]struct{}{
	EventView: {}, EventImpression: {}, EventSwipeLeft: {}, EventSwipeRight: {}, EventCardOpen: {},
	EventCTRClick: {}, EventChatStart: {}, EventDonation: {}, EventBoost: {}, EventProfileChange: {},
}

var validBucketSizes = map[BucketSize]struct{}{
	BucketMinute: {}, BucketHour: {}, BucketDay: {},
}
