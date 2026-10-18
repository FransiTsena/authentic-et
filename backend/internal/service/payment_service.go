package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/domain"
	"github.com/authentic-et/authentic-et/internal/provider"
	"github.com/google/uuid"
)

type PaymentService struct {
	cfg       *config.Config
	repos     *domain.Repositories
	registry  *provider.Registry
	logger    *slog.Logger
}

func NewPaymentService(
	cfg *config.Config,
	repos *domain.Repositories,
	registry *provider.Registry,
	logger *slog.Logger,
) *PaymentService {
	return &PaymentService{
		cfg:      cfg,
		repos:    repos,
		registry: registry,
		logger:   logger,
	}
}

func (s *PaymentService) VerifyTransaction(
	ctx context.Context,
	req domain.VerifyPaymentRequest,
	apiKey *domain.APIKey,
	platformInternal bool,
) (*domain.PaymentVerificationResult, error) {
	providerKey := strings.ToLower(strings.TrimSpace(req.Provider))
	cleanID := provider.CleanTransactionID(req.TransactionID)

	if cleanID == "" {
		return nil, domain.NewAppError("INVALID_TRANSACTION_ID", "Transaction ID is required", 400, nil)
	}

	// 0. Handle Mock Test Mode for development/sandbox
	if s.cfg.MockMode && (strings.HasPrefix(cleanID, "TEST_") || strings.HasPrefix(cleanID, "SANDBOX_") || strings.HasPrefix(cleanID, "MOCK_")) {
		mockP, err := s.registry.Get("mock")
		if err == nil {
			mockRes, err := mockP.Verify(ctx, cleanID)
			if err == nil {
				if req.ExpectedAmount != nil && *req.ExpectedAmount > 0 && mockRes.Amount == 500.0 {
					mockRes.Amount = *req.ExpectedAmount
				}
				return mockRes, nil
			}
		}
	}

	// 1. Fetch provider from registry
	prov, err := s.registry.Get(providerKey)
	if err != nil {
		return nil, domain.NewAppError("UNSUPPORTED_PROVIDER", fmt.Sprintf("Provider '%s' is not supported", providerKey), 400, err)
	}

	// 2. Execute verification with upstream provider
	result, err := prov.Verify(ctx, cleanID)
	if err != nil {
		s.logger.Error("provider verification execution failed", "provider", providerKey, "transactionId", cleanID, "error", err)
		return nil, domain.NewAppError("PROVIDER_ERROR", "Failed to communicate with provider", 502, err)
	}

	resolvedID := cleanID
	if result.TransactionID != "" {
		resolvedID = provider.CleanTransactionID(result.TransactionID)
	}

	// 3. Amount tolerance check (if expectedAmount is supplied)
	if result.Verified && req.ExpectedAmount != nil && *req.ExpectedAmount > 0 {
		expected := *req.ExpectedAmount
		actual := result.Amount
		diff := actual - expected

		// Tolerates -underpay (e.g. 1 ETB) and +overpay (e.g. 5 ETB)
		if diff < -s.cfg.UnderpayTolerance || diff > s.cfg.OverpayTolerance {
			result.Verified = false
			result.Status = "FAILED"
			result.ErrorCode = "AMOUNT_MISMATCH"
			if diff < -s.cfg.UnderpayTolerance {
				result.ErrorMessage = fmt.Sprintf("Payment amount mismatch: received %.2f %s, expected at least %.2f %s", actual, result.Currency, expected, result.Currency)
			} else {
				result.ErrorMessage = fmt.Sprintf("Payment amount mismatch: received %.2f %s, expected %.2f %s (exceeds tolerance of +%.2f ETB)", actual, result.Currency, expected, result.Currency, s.cfg.OverpayTolerance)
			}
		}
	}

	// 4. Receiver merchant check
	if result.Verified && result.ReceiverIdentifier != "" {
		if !provider.IsReceiverMatchingMerchant(result.ReceiverIdentifier, s.cfg.MerchantNames, s.cfg.MerchantAccounts) {
			result.Verified = false
			result.Status = "FAILED"
			result.ErrorCode = "RECIPIENT_MISMATCH"
			result.ErrorMessage = fmt.Sprintf("Payment recipient mismatch: slip recipient '%s' does not match merchant account", result.ReceiverIdentifier)
		}
	}

	// 5. Check local database for existing transaction record
	existing, err := s.repos.Payments.GetByProviderAndTxID(ctx, providerKey, resolvedID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		s.logger.Error("database query failed", "error", err)
	}

	if existing != nil {
		result.AlreadyProcessed = true
		result.Claimed = existing.Claimed
		result.ClaimedAt = existing.ClaimedAt
		result.ClaimedBy = existing.ClaimedBy

		if existing.Claimed {
			result.Verified = false
			result.Status = "FAILED"
			result.ErrorCode = "TRANSACTION_ALREADY_CLAIMED"
			result.ErrorMessage = fmt.Sprintf("Transaction '%s' via '%s' has already been claimed by %s", resolvedID, providerKey, existing.ClaimedBy)
			result.WarningMessage = result.ErrorMessage
		} else if !result.Verified {
			result.WarningMessage = result.ErrorMessage
		} else {
			result.WarningMessage = fmt.Sprintf("Transaction '%s' was previously verified but is not yet claimed", resolvedID)
		}

		s.logger.Info("transaction re-verified", "provider", providerKey, "txID", resolvedID, "claimed", existing.Claimed)
	} else if result.Verified {
		// Record newly verified payment
		now := time.Now().UTC()
		isClaimed := req.AutoClaim
		var claimedAt *time.Time
		claimedBy := ""
		if isClaimed {
			claimedAt = &now
			claimedBy = req.ClaimedBy
			if claimedBy == "" {
				claimedBy = "system"
			}
		}

		refID := req.ReferenceID
		if refID == "" {
			refID = "tx_" + resolvedID
		}
		refType := req.ReferenceType
		if refType == "" {
			refType = "direct_verification"
		}

		newPayment := &domain.Payment{
			ID:                 uuid.NewString(),
			ProviderKey:        providerKey,
			TransactionID:      resolvedID,
			ReferenceID:        refID,
			ReferenceType:      refType,
			Amount:             result.Amount,
			Currency:           result.Currency,
			Status:             domain.PaymentStatusVerified,
			Claimed:            isClaimed,
			ClaimedAt:          claimedAt,
			ClaimedBy:          claimedBy,
			PayerIdentifier:    result.PayerIdentifier,
			ReceiverIdentifier: result.ReceiverIdentifier,
			RawPayload:         result.RawDetails,
			VerifiedAt:         &now,
		}

		if isClaimed {
			newPayment.Status = domain.PaymentStatusClaimed
		}

		if apiKey != nil {
			newPayment.APIKeyID = apiKey.ID
			newPayment.APIKeyPrefix = apiKey.Prefix
			newPayment.APIKeyName = apiKey.Name
			newPayment.OwnerUserID = apiKey.OwnerUserID
		} else if req.UserID != "" {
			newPayment.OwnerUserID = req.UserID
		}

		if err := s.repos.Payments.Create(ctx, newPayment); err != nil {
			if errors.Is(err, domain.ErrDuplicateKey) {
				// Concurrent write won; re-fetch
				if dup, dupErr := s.repos.Payments.GetByProviderAndTxID(ctx, providerKey, resolvedID); dupErr == nil && dup != nil {
					result.AlreadyProcessed = true
					result.Claimed = dup.Claimed
					result.ClaimedAt = dup.ClaimedAt
					result.ClaimedBy = dup.ClaimedBy
					if dup.Claimed {
						result.Verified = false
						result.Status = "FAILED"
						result.ErrorCode = "TRANSACTION_ALREADY_CLAIMED"
						result.ErrorMessage = fmt.Sprintf("Transaction '%s' has already been claimed", resolvedID)
					}
				}
			} else {
				s.logger.Error("failed to record verified payment", "error", err)
			}
		} else {
			result.AlreadyProcessed = false
			result.Claimed = isClaimed
			result.ClaimedAt = claimedAt
			result.ClaimedBy = claimedBy

			// Increment API Key usage
			if apiKey != nil && !platformInternal {
				today := time.Now().UTC().Format("2006-01-02")
				_ = s.repos.APIKeys.IncrementUsage(ctx, apiKey.ID, result.Amount, today)
			}

			// Record audit event
			_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
				ID:           uuid.NewString(),
				Action:       "payment.verified",
				ActorType:    "api_key",
				ActorID:      newPayment.APIKeyID,
				ResourceID:   newPayment.ID,
				ResourceType: "payment",
				Details: map[string]any{
					"provider":      providerKey,
					"transactionId": resolvedID,
					"amount":        result.Amount,
					"claimed":       isClaimed,
				},
			})
		}
	}

	return result, nil
}

