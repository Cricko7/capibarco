package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/petmatch/petmatch/internal/application"
	"github.com/petmatch/petmatch/internal/domain"
)

type MockProvider struct {
	mu      sync.RWMutex
	baseURL string
	secret  []byte
	intents map[string]application.PaymentIntent
}

func NewMockProvider(baseURL string, secret string) *MockProvider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://mock-payments.petmatch.local/pay"
	}
	if strings.TrimSpace(secret) == "" {
		secret = "local-development-secret"
	}
	return &MockProvider{
		baseURL: baseURL,
		secret:  []byte(secret),
		intents: make(map[string]application.PaymentIntent),
	}
}

func (p *MockProvider) CreateIntent(_ context.Context, input application.PaymentIntentInput) (application.PaymentIntent, error) {
	if strings.TrimSpace(input.Provider) != "mock" {
		return application.PaymentIntent{}, fmt.Errorf("%w: unsupported payment provider", domain.ErrValidation)
	}
	if strings.TrimSpace(input.DonationID) == "" {
		return application.PaymentIntent{}, fmt.Errorf("%w: donation id is required", domain.ErrValidation)
	}

	providerPaymentID := "mock_" + input.DonationID
	intent := application.PaymentIntent{
		Provider:          "mock",
		ProviderPaymentID: providerPaymentID,
		PaymentURL:        p.paymentURL(providerPaymentID),
		ClientSecret:      p.clientSecret(providerPaymentID),
	}

	p.mu.Lock()
	p.intents[providerPaymentID] = intent
	p.mu.Unlock()

	return intent, nil
}

func (p *MockProvider) GetIntent(_ context.Context, providerPaymentID string) (application.PaymentIntent, error) {
	p.mu.RLock()
	intent, ok := p.intents[providerPaymentID]
	p.mu.RUnlock()
	if ok {
		return intent, nil
	}
	if !strings.HasPrefix(providerPaymentID, "mock_") {
		return application.PaymentIntent{}, domain.ErrNotFound
	}
	return application.PaymentIntent{
		Provider:          "mock",
		ProviderPaymentID: providerPaymentID,
		PaymentURL:        p.paymentURL(providerPaymentID),
		ClientSecret:      p.clientSecret(providerPaymentID),
	}, nil
}

func (p *MockProvider) Confirm(_ context.Context, providerPaymentID string) (application.PaymentConfirmation, error) {
	if strings.TrimSpace(providerPaymentID) == "" {
		return application.PaymentConfirmation{}, fmt.Errorf("%w: provider payment id is required", domain.ErrValidation)
	}
	if strings.Contains(providerPaymentID, "fail") {
		return application.PaymentConfirmation{
			ProviderPaymentID: providerPaymentID,
			Succeeded:         false,
			FailureReason:     "mock payment declined",
		}, nil
	}
	return application.PaymentConfirmation{ProviderPaymentID: providerPaymentID, Succeeded: true}, nil
}

func (p *MockProvider) paymentURL(providerPaymentID string) string {
	values := url.Values{}
	values.Set("payment_id", providerPaymentID)
	return p.baseURL + "?" + values.Encode()
}

func (p *MockProvider) clientSecret(providerPaymentID string) string {
	mac := hmac.New(sha256.New, p.secret)
	_, _ = mac.Write([]byte(providerPaymentID))
	return "mock_cs_" + hex.EncodeToString(mac.Sum(nil))
}
