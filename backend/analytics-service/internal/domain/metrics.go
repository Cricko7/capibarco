package domain

import "time"

type ProfileMetric struct {
	ProfileID string
	Bucket    time.Time
	Size      BucketSize
	Counters  map[EventType]int64
}

type ExtendedStats struct {
	ProfileID      string
	Views          int64
	Impressions    int64
	CTR            float64
	CardOpens      int64
	ChatStarts     int64
	Donations      int64
	Boosts         int64
	ProfileChanges int64
	From           time.Time
	To             time.Time
}

type RankingFeedback struct {
	ProfileID      string
	Bucket         time.Time
	CTR            float64
	SwipeRightRate float64
	Engagement     float64
}
