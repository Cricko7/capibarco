package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCreateDonationIntentIsIdempotentAndUsesMockProvider(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	payments := &fakePaymentProvider{}
	events := &fakeEventPublisher{}
	service := newTestService(store, payments, events)
	amount := mustMoney(t)

	first, err := service.CreateDonationIntent(context.Background(), application.CreateDonationIntentInput{
		PayerProfileID: "profile_1",
		TargetType:     domain.DonationTargetAnimal,
		TargetID:       "animal_1",
		Amount:         amount,
		Provider:       "mock",
		IdempotencyKey: "client-key",
		CorrelationID:  "corr_1",
	})
	require.NoError(t, err)

	second, err := service.CreateDonationIntent(context.Background(), application.CreateDonationIntentInput{
		PayerProfileID: "profile_1",
		TargetType:     domain.DonationTargetAnimal,
		TargetID:       "animal_1",
		Amount:         amount,
		Provider:       "mock",
		IdempotencyKey: "client-key",
		CorrelationID:  "corr_1",
	})
	require.NoError(t, err)

	require.Equal(t, first.Donation.ID, second.Donation.ID)
	require.Equal(t, first.Payment.ProviderPaymentID, second.Payment.ProviderPaymentID)
	require.NotEmpty(t, first.Payment.PaymentURL)
	require.NotEmpty(t, first.Payment.ClientSecret)
	require.Equal(t, 1, payments.created)
	require.Len(t, store.donations, 1)
}

func TestConfirmDonationCreatesImmutableLedgerAndPublishesEvent(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	payments := &fakePaymentProvider{}
	events := &fakeEventPublisher{}
	service := newTestService(store, payments, events)
	amount := mustMoney(t)

	intent, err := service.CreateDonationIntent(context.Background(), application.CreateDonationIntentInput{
		PayerProfileID: "profile_1",
		TargetType:     domain.DonationTargetShelter,
		TargetID:       "shelter_1",
		Amount:         amount,
		Provider:       "mock",
		IdempotencyKey: "create-key",
	})
	require.NoError(t, err)

	confirmed, err := service.ConfirmDonation(context.Background(), application.ConfirmDonationInput{
		DonationID:        intent.Donation.ID,
		ProviderPaymentID: intent.Payment.ProviderPaymentID,
		IdempotencyKey:    "confirm-key",
		TraceID:           "trace_1",
		CorrelationID:     "corr_1",
	})
	require.NoError(t, err)

	require.Equal(t, domain.PaymentSucceeded, confirmed.Donation.Status)
	require.Len(t, store.ledger, 1)
	require.Equal(t, intent.Donation.ID, store.ledger[0].ReferenceID)
	require.Equal(t, []string{"billing.donation_succeeded"}, events.topics)
}

