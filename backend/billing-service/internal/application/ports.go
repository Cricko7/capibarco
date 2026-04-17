package application

import (
	"context"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
)

type Store interface {
	WithinTx(ctx context.Context, fn func(context.Context) error) error
	GetIdempotency(ctx context.Context, scope string, hash string) (IdempotencyRecord, error)
	SaveIdempotency(ctx context.Context, record IdempotencyRecord) error
	CreateDonation(ctx context.Context, donation domain.Donation) error
	GetDonation(ctx context.Context, id string) (domain.Donation, error)
	UpdateDonation(ctx context.Context, donation domain.Donation) error
	ListDonations(ctx context.Context, filter ListDonationsFilter) ([]domain.Donation, string, error)
	CreateBoost(ctx context.Context, boost domain.Boost) error
	GetBoost(ctx context.Context, id string) (domain.Boost, error)
	UpdateBoost(ctx context.Context, boost domain.Boost) error
	IsAnimalArchived(ctx context.Context, animalID string) (bool, error)
	CreateEntitlement(ctx context.Context, entitlement domain.Entitlement) error
	GetEntitlements(ctx context.Context, filter GetEntitlementsFilter) ([]domain.Entitlement, error)
	AddLedgerEntry(ctx context.Context, entry domain.LedgerEntry) error
	ListLedgerEntries(ctx context.Context, filter ListLedgerEntriesFilter) ([]domain.LedgerEntry, string, error)
}

type PaymentProvider interface {
	CreateIntent(ctx context.Context, input PaymentIntentInput) (PaymentIntent, error)
	GetIntent(ctx context.Context, providerPaymentID string) (PaymentIntent, error)
	Confirm(ctx context.Context, providerPaymentID string) (PaymentConfirmation, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event BillingEvent) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewID(prefix string) string
}

type IdempotencyRecord struct {
	Scope             string
	KeyHash           string
	ResourceKind      string
	ResourceID        string
	RelatedResourceID string
	CreatedAt         time.Time
}

type PaymentIntentInput struct {
	Provider       string
	DonationID     string
	PayerProfileID string
	Amount         domain.Money
	Description    string
}

type PaymentIntent struct {
	Provider          string
	ProviderPaymentID string
	PaymentURL        string
	ClientSecret      string
}

type PaymentConfirmation struct {
	ProviderPaymentID string
	Succeeded         bool
	FailureReason     string
}

type BillingEvent struct {
	Topic          string
	PartitionKey   string
	Type           string
	TraceID        string
	CorrelationID  string
	IdempotencyKey string
	Payload        any
}

type ListDonationsFilter struct {
	ProfileID  string
	TargetType domain.DonationTargetType
	PageSize   int
	PageToken  string
}

type GetEntitlementsFilter struct {
	OwnerProfileID string
	Types          []domain.EntitlementType
	ResourceID     string
}

type ListLedgerEntriesFilter struct {
	ProfileID string
	PageSize  int
	PageToken string
}
