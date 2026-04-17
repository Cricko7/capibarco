package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/petmatch/petmatch/internal/domain"
)

const (
	scopeCreateDonationIntent = "CreateDonationIntent"
	scopeConfirmDonation      = "ConfirmDonation"
	scopeCreateBoost          = "CreateBoost"
	scopePurchaseEntitlement  = "PurchaseEntitlement"
)

type Dependencies struct {
	Store    Store
	Payments PaymentProvider
	Events   EventPublisher
	Clock    Clock
	IDGen    IDGenerator
	Retry    RetryPolicy
	Breaker  *CircuitBreaker
}

type Service struct {
	store    Store
	payments PaymentProvider
	events   EventPublisher
	clock    Clock
	idGen    IDGenerator
	retry    RetryPolicy
	breaker  *CircuitBreaker
}

func NewService(deps Dependencies) *Service {
	return &Service{
		store:    deps.Store,
		payments: deps.Payments,
		events:   deps.Events,
		clock:    deps.Clock,
		idGen:    deps.IDGen,
		retry:    deps.Retry,
		breaker:  deps.Breaker,
	}
}

type CreateDonationIntentInput struct {
	PayerProfileID string
	TargetType     domain.DonationTargetType
	TargetID       string
	Amount         domain.Money
	Provider       string
	IdempotencyKey string
	TraceID        string
	CorrelationID  string
}

type CreateDonationIntentResult struct {
	Donation domain.Donation
	Payment  PaymentIntent
}

func (s *Service) CreateDonationIntent(ctx context.Context, input CreateDonationIntentInput) (CreateDonationIntentResult, error) {
	keyHash, err := domain.HashIdempotencyKey(scopeCreateDonationIntent, input.IdempotencyKey)
	if err != nil {
		return CreateDonationIntentResult{}, err
	}

	if record, err := s.store.GetIdempotency(ctx, scopeCreateDonationIntent, keyHash); err == nil {
		donation, getErr := s.store.GetDonation(ctx, record.ResourceID)
		if getErr != nil {
			return CreateDonationIntentResult{}, fmt.Errorf("get idempotent donation: %w", getErr)
		}
		intent, getErr := s.payments.GetIntent(ctx, donation.ProviderPaymentID)
		if getErr != nil {
			return CreateDonationIntentResult{}, fmt.Errorf("get idempotent payment intent: %w", getErr)
		}
		return CreateDonationIntentResult{Donation: donation, Payment: intent}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return CreateDonationIntentResult{}, fmt.Errorf("get idempotency record: %w", err)
	}

	now := s.clock.Now()
	donation, err := domain.NewDonation(domain.NewDonationParams{
		ID:             s.idGen.NewID("don"),
		PayerProfileID: strings.TrimSpace(input.PayerProfileID),
		TargetType:     input.TargetType,
		TargetID:       strings.TrimSpace(input.TargetID),
		Amount:         input.Amount,
		Provider:       strings.TrimSpace(input.Provider),
		CreatedAt:      now,
	})
	if err != nil {
		return CreateDonationIntentResult{}, err
	}

	var intent PaymentIntent
	err = s.callPayment(ctx, func(callCtx context.Context) error {
		var createErr error
		intent, createErr = s.payments.CreateIntent(callCtx, PaymentIntentInput{
			Provider:       donation.Provider,
			DonationID:     donation.ID,
			PayerProfileID: donation.PayerProfileID,
			Amount:         donation.Amount,
			Description:    "PetMatch donation",
		})
		return createErr
	})
	if err != nil {
		return CreateDonationIntentResult{}, fmt.Errorf("create payment intent: %w", err)
	}
	donation.ProviderPaymentID = intent.ProviderPaymentID

	err = s.store.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.store.CreateDonation(txCtx, donation); err != nil {
			return fmt.Errorf("create donation: %w", err)
		}
		return s.store.SaveIdempotency(txCtx, IdempotencyRecord{
			Scope:        scopeCreateDonationIntent,
			KeyHash:      keyHash,
			ResourceKind: "donation",
			ResourceID:   donation.ID,
			CreatedAt:    now,
		})
	})
	if err != nil {
		return CreateDonationIntentResult{}, err
	}

	return CreateDonationIntentResult{Donation: donation, Payment: intent}, nil
}

type ConfirmDonationInput struct {
	DonationID        string
	ProviderPaymentID string
	IdempotencyKey    string
	TraceID           string
	CorrelationID     string
}

type ConfirmDonationResult struct {
	Donation domain.Donation
}

