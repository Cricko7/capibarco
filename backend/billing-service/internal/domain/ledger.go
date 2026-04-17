package domain

import "time"

type LedgerEntry struct {
	ID          string
	ProfileID   string
	Amount      Money
	Reason      string
	ReferenceID string
	CreatedAt   time.Time
}