func (s *PaymentService) ClaimTransaction(ctx context.Context, req domain.ClaimPaymentRequest) (*domain.Payment, error) {
	providerKey := strings.ToLower(strings.TrimSpace(req.Provider))
	cleanID := provider.CleanTransactionID(req.TransactionID)

	if cleanID == "" {
		return nil, domain.NewAppError("INVALID_TRANSACTION_ID", "Transaction ID is required", 400, nil)
	}

	existing, err := s.repos.Payments.GetByProviderAndTxID(ctx, providerKey, cleanID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if existing == nil {
		// Verify first
		verifyRes, err := s.VerifyTransaction(ctx, domain.VerifyPaymentRequest{
			Provider:      providerKey,
			TransactionID: cleanID,
			AutoClaim:     true,
			ClaimedBy:     req.ClaimedBy,
		}, nil, true)
		if err != nil {
			return nil, err
		}
		if !verifyRes.Verified {
			return nil, domain.NewAppError("VERIFICATION_FAILED", verifyRes.ErrorMessage, 400, nil)
		}
		return s.repos.Payments.GetByProviderAndTxID(ctx, providerKey, cleanID)
	}

	if existing.Claimed {
		return existing, domain.ErrTransactionAlreadyClaimed
	}

	// Security safeguard: If payment was created by a specific user account,
	// only that user or system/admin can claim it.
	if existing.OwnerUserID != "" && req.UserID != "" && existing.OwnerUserID != req.UserID {
		return nil, domain.NewAppError("FORBIDDEN", "You do not have permission to claim this transaction", 403, nil)
	}

	claimedBy := req.ClaimedBy
	if claimedBy == "" {
		claimedBy = "developer"
	}
	now := time.Now().UTC()

	claimedPayment, err := s.repos.Payments.Claim(ctx, existing.ID, claimedBy, now)
	if err != nil {
		return nil, err
	}

	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "payment.claimed",
		ActorType:    "user",
		ActorID:      claimedBy,
		ResourceID:   claimedPayment.ID,
		ResourceType: "payment",
		Details: map[string]any{
			"provider":      providerKey,
			"transactionId": cleanID,
			"amount":        claimedPayment.Amount,
		},
	})

	return claimedPayment, nil
}