func (s *Service) ConfirmDonation(ctx context.Context, input ConfirmDonationInput) (ConfirmDonationResult, error) {
	keyHash, err := domain.HashIdempotencyKey(scopeConfirmDonation, input.IdempotencyKey)
	if err != nil {
		return ConfirmDonationResult{}, err
	}
	if record, err := s.store.GetIdempotency(ctx, scopeConfirmDonation, keyHash); err == nil {
		donation, getErr := s.store.GetDonation(ctx, record.ResourceID)
		return ConfirmDonationResult{Donation: donation}, getErr
	} else if !errors.Is(err, domain.ErrNotFound) {
		return ConfirmDonationResult{}, fmt.Errorf("get idempotency record: %w", err)
	}

	donation, err := s.store.GetDonation(ctx, input.DonationID)
	if err != nil {
		return ConfirmDonationResult{}, fmt.Errorf("get donation: %w", err)
	}

	var confirmation PaymentConfirmation
	err = s.callPayment(ctx, func(callCtx context.Context) error {
		var confirmErr error
		confirmation, confirmErr = s.payments.Confirm(callCtx, input.ProviderPaymentID)
		return confirmErr
	})
	if err != nil {
		return ConfirmDonationResult{}, fmt.Errorf("confirm payment: %w", err)
	}

	now := s.clock.Now()
	if !confirmation.Succeeded {
		if markErr := donation.MarkFailed(confirmation.FailureReason, now); markErr != nil {
			return ConfirmDonationResult{}, markErr
		}
		return ConfirmDonationResult{}, s.persistFailedDonation(ctx, donation, input)
	}

	if err := donation.MarkSucceeded(confirmation.ProviderPaymentID, now); err != nil {
		return ConfirmDonationResult{}, err
	}
	err = s.store.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.store.UpdateDonation(txCtx, donation); err != nil {
			return fmt.Errorf("update donation: %w", err)
		}
		if err := s.store.AddLedgerEntry(txCtx, domain.LedgerEntry{
			ID:          s.idGen.NewID("ledger"),
			ProfileID:   donation.PayerProfileID,
			Amount:      donation.Amount,
			Reason:      "donation_succeeded",
			ReferenceID: donation.ID,
			CreatedAt:   now,
		}); err != nil {
			return fmt.Errorf("add ledger entry: %w", err)
		}
		return s.store.SaveIdempotency(txCtx, IdempotencyRecord{
			Scope:        scopeConfirmDonation,
			KeyHash:      keyHash,
			ResourceKind: "donation",
			ResourceID:   donation.ID,
			CreatedAt:    now,
		})
	})
	if err != nil {
		return ConfirmDonationResult{}, err
	}
	if err := s.events.Publish(ctx, BillingEvent{
		Topic:          "billing.donation_succeeded",
		PartitionKey:   donation.ID,
		Type:           "DonationSucceededEvent",
		TraceID:        input.TraceID,
		CorrelationID:  input.CorrelationID,
		IdempotencyKey: input.IdempotencyKey,
		Payload:        donation,
	}); err != nil {
		return ConfirmDonationResult{}, fmt.Errorf("publish donation succeeded: %w", err)
	}
	return ConfirmDonationResult{Donation: donation}, nil
}

func (s *Service) persistFailedDonation(ctx context.Context, donation domain.Donation, input ConfirmDonationInput) error {
	return s.store.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.store.UpdateDonation(txCtx, donation); err != nil {
			return fmt.Errorf("update failed donation: %w", err)
		}
		return s.events.Publish(txCtx, BillingEvent{
			Topic:          "billing.donation_failed",
			PartitionKey:   donation.ID,
			Type:           "DonationFailedEvent",
			TraceID:        input.TraceID,
			CorrelationID:  input.CorrelationID,
			IdempotencyKey: input.IdempotencyKey,
			Payload:        donation,
		})
	})
}

type CreateBoostInput struct {
	AnimalID       string
	OwnerProfileID string
	DonationID     string
	Duration       time.Duration
	IdempotencyKey string
	TraceID        string
	CorrelationID  string
}

type CreateBoostResult struct {
	Boost domain.Boost
}

