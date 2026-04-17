package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/config"
	"github.com/petmatch/petmatch/internal/domain"
)

type txKey struct{}

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, cfg config.PostgresConfig) (*Store, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, cfg.HealthTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("tx rollback after %w: %w", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *Store) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return s.pool
}

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.ConstraintName)
	}
	return err
}

func (s *Store) GetIdempotency(ctx context.Context, scope string, hash string) (application.IdempotencyRecord, error) {
	var r application.IdempotencyRecord
	err := s.q(ctx).QueryRow(ctx, `
		SELECT scope, key_hash, resource_kind, resource_id, COALESCE(related_resource_id, ''), created_at
		FROM idempotency_keys
		WHERE scope = $1 AND key_hash = $2
	`, scope, hash).Scan(&r.Scope, &r.KeyHash, &r.ResourceKind, &r.ResourceID, &r.RelatedResourceID, &r.CreatedAt)
	if err != nil {
		return application.IdempotencyRecord{}, mapErr(err)
	}
	return r, nil
}

func (s *Store) SaveIdempotency(ctx context.Context, record application.IdempotencyRecord) error {
	_, err := s.q(ctx).Exec(ctx, `
		INSERT INTO idempotency_keys(scope, key_hash, resource_kind, resource_id, related_resource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(scope, key_hash) DO NOTHING
	`, record.Scope, record.KeyHash, record.ResourceKind, record.ResourceID, record.RelatedResourceID, record.CreatedAt)
	return mapErr(err)
}

func (s *Store) CreateDonation(ctx context.Context, d domain.Donation) error {
	_, err := s.q(ctx).Exec(ctx, `
		INSERT INTO donations(
			donation_id, payer_profile_id, target_type, target_id, currency_code, units, nanos,
			status, provider, provider_payment_id, failure_reason, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, d.ID, d.PayerProfileID, d.TargetType, d.TargetID, d.Amount.CurrencyCode, d.Amount.Units, d.Amount.Nanos,
		d.Status, d.Provider, d.ProviderPaymentID, d.FailureReason, d.CreatedAt, d.UpdatedAt)
	return mapErr(err)
}

func (s *Store) GetDonation(ctx context.Context, id string) (domain.Donation, error) {
	row := s.q(ctx).QueryRow(ctx, `
		SELECT donation_id, payer_profile_id, target_type, target_id, currency_code, units, nanos,
			status, provider, provider_payment_id, failure_reason, created_at, updated_at
		FROM donations
		WHERE donation_id = $1
	`, id)
	return scanDonation(row)
}

func (s *Store) UpdateDonation(ctx context.Context, d domain.Donation) error {
	tag, err := s.q(ctx).Exec(ctx, `
		UPDATE donations
		SET status=$2, provider_payment_id=$3, failure_reason=$4, updated_at=$5
		WHERE donation_id=$1
	`, d.ID, d.Status, d.ProviderPaymentID, d.FailureReason, d.UpdatedAt)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) ListDonations(ctx context.Context, filter application.ListDonationsFilter) ([]domain.Donation, string, error) {
	limit := normalizeLimit(filter.PageSize)
	cursorAt, cursorID, hasCursor, err := decodeCursor(filter.PageToken)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.q(ctx).Query(ctx, `
		SELECT donation_id, payer_profile_id, target_type, target_id, currency_code, units, nanos,
			status, provider, provider_payment_id, failure_reason, created_at, updated_at
		FROM donations
		WHERE ($1 = '' OR payer_profile_id = $1)
		  AND ($2 = '' OR target_type = $2)
		  AND ($3 = false OR (created_at, donation_id) < ($4, $5))
		ORDER BY created_at DESC, donation_id DESC
		LIMIT $6
	`, filter.ProfileID, filter.TargetType, hasCursor, cursorAt, cursorID, limit+1)
	if err != nil {
		return nil, "", mapErr(err)
	}
	defer rows.Close()
	return collectDonations(rows, limit)
}

func (s *Store) CreateBoost(ctx context.Context, b domain.Boost) error {
	_, err := s.q(ctx).Exec(ctx, `
		INSERT INTO boosts(boost_id, animal_id, owner_profile_id, donation_id, starts_at, expires_at, active, cancel_reason, cancelled_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, b.ID, b.AnimalID, b.OwnerProfileID, b.DonationID, b.StartsAt, b.ExpiresAt, b.Active, b.CancelReason, nullableTime(b.CancelledAt))
	return mapErr(err)
}

func (s *Store) GetBoost(ctx context.Context, id string) (domain.Boost, error) {
	row := s.q(ctx).QueryRow(ctx, `
		SELECT boost_id, animal_id, owner_profile_id, donation_id, starts_at, expires_at, active,
			COALESCE(cancel_reason, ''), COALESCE(cancelled_at, '0001-01-01'::timestamp)
		FROM boosts
		WHERE boost_id=$1
	`, id)
	return scanBoost(row)
}

func (s *Store) UpdateBoost(ctx context.Context, b domain.Boost) error {
	tag, err := s.q(ctx).Exec(ctx, `
		UPDATE boosts
		SET active=$2, cancel_reason=$3, cancelled_at=$4
		WHERE boost_id=$1
	`, b.ID, b.Active, b.CancelReason, nullableTime(b.CancelledAt))
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) IsAnimalArchived(ctx context.Context, animalID string) (bool, error) {
	var exists bool
	err := s.q(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM archived_animals WHERE animal_id=$1)`, animalID).Scan(&exists)
	return exists, mapErr(err)
}

func (s *Store) CreateEntitlement(ctx context.Context, e domain.Entitlement) error {
	_, err := s.q(ctx).Exec(ctx, `
		INSERT INTO entitlements(entitlement_id, owner_profile_id, type, resource_id, starts_at, expires_at, active)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, e.ID, e.OwnerProfileID, e.Type, e.ResourceID, e.StartsAt, e.ExpiresAt, e.Active)
	return mapErr(err)
}

