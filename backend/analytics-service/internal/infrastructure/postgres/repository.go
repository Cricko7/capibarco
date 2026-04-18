package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/petmatch/petmatch/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Repository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}
	return &Repository{pool: pool}, nil
}

func (r *Repository) Close() { r.pool.Close() }

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *Repository) IngestRawEvent(ctx context.Context, event domain.Event) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `
		INSERT INTO raw_events(event_id, profile_id, actor_id, event_type, occurred_at, metadata)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (event_id) DO NOTHING`,
		event.EventID, event.ProfileID, event.ActorID, string(event.Type), event.OccurredAt, event.Metadata,
	)
	if err != nil {
		return false, fmt.Errorf("insert raw event: %w", err)
	}
	return cmd.RowsAffected() == 1, nil
}

func (r *Repository) AggregateEvent(ctx context.Context, event domain.Event, bucket domain.BucketSize) error {
	bucketAt := bucket.Normalize(event.OccurredAt)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO aggregated_metrics(profile_id, bucket_size, bucket_at, event_type, value)
		VALUES ($1,$2,$3,$4,1)
		ON CONFLICT(profile_id, bucket_size, bucket_at, event_type)
		DO UPDATE SET value = aggregated_metrics.value + 1`,
		event.ProfileID, string(bucket), bucketAt, string(event.Type),
	)
	if err != nil {
		return fmt.Errorf("upsert aggregate: %w", err)
	}
	return nil
}

func (r *Repository) MetricsByBucket(ctx context.Context, profileID string, from, to time.Time, bucket domain.BucketSize) ([]domain.ProfileMetric, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT bucket_at, event_type, value
		FROM aggregated_metrics
		WHERE profile_id=$1 AND bucket_size=$2 AND bucket_at BETWEEN $3 AND $4
		ORDER BY bucket_at`, profileID, string(bucket), bucket.Normalize(from), bucket.Normalize(to))
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	group := map[time.Time]map[domain.EventType]int64{}
	for rows.Next() {
		var b time.Time
		var et string
		var value int64
		if err := rows.Scan(&b, &et, &value); err != nil {
			return nil, fmt.Errorf("scan metric row: %w", err)
		}
		if _, ok := group[b]; !ok {
			group[b] = map[domain.EventType]int64{}
		}
		group[b][domain.EventType(et)] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric rows: %w", err)
	}

	result := make([]domain.ProfileMetric, 0, len(group))
	for bucketAt, counters := range group {
		result = append(result, domain.ProfileMetric{ProfileID: profileID, Bucket: bucketAt, Size: bucket, Counters: counters})
	}
	return result, nil
}

func (r *Repository) ExtendedStats(ctx context.Context, profileID string, from, to time.Time) (domain.ExtendedStats, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT event_type, COALESCE(SUM(value),0)
		FROM aggregated_metrics
		WHERE profile_id=$1 AND bucket_size='day' AND bucket_at BETWEEN $2 AND $3
		GROUP BY event_type`, profileID, domain.BucketDay.Normalize(from), domain.BucketDay.Normalize(to))
	if err != nil {
		return domain.ExtendedStats{}, fmt.Errorf("query extended stats: %w", err)
	}
	defer rows.Close()

	stats := domain.ExtendedStats{ProfileID: profileID, From: from, To: to}
	for rows.Next() {
		var et string
		var value int64
		if err := rows.Scan(&et, &value); err != nil {
			return domain.ExtendedStats{}, fmt.Errorf("scan stats row: %w", err)
		}
		switch domain.EventType(et) {
		case domain.EventView:
			stats.Views = value
		case domain.EventImpression:
			stats.Impressions = value
		case domain.EventCardOpen:
			stats.CardOpens = value
		case domain.EventChatStart:
			stats.ChatStarts = value
		case domain.EventDonation:
			stats.Donations = value
		case domain.EventBoost:
			stats.Boosts = value
		case domain.EventProfileChange:
			stats.ProfileChanges = value
		}
	}
	if err := rows.Err(); err != nil {
		return domain.ExtendedStats{}, fmt.Errorf("iterate stats rows: %w", err)
	}
	if stats.Impressions > 0 {
		stats.CTR = float64(stats.Views) / float64(stats.Impressions)
	}
	return stats, nil
}

func (r *Repository) RankingFeedback(ctx context.Context, from, to time.Time, limit int) ([]domain.RankingFeedback, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT profile_id, bucket_at,
		COALESCE(SUM(CASE WHEN event_type='view' THEN value ELSE 0 END),0) AS views,
		COALESCE(SUM(CASE WHEN event_type='feed_impression' THEN value ELSE 0 END),0) AS impressions,
		COALESCE(SUM(CASE WHEN event_type='swipe_right' THEN value ELSE 0 END),0) AS swipe_right,
		COALESCE(SUM(CASE WHEN event_type='swipe_left' THEN value ELSE 0 END),0) AS swipe_left,
		COALESCE(SUM(CASE WHEN event_type IN ('chat_start','donation','boost') THEN value ELSE 0 END),0) AS engagement
		FROM aggregated_metrics
		WHERE bucket_size='hour' AND bucket_at BETWEEN $1 AND $2
		GROUP BY profile_id, bucket_at
		ORDER BY engagement DESC
		LIMIT $3`, domain.BucketHour.Normalize(from), domain.BucketHour.Normalize(to), limit)
	if err != nil {
		return nil, fmt.Errorf("query ranking feedback: %w", err)
	}
	defer rows.Close()

	items := make([]domain.RankingFeedback, 0, limit)
	for rows.Next() {
		var (
			id           string
			bucket       time.Time
			views        int64
			impressions  int64
			swipeRight   int64
			swipeLeft    int64
			engagementCT int64
		)
		if err := rows.Scan(&id, &bucket, &views, &impressions, &swipeRight, &swipeLeft, &engagementCT); err != nil {
			return nil, fmt.Errorf("scan ranking row: %w", err)
		}
		item := domain.RankingFeedback{ProfileID: id, Bucket: bucket, Engagement: float64(engagementCT)}
		if impressions > 0 {
			item.CTR = float64(views) / float64(impressions)
		}
		if swipeRight+swipeLeft > 0 {
			item.SwipeRightRate = float64(swipeRight) / float64(swipeRight+swipeLeft)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ranking rows: %w", err)
	}
	return items, nil
}

func IsDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