func TestCreateBoostRejectsArchivedAnimal(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	service := newTestService(store, &fakePaymentProvider{}, &fakeEventPublisher{})
	amount := mustMoney(t)
	donation, err := domain.NewDonation(domain.NewDonationParams{
		ID:             "don_1",
		PayerProfileID: "profile_1",
		TargetType:     domain.DonationTargetAnimal,
		TargetID:       "animal_1",
		Amount:         amount,
		Provider:       "mock",
		CreatedAt:      time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NoError(t, donation.MarkSucceeded("pay_1", time.Now().UTC()))
	require.NoError(t, store.CreateDonation(context.Background(), donation))
	store.archivedAnimals["animal_1"] = true

	_, err = service.CreateBoost(context.Background(), application.CreateBoostInput{
		AnimalID:       "animal_1",
		OwnerProfileID: "profile_1",
		DonationID:     "don_1",
		Duration:       24 * time.Hour,
		IdempotencyKey: "boost-key",
	})

	require.ErrorIs(t, err, domain.ErrArchivedAnimal)
}

func TestPurchaseEntitlementCreatesGrantLedgerAndEvent(t *testing.T) {
	t.Parallel()

	store := newMemoryStore()
	events := &fakeEventPublisher{}
	service := newTestService(store, &fakePaymentProvider{}, events)
	amount := mustMoney(t)

	result, err := service.PurchaseEntitlement(context.Background(), application.PurchaseEntitlementInput{
		OwnerProfileID: "profile_1",
		Type:           domain.EntitlementAdvancedFilters,
		Amount:         amount,
		Duration:       30 * 24 * time.Hour,
		IdempotencyKey: "entitlement-key",
	})
	require.NoError(t, err)

	require.Equal(t, domain.PaymentSucceeded, result.Donation.Status)
	require.True(t, result.Entitlement.Active)
	require.Len(t, store.entitlements, 1)
	require.Len(t, store.ledger, 1)
	require.Equal(t, []string{"billing.entitlement_granted"}, events.topics)

	replayed, err := service.PurchaseEntitlement(context.Background(), application.PurchaseEntitlementInput{
		OwnerProfileID: "profile_1",
		Type:           domain.EntitlementAdvancedFilters,
		Amount:         amount,
		Duration:       30 * 24 * time.Hour,
		IdempotencyKey: "entitlement-key",
	})
	require.NoError(t, err)
	require.Equal(t, result.Entitlement.ID, replayed.Entitlement.ID)
	require.Equal(t, result.Donation.ID, replayed.Donation.ID)
	require.Len(t, store.entitlements, 1)
	require.Len(t, store.ledger, 1)
}

func newTestService(store *memoryStore, payments application.PaymentProvider, events *fakeEventPublisher) *application.Service {
	return application.NewService(application.Dependencies{
		Store:    store,
		Payments: payments,
		Events:   events,
		Clock:    fixedClock{now: time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)},
		IDGen:    sequentialIDs{},
		Retry:    application.RetryPolicy{Attempts: 1},
		Breaker:  application.NewCircuitBreaker(2, time.Second),
	})
}

func mustMoney(t *testing.T) domain.Money {
	t.Helper()

	amount, err := domain.NewMoney("USD", 10, 0)
	require.NoError(t, err)
	return amount
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequentialIDs struct{}

func (sequentialIDs) NewID(prefix string) string {
	return prefix + "_1"
}

type fakePaymentProvider struct {
	created int
}

func (p *fakePaymentProvider) CreateIntent(_ context.Context, input application.PaymentIntentInput) (application.PaymentIntent, error) {
	p.created++
	return application.PaymentIntent{
		Provider:          input.Provider,
		ProviderPaymentID: "pay_" + input.DonationID,
		PaymentURL:        "https://mock.pay/" + input.DonationID,
		ClientSecret:      "mock_secret_" + input.DonationID,
	}, nil
}

func (p *fakePaymentProvider) GetIntent(_ context.Context, providerPaymentID string) (application.PaymentIntent, error) {
	return application.PaymentIntent{
		Provider:          "mock",
		ProviderPaymentID: providerPaymentID,
		PaymentURL:        "https://mock.pay/" + providerPaymentID,
		ClientSecret:      "mock_secret_" + providerPaymentID,
	}, nil
}

func (p *fakePaymentProvider) Confirm(_ context.Context, providerPaymentID string) (application.PaymentConfirmation, error) {
	return application.PaymentConfirmation{ProviderPaymentID: providerPaymentID, Succeeded: true}, nil
}

type fakeEventPublisher struct {
	topics []string
}

func (p *fakeEventPublisher) Publish(_ context.Context, event application.BillingEvent) error {
	p.topics = append(p.topics, event.Topic)
	return nil
}

type memoryStore struct {
	donations       map[string]domain.Donation
	boosts          map[string]domain.Boost
	entitlements    map[string]domain.Entitlement
	ledger          []domain.LedgerEntry
	idempotency     map[string]application.IdempotencyRecord
	archivedAnimals map[string]bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		donations:       make(map[string]domain.Donation),
		boosts:          make(map[string]domain.Boost),
		entitlements:    make(map[string]domain.Entitlement),
		idempotency:     make(map[string]application.IdempotencyRecord),
		archivedAnimals: make(map[string]bool),
	}
}

func (s *memoryStore) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (s *memoryStore) GetIdempotency(_ context.Context, scope string, hash string) (application.IdempotencyRecord, error) {
	record, ok := s.idempotency[scope+":"+hash]
	if !ok {
		return application.IdempotencyRecord{}, domain.ErrNotFound
	}
	return record, nil
}

func (s *memoryStore) SaveIdempotency(_ context.Context, record application.IdempotencyRecord) error {
	s.idempotency[record.Scope+":"+record.KeyHash] = record
	return nil
}

func (s *memoryStore) CreateDonation(_ context.Context, donation domain.Donation) error {
	if _, ok := s.donations[donation.ID]; ok {
		return domain.ErrConflict
	}
	s.donations[donation.ID] = donation
	return nil
}

func (s *memoryStore) GetDonation(_ context.Context, id string) (domain.Donation, error) {
	donation, ok := s.donations[id]
	if !ok {
		return domain.Donation{}, domain.ErrNotFound
	}
	return donation, nil
}

func (s *memoryStore) UpdateDonation(_ context.Context, donation domain.Donation) error {
	if _, ok := s.donations[donation.ID]; !ok {
		return domain.ErrNotFound
	}
	s.donations[donation.ID] = donation
	return nil
}

func (s *memoryStore) CreateBoost(_ context.Context, boost domain.Boost) error {
	s.boosts[boost.ID] = boost
	return nil
}

func (s *memoryStore) GetBoost(_ context.Context, id string) (domain.Boost, error) {
	boost, ok := s.boosts[id]
	if !ok {
		return domain.Boost{}, domain.ErrNotFound
	}
	return boost, nil
}

func (s *memoryStore) UpdateBoost(_ context.Context, boost domain.Boost) error {
	s.boosts[boost.ID] = boost
	return nil
}

func (s *memoryStore) IsAnimalArchived(_ context.Context, animalID string) (bool, error) {
	return s.archivedAnimals[animalID], nil
}

func (s *memoryStore) CreateEntitlement(_ context.Context, entitlement domain.Entitlement) error {
	s.entitlements[entitlement.ID] = entitlement
	return nil
}

func (s *memoryStore) AddLedgerEntry(_ context.Context, entry domain.LedgerEntry) error {
	for _, existing := range s.ledger {
		if existing.ID == entry.ID {
			return nil
		}
	}
	s.ledger = append(s.ledger, entry)
	return nil
}

func (s *memoryStore) ListDonations(_ context.Context, _ application.ListDonationsFilter) ([]domain.Donation, string, error) {
	return nil, "", errors.New("not used")
}

func (s *memoryStore) GetEntitlements(_ context.Context, _ application.GetEntitlementsFilter) ([]domain.Entitlement, error) {
	items := make([]domain.Entitlement, 0, len(s.entitlements))
	for _, entitlement := range s.entitlements {
		items = append(items, entitlement)
	}
	return items, nil
}

func (s *memoryStore) ListLedgerEntries(_ context.Context, _ application.ListLedgerEntriesFilter) ([]domain.LedgerEntry, string, error) {
	return nil, "", errors.New("not used")
}
