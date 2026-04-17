package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	app "github.com/petmatch/petmatch/internal/app/user"
	domain "github.com/petmatch/petmatch/internal/domain/user"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository   { return &Repository{pool: pool} }
func (r *Repository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

func (r *Repository) GetProfile(ctx context.Context, id string) (domain.Profile, error) {
	const q = `SELECT profile_id, auth_user_id, profile_type, display_name, bio, avatar_url, city, visibility, created_at, updated_at FROM user_profiles WHERE profile_id=$1`
	var p domain.Profile
	if err := r.pool.QueryRow(ctx, q, id).Scan(&p.ID, &p.AuthUserID, &p.ProfileType, &p.DisplayName, &p.Bio, &p.AvatarURL, &p.City, &p.Visibility, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Profile{}, domain.ErrNotFound
		}
		return domain.Profile{}, fmt.Errorf("query profile: %w", err)
	}
	return p, nil
}
func (r *Repository) BatchGetProfiles(ctx context.Context, ids []string) ([]domain.Profile, error) {
	rows, err := r.pool.Query(ctx, `SELECT profile_id, auth_user_id, profile_type, display_name, bio, avatar_url, city, visibility, created_at, updated_at FROM user_profiles WHERE profile_id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []domain.Profile{}
	for rows.Next() {
		var p domain.Profile
		if err := rows.Scan(&p.ID, &p.AuthUserID, &p.ProfileType, &p.DisplayName, &p.Bio, &p.AvatarURL, &p.City, &p.Visibility, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, p)
	}
	return res, rows.Err()
}
func (r *Repository) SearchProfiles(ctx context.Context, q app.SearchProfilesQuery) ([]domain.Profile, string, error) {
	limit := 20
	if q.Page.PageSize > 0 {
		limit = int(q.Page.PageSize)
	}
	offset := 0
	if q.Page.PageToken != "" {
		o, _ := strconv.Atoi(q.Page.PageToken)
		offset = o
	}
	rows, err := r.pool.Query(ctx, `SELECT profile_id, auth_user_id, profile_type, display_name, bio, avatar_url, city, visibility, created_at, updated_at
FROM user_profiles
WHERE ($1='' OR city=$1) AND ($2='' OR display_name ILIKE '%' || $2 || '%')
ORDER BY updated_at DESC LIMIT $3 OFFSET $4`, q.City, q.Query, limit+1, offset)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	profiles := []domain.Profile{}
	for rows.Next() {
		var p domain.Profile
		if err := rows.Scan(&p.ID, &p.AuthUserID, &p.ProfileType, &p.DisplayName, &p.Bio, &p.AvatarURL, &p.City, &p.Visibility, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, "", err
		}
		profiles = append(profiles, p)
	}
	next := ""
	if len(profiles) > limit {
		profiles = profiles[:limit]
		next = strconv.Itoa(offset + limit)
	}
	return profiles, next, rows.Err()
}
func (r *Repository) UpdateProfile(ctx context.Context, p domain.Profile, _ []string) (domain.Profile, error) {
	const q = `INSERT INTO user_profiles(profile_id, auth_user_id, profile_type, display_name, bio, avatar_url, city, visibility, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8, COALESCE((SELECT created_at FROM user_profiles WHERE profile_id=$1), now()),$9)
ON CONFLICT(profile_id) DO UPDATE SET auth_user_id=EXCLUDED.auth_user_id, profile_type=EXCLUDED.profile_type, display_name=EXCLUDED.display_name, bio=EXCLUDED.bio, avatar_url=EXCLUDED.avatar_url, city=EXCLUDED.city, visibility=EXCLUDED.visibility, updated_at=EXCLUDED.updated_at
RETURNING created_at, updated_at`
	if err := r.pool.QueryRow(ctx, q, p.ID, p.AuthUserID, p.ProfileType, p.DisplayName, p.Bio, p.AvatarURL, p.City, p.Visibility, p.UpdatedAt).Scan(&p.CreatedAt, &p.UpdatedAt); err != nil {
		return domain.Profile{}, err
	}
	return p, nil
}
func (r *Repository) CreateReview(ctx context.Context, rv domain.Review) (domain.Review, error) {
	const q = `INSERT INTO user_reviews(review_id,target_profile_id,author_profile_id,rating,text,match_id,visibility,created_at,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := r.pool.Exec(ctx, q, rv.ID, rv.TargetProfileID, rv.AuthorProfileID, rv.Rating, rv.Text, nullable(rv.MatchID), rv.Visibility, rv.CreatedAt, rv.UpdatedAt)
	if err != nil {
		return domain.Review{}, err
	}
	return rv, nil
}
func (r *Repository) UpdateReview(ctx context.Context, rv domain.Review, _ []string) (domain.Review, error) {
	_, err := r.pool.Exec(ctx, `UPDATE user_reviews SET rating=$1,text=$2,visibility=$3,updated_at=$4 WHERE review_id=$5`, rv.Rating, rv.Text, rv.Visibility, rv.UpdatedAt, rv.ID)
	if err != nil {
		return domain.Review{}, err
	}
	return rv, nil
}
func (r *Repository) ListReviews(ctx context.Context, targetProfileID string, page app.PageRequest) ([]domain.Review, string, error) {
	limit := 20
	if page.PageSize > 0 {
		limit = int(page.PageSize)
	}
	offset := 0
	if page.PageToken != "" {
		o, _ := strconv.Atoi(page.PageToken)
		offset = o
	}
	rows, err := r.pool.Query(ctx, `SELECT review_id,target_profile_id,author_profile_id,rating,text,COALESCE(match_id,''),visibility,created_at,updated_at FROM user_reviews WHERE target_profile_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, targetProfileID, limit+1, offset)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := []domain.Review{}
	for rows.Next() {
		var rv domain.Review
		if err := rows.Scan(&rv.ID, &rv.TargetProfileID, &rv.AuthorProfileID, &rv.Rating, &rv.Text, &rv.MatchID, &rv.Visibility, &rv.CreatedAt, &rv.UpdatedAt); err != nil {
			return nil, "", err
		}
		out = append(out, rv)
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = strconv.Itoa(offset + limit)
	}
	return out, next, rows.Err()
}
func (r *Repository) GetReputation(ctx context.Context, profileID string) (domain.Reputation, error) {
	var rep domain.Reputation
	err := r.pool.QueryRow(ctx, `SELECT target_profile_id, COALESCE(AVG(rating),0)::float8, COUNT(*), COALESCE(MAX(updated_at), now()) FROM user_reviews WHERE target_profile_id=$1 GROUP BY target_profile_id`, profileID).Scan(&rep.ProfileID, &rep.AverageRating, &rep.ReviewsCount, &rep.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Reputation{ProfileID: profileID}, nil
		}
		return domain.Reputation{}, err
	}
	return rep, nil
}

func nullable(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