func (s *Service) CreateBoost(ctx context.Context, input CreateBoostInput) (CreateBoostResult, error) {
	keyHash, err := domain.HashIdempotencyKey(scopeCreateBoost, input.IdempotencyKey)
	if err != nil {
		return CreateBoostResult{}, err
	}
	if record, err := s.store.GetIdempotency(ctx, scopeCreateBoost, keyHash); err == nil {
		boost, getErr := s.store.GetBoost(ctx, record.ResourceID)
		return CreateBoostResult{Boost: boost}, getErr
	} else if !errors.Is(err, domain.ErrNotFound) {
		return CreateBoostResult{}, fmt.Errorf("get idempotency record: %w", err)
	}
	archived, err := s.store.IsAnimalArchived(ctx, input.AnimalID)
	if err != nil {
		return CreateBoostResult{}, fmt.Errorf("check archived animal: %w", err)
	}
	if archived {
		return CreateBoostResult{}, domain.ErrArchivedAnimal
	}
	donation, err := s.store.GetDonation(ctx, input.DonationID)
	if err != nil {
		return CreateBoostResult{}, fmt.Errorf("get donation: %w", err)
	}
	now := s.clock.Now()
	boost, err := domain.NewBoost(domain.NewBoostParams{
		ID:             s.idGen.NewID("boost"),
		AnimalID:       input.AnimalID,
		OwnerProfileID: input.OwnerProfileID,
		DonationID:     input.DonationID,
		DonationStatus: donation.Status,
		Duration:       input.Duration,
		StartsAt:       now,
	})
	if err != nil {
		return CreateBoostResult{}, err
	}
	err = s.store.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.store.CreateBoost(txCtx, boost); err != nil {
			return fmt.Errorf("create boost: %w", err)
		}
		return s.store.SaveIdempotency(txCtx, IdempotencyRecord{
			Scope:        scopeCreateBoost,
			KeyHash:      keyHash,
			ResourceKind: "boost",
			ResourceID:   boost.ID,
			CreatedAt:    now,
		})
	})
	if err != nil {
		return CreateBoostResult{}, err
	}
	if err := s.events.Publish(ctx, BillingEvent{
		Topic:          "billing.boost_activated",
		PartitionKey:   boost.AnimalID,
		Type:           "BoostActivatedEvent",
		TraceID:        input.TraceID,
		CorrelationID:  input.CorrelationID,
		IdempotencyKey: input.IdempotencyKey,
		Payload:        boost,
	}); err != nil {
		return CreateBoostResult{}, fmt.Errorf("publish boost activated: %w", err)
	}
	return CreateBoostResult{Boost: boost}, nil
}

type CancelBoostInput struct {
	BoostID string
	Reason  string
}

type CancelBoostResult struct {
	Boost domain.Boost
}

func (s *Service) CancelBoost(ctx context.Context, input CancelBoostInput) (CancelBoostResult, error) {
	boost, err := s.store.GetBoost(ctx, input.BoostID)
	if err != nil {
		return CancelBoostResult{}, fmt.Errorf("get boost: %w", err)
	}
	boost.Cancel(input.Reason, s.clock.Now())
	if err := s.store.UpdateBoost(ctx, boost); err != nil {
		return CancelBoostResult{}, fmt.Errorf("update boost: %w", err)
	}
	return CancelBoostResult{Boost: boost}, nil
}

type PurchaseEntitlementInput struct {
	OwnerProfileID string
	Type           domain.EntitlementType
	ResourceID     string
	Amount         domain.Money
	Duration       time.Duration
	IdempotencyKey string
	TraceID        string
	CorrelationID  string
}

type PurchaseEntitlementResult struct {
	Entitlement domain.Entitlement
	Donation    domain.Donation
}

