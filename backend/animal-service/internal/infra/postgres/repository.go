// Package postgres contains PostgreSQL adapters.
package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	domain "github.com/petmatch/petmatch/internal/domain/animal"
)

const maxPageSize int32 = 100

// Repository persists animal profiles in PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a PostgreSQL animal repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Ping verifies PostgreSQL connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	if err := r.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// Create inserts a profile or returns the profile linked to idempotencyKey.
func (r *Repository) Create(ctx context.Context, profile domain.Profile, idempotencyKey string) (domain.Profile, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Profile{}, fmt.Errorf("begin create animal tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if idempotencyKey != "" {
		existing, found, err := r.getByIdempotencyKey(ctx, tx, idempotencyKey)
		if err != nil {
			return domain.Profile{}, err
		}
		if found {
			return existing, nil
		}
	}

	if _, err := tx.Exec(ctx, insertAnimalSQL, profileArgs(profile)...); err != nil {
		return domain.Profile{}, mapPostgresError(err)
	}
	if idempotencyKey != "" {
		if _, err := tx.Exec(ctx, `INSERT INTO idempotency_keys (key, animal_id) VALUES ($1, $2)`, idempotencyKey, profile.ID); err != nil {
			return domain.Profile{}, mapPostgresError(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Profile{}, fmt.Errorf("commit create animal tx: %w", err)
	}
	return profile, nil
}

// Get loads an animal profile by ID.
func (r *Repository) Get(ctx context.Context, id string) (domain.Profile, error) {
	profile, err := scanProfile(r.pool.QueryRow(ctx, selectAnimalSQL+` WHERE animal_id = $1`, id))
	if err != nil {
		return domain.Profile{}, mapPostgresError(err)
	}
	return profile, nil
}

// BatchGet loads profiles in the requested ID set.
func (r *Repository) BatchGet(ctx context.Context, ids []string) ([]domain.Profile, error) {
	rows, err := r.pool.Query(ctx, selectAnimalSQL+` WHERE animal_id = ANY($1::text[])`, ids)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	defer rows.Close()
	return scanProfiles(rows)
}

// Search returns profiles matching filters with opaque cursor pagination.
func (r *Repository) Search(ctx context.Context, query domain.SearchQuery) (domain.SearchResult, error) {
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	offset, err := decodeCursor(query.PageToken)
	if err != nil {
		return domain.SearchResult{}, err
	}

	where, args := buildSearchWhere(query)
	args = append(args, pageSize+1, offset)
	sql := selectAnimalSQL + where + fmt.Sprintf(" ORDER BY boosted DESC, audit_updated_at DESC, animal_id ASC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return domain.SearchResult{}, mapPostgresError(err)
	}
	defer rows.Close()

	items, err := scanProfiles(rows)
	if err != nil {
		return domain.SearchResult{}, err
	}
	result := domain.SearchResult{Items: items}
	if int32(len(items)) > pageSize {
		result.Items = items[:pageSize]
		result.NextPageToken = encodeCursor(offset + int(pageSize))
	}
	return result, nil
}

// Update stores the whole profile aggregate.
func (r *Repository) Update(ctx context.Context, profile domain.Profile) (domain.Profile, error) {
	args := profileArgs(profile)
	args = append(args, profile.ID)
	commandTag, err := r.pool.Exec(ctx, updateAnimalSQL, args...)
	if err != nil {
		return domain.Profile{}, mapPostgresError(err)
	}
	if commandTag.RowsAffected() == 0 {
		return domain.Profile{}, domain.ErrNotFound
	}
	return profile, nil
}

// RegisterIdempotency links a key to an existing animal.
func (r *Repository) RegisterIdempotency(ctx context.Context, key string, animalID string) error {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO idempotency_keys (key, animal_id) VALUES ($1, $2) ON CONFLICT (key) DO NOTHING`, key, animalID)
	if err != nil {
		return mapPostgresError(err)
	}
	return nil
}

func (r *Repository) getByIdempotencyKey(ctx context.Context, tx pgx.Tx, key string) (domain.Profile, bool, error) {
	profile, err := scanProfile(tx.QueryRow(ctx, selectAnimalSQL+` JOIN idempotency_keys idem ON idem.animal_id = animals.animal_id WHERE idem.key = $1`, key))
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Profile{}, false, nil
	}
	if err != nil {
		return domain.Profile{}, false, err
	}
	return profile, true, nil
}

func buildSearchWhere(query domain.SearchQuery) (string, []any) {
	var clauses []string
	var args []any
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if len(query.Species) > 0 {
		clauses = append(clauses, "species = ANY("+addArg(int32s(query.Species))+`::int[])`)
	}
	if len(query.Breeds) > 0 {
		clauses = append(clauses, "LOWER(breed) = ANY("+addArg(lowerStrings(query.Breeds))+`::text[])`)
	}
	if len(query.Sexes) > 0 {
		clauses = append(clauses, "sex = ANY("+addArg(int32s(query.Sexes))+`::int[])`)
	}
	if len(query.Sizes) > 0 {
		clauses = append(clauses, "size = ANY("+addArg(int32s(query.Sizes))+`::int[])`)
	}
	if query.MinAgeMonths != nil {
		clauses = append(clauses, "age_months >= "+addArg(*query.MinAgeMonths))
	}
	if query.MaxAgeMonths != nil {
		clauses = append(clauses, "age_months <= "+addArg(*query.MaxAgeMonths))
	}
	if query.City != nil && strings.TrimSpace(*query.City) != "" {
		clauses = append(clauses, "LOWER(location->>'city') = LOWER("+addArg(strings.TrimSpace(*query.City))+")")
	}
	if len(query.Statuses) > 0 {
		clauses = append(clauses, "status = ANY("+addArg(int32s(query.Statuses))+`::int[])`)
	} else if query.OwnerProfileID == "" {
		clauses = append(clauses, fmt.Sprintf("status = %d AND visibility = %d", domain.StatusAvailable, domain.VisibilityPublic))
	}
	if len(query.Traits) > 0 {
		clauses = append(clauses, "EXISTS (SELECT 1 FROM jsonb_array_elements_text(traits) AS trait WHERE LOWER(trait) = ANY("+addArg(lowerStrings(query.Traits))+`::text[]))`)
	}
	if query.Vaccinated != nil {
		clauses = append(clauses, "vaccinated = "+addArg(*query.Vaccinated))
	}
	if query.Sterilized != nil {
		clauses = append(clauses, "sterilized = "+addArg(*query.Sterilized))
	}
	if query.BoostedOnly != nil && *query.BoostedOnly {
		clauses = append(clauses, "boosted = true AND (boost_expires_at IS NULL OR boost_expires_at > NOW())")
	}
	if query.OwnerProfileID != "" {
		clauses = append(clauses, "owner_profile_id = "+addArg(query.OwnerProfileID))
	}
	if query.NearLatitude != nil && query.NearLongitude != nil && query.RadiusKM != nil {
		lat := addArg(*query.NearLatitude)
		lon := addArg(*query.NearLongitude)
		radius := addArg(*query.RadiusKM)
		clauses = append(clauses, fmt.Sprintf(`location ? 'latitude' AND location ? 'longitude' AND (
			6371 * acos(least(1, cos(radians(%s)) * cos(radians((location->>'latitude')::double precision)) *
			cos(radians((location->>'longitude')::double precision) - radians(%s)) +
			sin(radians(%s)) * sin(radians((location->>'latitude')::double precision))))
		) <= %s`, lat, lon, lat, radius))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func scanProfiles(rows pgx.Rows) ([]domain.Profile, error) {
	var profiles []domain.Profile
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostgresError(err)
	}
	return profiles, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProfile(row scanner) (domain.Profile, error) {
	var profile domain.Profile
	var age pgtype.Int4
	var boostExpiresAt pgtype.Timestamptz
	var traitsJSON []byte
	var medicalJSON []byte
	var locationJSON []byte
	var photosJSON []byte
	var ownerType int32
	var species int32
	var sex int32
	var size int32
	var status int32
	var visibility int32

	err := row.Scan(
		&profile.ID,
		&profile.OwnerProfileID,
		&ownerType,
		&profile.Name,
		&species,
		&profile.Breed,
		&sex,
		&size,
		&age,
		&profile.Description,
		&traitsJSON,
		&medicalJSON,
		&profile.Vaccinated,
		&profile.Sterilized,
		&status,
		&locationJSON,
		&photosJSON,
		&visibility,
		&profile.Boosted,
		&boostExpiresAt,
		&profile.Audit.CreatedAt,
		&profile.Audit.UpdatedAt,
		&profile.Audit.CreatedBy,
		&profile.Audit.UpdatedBy,
		&profile.DonationCount,
		&profile.InterestCount,
	)
	if err != nil {
		return domain.Profile{}, mapPostgresError(err)
	}
	profile.OwnerType = domain.OwnerType(ownerType)
	profile.Species = domain.Species(species)
	profile.Sex = domain.Sex(sex)
	profile.Size = domain.Size(size)
	profile.Status = domain.Status(status)
	profile.Visibility = domain.Visibility(visibility)
	if age.Valid {
		value := age.Int32
		profile.AgeMonths = &value
	}
	if boostExpiresAt.Valid {
		value := boostExpiresAt.Time
		profile.BoostExpiresAt = &value
	}
	if err := json.Unmarshal(traitsJSON, &profile.Traits); err != nil {
		return domain.Profile{}, fmt.Errorf("decode animal traits: %w", err)
	}
	if err := json.Unmarshal(medicalJSON, &profile.MedicalNotes); err != nil {
		return domain.Profile{}, fmt.Errorf("decode animal medical notes: %w", err)
	}
	if err := json.Unmarshal(locationJSON, &profile.Location); err != nil {
		return domain.Profile{}, fmt.Errorf("decode animal location: %w", err)
	}
	if err := json.Unmarshal(photosJSON, &profile.Photos); err != nil {
		return domain.Profile{}, fmt.Errorf("decode animal photos: %w", err)
	}
	return profile, nil
}

func profileArgs(profile domain.Profile) []any {
	traits := mustJSON(profile.Traits)
	medicalNotes := mustJSON(profile.MedicalNotes)
	location := mustJSON(profile.Location)
	photos := mustJSON(profile.Photos)
	return []any{
		profile.ID,
		profile.OwnerProfileID,
		int32(profile.OwnerType),
		profile.Name,
		int32(profile.Species),
		profile.Breed,
		int32(profile.Sex),
		int32(profile.Size),
		profile.AgeMonths,
		profile.Description,
		traits,
		medicalNotes,
		profile.Vaccinated,
		profile.Sterilized,
		int32(profile.Status),
		location,
		photos,
		int32(profile.Visibility),
		profile.Boosted,
		profile.BoostExpiresAt,
		profile.Audit.CreatedAt,
		profile.Audit.UpdatedAt,
		profile.Audit.CreatedBy,
		profile.Audit.UpdatedBy,
		profile.DonationCount,
		profile.InterestCount,
	}
}

func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("marshal postgres json payload: %v", err))
	}
	return data
}

func mapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.Detail)
		case "23503":
			return fmt.Errorf("%w: foreign key violation: %s", domain.ErrInvalidArgument, pgErr.Detail)
		}
	}
	return err
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid page token", domain.ErrInvalidArgument)
	}
	offset, err := strconv.Atoi(string(raw))
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("%w: invalid page token", domain.ErrInvalidArgument)
	}
	return offset, nil
}

func int32s[T ~int32](values []T) []int32 {
	result := make([]int32, 0, len(values))
	for _, value := range values {
		result = append(result, int32(value))
	}
	return result
}

func lowerStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

const selectAnimalSQL = `
SELECT
	animal_id,
	owner_profile_id,
	owner_type,
	name,
	species,
	breed,
	sex,
	size,
	age_months,
	description,
	traits,
	medical_notes,
	vaccinated,
	sterilized,
	status,
	location,
	photos,
	visibility,
	boosted,
	boost_expires_at,
	audit_created_at,
	audit_updated_at,
	audit_created_by,
	audit_updated_by,
	donation_count,
	interest_count