func (s *Store) GetEntitlements(ctx context.Context, filter application.GetEntitlementsFilter) ([]domain.Entitlement, error) {
	rows, err := s.q(ctx).Query(ctx, `
		SELECT entitlement_id, owner_profile_id, type, COALESCE(resource_id, ''), starts_at, expires_at, active
		FROM entitlements
		WHERE owner_profile_id=$1
		  AND ($2 = '' OR resource_id = $2)
		  AND active = true
		  AND expires_at > now()
		ORDER BY expires_at DESC, entitlement_id DESC
	`, filter.OwnerProfileID, filter.ResourceID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	var result []domain.Entitlement
	typeFilter := make(map[domain.EntitlementType]struct{}, len(filter.Types))
	for _, typ := range filter.Types {
		typeFilter[typ] = struct{}{}
	}
	for rows.Next() {
		var e domain.Entitlement
		if err := rows.Scan(&e.ID, &e.OwnerProfileID, &e.Type, &e.ResourceID, &e.StartsAt, &e.ExpiresAt, &e.Active); err != nil {
			return nil, mapErr(err)
		}
		if len(typeFilter) > 0 {
			if _, ok := typeFilter[e.Type]; !ok {
				continue
			}
		}
		result = append(result, e)
	}
	return result, mapErr(rows.Err())
}

func (s *Store) AddLedgerEntry(ctx context.Context, entry domain.LedgerEntry) error {
	_, err := s.q(ctx).Exec(ctx, `
		INSERT INTO ledger_entries(ledger_entry_id, profile_id, currency_code, units, nanos, reason, reference_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(ledger_entry_id) DO NOTHING
	`, entry.ID, entry.ProfileID, entry.Amount.CurrencyCode, entry.Amount.Units, entry.Amount.Nanos,
		entry.Reason, entry.ReferenceID, entry.CreatedAt)
	return mapErr(err)
}

func (s *Store) ListLedgerEntries(ctx context.Context, filter application.ListLedgerEntriesFilter) ([]domain.LedgerEntry, string, error) {
	limit := normalizeLimit(filter.PageSize)
	cursorAt, cursorID, hasCursor, err := decodeCursor(filter.PageToken)
	if err != nil {
		return nil, "", err
	}
	rows, err := s.q(ctx).Query(ctx, `
		SELECT ledger_entry_id, profile_id, currency_code, units, nanos, reason, reference_id, created_at
		FROM ledger_entries
		WHERE ($1 = '' OR profile_id=$1)
		  AND ($2 = false OR (created_at, ledger_entry_id) < ($3, $4))
		ORDER BY created_at DESC, ledger_entry_id DESC
		LIMIT $5
	`, filter.ProfileID, hasCursor, cursorAt, cursorID, limit+1)
	if err != nil {
		return nil, "", mapErr(err)
	}
	defer rows.Close()
	return collectLedger(rows, limit)
}

func normalizeLimit(pageSize int) int {
	if pageSize <= 0 {
		return 50
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