func (s *PaymentService) ListTransactions(ctx context.Context, filter domain.PaymentListFilter) ([]*domain.Payment, int64, error) {
	return s.repos.Payments.List(ctx, filter)
}

func (s *PaymentService) GetStats(ctx context.Context, ownerUserID string) (*domain.VerificationStats, error) {
	return s.repos.Payments.GetStats(ctx, ownerUserID)
}

func (s *PaymentService) ReconcileTransaction(ctx context.Context, paymentID, adminUserID string) (*domain.Payment, error) {
	p, err := s.repos.Payments.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	// Re-verify against provider
	prov, err := s.registry.Get(p.ProviderKey)
	if err != nil {
		return nil, err
	}

	res, err := prov.Verify(ctx, p.TransactionID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if res.Verified {
		p.Status = domain.PaymentStatusVerified
		if p.Claimed {
			p.Status = domain.PaymentStatusClaimed
		}
		p.Amount = res.Amount
		p.PayerIdentifier = res.PayerIdentifier
		p.ReceiverIdentifier = res.ReceiverIdentifier
		p.VerifiedAt = &now
		p.ErrorMessage = ""
		p.ErrorCode = ""
	} else {
		p.ErrorMessage = res.ErrorMessage
		p.ErrorCode = res.ErrorCode
	}

	if err := s.repos.Payments.Update(ctx, p); err != nil {
		return nil, err
	}

	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "payment.reconciled",
		ActorType:    "admin",
		ActorID:      adminUserID,
		ResourceID:   p.ID,
		ResourceType: "payment",
		Details: map[string]any{
			"verified": res.Verified,
			"status":   p.Status,
		},
	})

	return p, nil
}

func HashAPIKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

func ValidateTolerance(actual, expected, underpayTolerance, overpayTolerance float64) bool {
	diff := actual - expected
	return diff >= -(underpayTolerance + 0.01) && diff <= (overpayTolerance + 0.01)
}