func (s *Service) PurchaseEntitlement(ctx context.Context, input PurchaseEntitlementInput) (PurchaseEntitlementResult, error) {
	keyHash, err := domain.HashIdempotencyKey(scopePurchaseEntitlement, input.IdempotencyKey)
	if err != nil {
		return PurchaseEntitlementResult{}, err
	}
	if record, err := s.store.GetIdempotency(ctx, scopePurchaseEntitlement, keyHash); err == nil {
		entitlements, getErr := s.store.GetEntitlements(ctx, GetEntitlementsFilter{OwnerProfileID: input.OwnerProfileID})
		if getErr != nil {
			return PurchaseEntitlementResult{}, getErr
		}
		for _, entitlement := range entitlements {
			if entitlement.ID == record.ResourceID {
				donation, donationErr := s.store.GetDonation(ctx, record.RelatedResourceID)
				return PurchaseEntitlementResult{Entitlement: entitlement, Donation: donation}, donationErr
			}
		}
		return PurchaseEntitlementResult{}, domain.ErrNotFound
	} else if !errors.Is(err, domain.ErrNotFound) {
		return PurchaseEntitlementResult{}, fmt.Errorf("get idempotency record: %w", err)
	}

	now := s.clock.Now()
	donation, err := domain.NewDonation(domain.NewDonationParams{
		ID:             s.idGen.NewID("don"),
		PayerProfileID: input.OwnerProfileID,
		TargetType:     domain.DonationTargetShelter,
		TargetID:       input.OwnerProfileID,
		Amount:         input.Amount,
		Provider:       "mock",
		CreatedAt:      now,
	})
	if err != nil {
		return PurchaseEntitlementResult{}, err
	}
	intent, err := s.createAndConfirmInternalPayment(ctx, donation)
	if err != nil {
		return PurchaseEntitlementResult{}, err
	}
	if err := donation.MarkSucceeded(intent.ProviderPaymentID, now); err != nil {
		return PurchaseEntitlementResult{}, err
	}
	entitlement, err := domain.NewEntitlement(domain.NewEntitlementParams{
		ID:             s.idGen.NewID("entitlement"),
		OwnerProfileID: input.OwnerProfileID,
		Type:           input.Type,
		ResourceID:     input.ResourceID,
		Duration:       input.Duration,
		StartsAt:       now,
	})
	if err != nil {
		return PurchaseEntitlementResult{}, err
	}
	err = s.store.WithinTx(ctx, func(txCtx context.Context) error {
		if err := s.store.CreateDonation(txCtx, donation); err != nil {
			return fmt.Errorf("create entitlement donation: %w", err)
		}
		if err := s.store.CreateEntitlement(txCtx, entitlement); err != nil {
			return fmt.Errorf("create entitlement: %w", err)
		}
		if err := s.store.AddLedgerEntry(txCtx, domain.LedgerEntry{
			ID:          s.idGen.NewID("ledger"),
			ProfileID:   input.OwnerProfileID,
			Amount:      input.Amount,
			Reason:      "entitlement_purchased",
			ReferenceID: entitlement.ID,
			CreatedAt:   now,
		}); err != nil {
			return fmt.Errorf("add ledger entry: %w", err)
		}
		return s.store.SaveIdempotency(txCtx, IdempotencyRecord{
			Scope:             scopePurchaseEntitlement,
			KeyHash:           keyHash,
			ResourceKind:      "entitlement",
			ResourceID:        entitlement.ID,
			RelatedResourceID: donation.ID,
			CreatedAt:         now,
		})
	})
	if err != nil {
		return PurchaseEntitlementResult{}, err
	}
	if err := s.events.Publish(ctx, BillingEvent{
		Topic:          "billing.entitlement_granted",
		PartitionKey:   entitlement.OwnerProfileID,
		Type:           "EntitlementGrantedEvent",
		TraceID:        input.TraceID,
		CorrelationID:  input.CorrelationID,
		IdempotencyKey: input.IdempotencyKey,
		Payload:        entitlement,
	}); err != nil {
		return PurchaseEntitlementResult{}, fmt.Errorf("publish entitlement granted: %w", err)
	}
	return PurchaseEntitlementResult{Entitlement: entitlement, Donation: donation}, nil
}

func (s *Service) createAndConfirmInternalPayment(ctx context.Context, donation domain.Donation) (PaymentIntent, error) {
	var intent PaymentIntent
	err := s.callPayment(ctx, func(callCtx context.Context) error {
		var createErr error
		intent, createErr = s.payments.CreateIntent(callCtx, PaymentIntentInput{
			Provider:       donation.Provider,
			DonationID:     donation.ID,
			PayerProfileID: donation.PayerProfileID,
			Amount:         donation.Amount,
			Description:    "PetMatch entitlement purchase",
		})
		return createErr
	})
	if err != nil {
		return PaymentIntent{}, fmt.Errorf("create entitlement payment: %w", err)
	}
	var confirmation PaymentConfirmation
	err = s.callPayment(ctx, func(callCtx context.Context) error {
		var confirmErr error
		confirmation, confirmErr = s.payments.Confirm(callCtx, intent.ProviderPaymentID)
		return confirmErr
	})
	if err != nil {
		return PaymentIntent{}, fmt.Errorf("confirm entitlement payment: %w", err)
	}
	if !confirmation.Succeeded {
		return PaymentIntent{}, fmt.Errorf("%w: entitlement payment failed: %s", domain.ErrValidation, confirmation.FailureReason)
	}
	return intent, nil
}

func (s *Service) GetDonation(ctx context.Context, donationID string) (domain.Donation, error) {
	return s.store.GetDonation(ctx, donationID)
}

func (s *Service) ListDonations(ctx context.Context, filter ListDonationsFilter) ([]domain.Donation, string, error) {
	return s.store.ListDonations(ctx, filter)
}

func (s *Service) GetEntitlements(ctx context.Context, filter GetEntitlementsFilter) ([]domain.Entitlement, error) {
	return s.store.GetEntitlements(ctx, filter)
}

func (s *Service) ListLedgerEntries(ctx context.Context, filter ListLedgerEntriesFilter) ([]domain.LedgerEntry, string, error) {
	return s.store.ListLedgerEntries(ctx, filter)
}

func (s *Service) callPayment(ctx context.Context, fn func(context.Context) error) error {
	call := func(callCtx context.Context) error {
		if s.breaker == nil {
			return fn(callCtx)
		}
		return s.breaker.Execute(callCtx, fn)
	}
	if s.retry.Attempts <= 1 {
		return call(ctx)
	}
	return s.retry.Do(ctx, call)
}
