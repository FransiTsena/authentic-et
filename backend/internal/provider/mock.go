package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) ProviderKey() string {
	return "mock"
}

func (p *MockProvider) DisplayName() string {
	return "Mock Sandbox Verifier"
}

func (p *MockProvider) Verify(ctx context.Context, transactionID string) (*domain.PaymentVerificationResult, error) {
	cleanID := CleanTransactionID(transactionID)
	upper := strings.ToUpper(cleanID)

	if strings.Contains(upper, "CLAIMED") {
		return &domain.PaymentVerificationResult{
			Verified:         false,
			ProviderKey:      "mock",
			TransactionID:    cleanID,
			Amount:           500.0,
			Currency:         "ETB",
			Status:           "FAILED",
			AlreadyProcessed: true,
			Claimed:          true,
			ClaimedBy:        "mock_developer",
			ErrorCode:        "TRANSACTION_ALREADY_CLAIMED",
			ErrorMessage:     fmt.Sprintf("Mock Test Mode: Transaction '%s' is simulated as already claimed.", cleanID),
		}, nil
	}

	if strings.Contains(upper, "NOTFOUND") || strings.Contains(upper, "FAIL") {
		return &domain.PaymentVerificationResult{
			Verified:         false,
			ProviderKey:      "mock",
			TransactionID:    cleanID,
			Currency:         "ETB",
			Status:           "FAILED",
			AlreadyProcessed: false,
			Claimed:          false,
			ErrorCode:        "TRANSACTION_NOT_FOUND",
			ErrorMessage:     fmt.Sprintf("Mock Test Mode: Transaction '%s' simulated as not found.", cleanID),
		}, nil
	}

	if strings.Contains(upper, "INSUFFICIENT") {
		return &domain.PaymentVerificationResult{
			Verified:           true,
			ProviderKey:        "mock",
			TransactionID:      cleanID,
			Amount:             10.0,
			Currency:           "ETB",
			Status:             "COMPLETED",
			PayerIdentifier:    "Sandbox Tester",
			ReceiverIdentifier: "Fransi Tsena (BeeCloud Merchant)",
			Timestamp:          time.Now().Format(time.RFC3339),
		}, nil
	}

	if strings.Contains(upper, "RECEIVER_MISMATCH") {
		return &domain.PaymentVerificationResult{
			Verified:           true,
			ProviderKey:        "mock",
			TransactionID:      cleanID,
			Amount:             500.0,
			Currency:           "ETB",
			Status:             "COMPLETED",
			PayerIdentifier:    "Sandbox Tester",
			ReceiverIdentifier: "Wrong Recipient Individual",
			Timestamp:          time.Now().Format(time.RFC3339),
		}, nil
	}

	return &domain.PaymentVerificationResult{
		Verified:           true,
		ProviderKey:        "mock",
		TransactionID:      cleanID,
		Amount:             500.0,
		Currency:           "ETB",
		Status:             "COMPLETED",
		PayerIdentifier:    "Test Developer (Mock Sandbox)",
		ReceiverIdentifier: "Fransi Tsena (BeeCloud Merchant)",
		Timestamp:          time.Now().Format(time.RFC3339),
	}, nil
}
