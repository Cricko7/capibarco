package postgres

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/petmatch/petmatch/internal/domain"
)

func scanDonation(row pgx.Row) (domain.Donation, error) {
	var d domain.Donation
	err := row.Scan(&d.ID, &d.PayerProfileID, &d.TargetType, &d.TargetID, &d.Amount.CurrencyCode, &d.Amount.Units,
		&d.Amount.Nanos, &d.Status, &d.Provider, &d.ProviderPaymentID, &d.FailureReason, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return domain.Donation{}, mapErr(err)
	}
	return d, nil
}

func scanBoost(row pgx.Row) (domain.Boost, error) {
	var b domain.Boost
	err := row.Scan(&b.ID, &b.AnimalID, &b.OwnerProfileID, &b.DonationID, &b.StartsAt, &b.ExpiresAt,
		&b.Active, &b.CancelReason, &b.CancelledAt)
	if err != nil {
		return domain.Boost{}, mapErr(err)
	}
	return b, nil
}

func collectDonations(rows pgx.Rows, limit int) ([]domain.Donation, string, error) {
	items := make([]domain.Donation, 0, limit)
	for rows.Next() {
		d, err := scanDonation(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, "", mapErr(err)
	}
	var next string
	if len(items) > limit {
		last := items[limit-1]
		next = encodeCursor(last.CreatedAt, last.ID)
		items = items[:limit]
	}
	return items, next, nil
}

func collectLedger(rows pgx.Rows, limit int) ([]domain.LedgerEntry, string, error) {
	items := make([]domain.LedgerEntry, 0, limit)
	for rows.Next() {
		var entry domain.LedgerEntry
		err := rows.Scan(&entry.ID, &entry.ProfileID, &entry.Amount.CurrencyCode, &entry.Amount.Units,
			&entry.Amount.Nanos, &entry.Reason, &entry.ReferenceID, &entry.CreatedAt)
		if err != nil {
			return nil, "", mapErr(err)
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, "", mapErr(err)
	}
	var next string
	if len(items) > limit {
		last := items[limit-1]
		next = encodeCursor(last.CreatedAt, last.ID)
		items = items[:limit]
	}
	return items, next, nil
}

func encodeCursor(createdAt time.Time, id string) string {
	raw := fmt.Sprintf("%s|%s", createdAt.UTC().Format(time.RFC3339Nano), id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(token string) (time.Time, string, bool, error) {
	if token == "" {
		return time.Time{}, "", false, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return time.Time{}, "", false, fmt.Errorf("%w: invalid page token", domain.ErrValidation)
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 || parts[1] == "" {
		return time.Time{}, "", false, fmt.Errorf("%w: invalid page token", domain.ErrValidation)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false, fmt.Errorf("%w: invalid page token", domain.ErrValidation)
	}
	return createdAt, parts[1], true, nil
}