FROM animals`

const insertAnimalSQL = `
INSERT INTO animals (
	animal_id,
	owner_profile_id,
	owner_type,
	name,
	species,
	breed,
	sex,
	size,
	age_months,
	description,
	traits,
	medical_notes,
	vaccinated,
	sterilized,
	status,
	location,
	photos,
	visibility,
	boosted,
	boost_expires_at,
	audit_created_at,
	audit_updated_at,
	audit_created_by,
	audit_updated_by,
	donation_count,
	interest_count
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
	$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
	$21, $22, $23, $24, $25, $26
)`

const updateAnimalSQL = `
UPDATE animals SET
	animal_id = $1,
	owner_profile_id = $2,
	owner_type = $3,
	name = $4,
	species = $5,
	breed = $6,
	sex = $7,
	size = $8,
	age_months = $9,
	description = $10,
	traits = $11,
	medical_notes = $12,
	vaccinated = $13,
	sterilized = $14,
	status = $15,
	location = $16,
	photos = $17,
	visibility = $18,
	boosted = $19,
	boost_expires_at = $20,
	audit_created_at = $21,
	audit_updated_at = $22,
	audit_created_by = $23,
	audit_updated_by = $24,
	donation_count = $25,
	interest_count = $26
WHERE animal_id = $27`
