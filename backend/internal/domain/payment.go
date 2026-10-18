package domain

import (
	"context"
	"fmt"
	"time"
)

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusVerifying PaymentStatus = "VERIFYING"
	PaymentStatusVerified  PaymentStatus = "VERIFIED"
	PaymentStatusClaimed   PaymentStatus = "CLAIMED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusExpired   PaymentStatus = "EXPIRED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
)

func (s PaymentStatus) String() string {
	return string(s)
}

func (s PaymentStatus) IsTerminal() bool {
	return s == PaymentStatusClaimed || s == PaymentStatusExpired || s == PaymentStatusRefunded
}

func ValidatePaymentTransition(from, to PaymentStatus) error {
	if from == to {
		return nil
	}

	validTransitions := map[PaymentStatus][]PaymentStatus{
		PaymentStatusPending:   {PaymentStatusVerifying, PaymentStatusFailed, PaymentStatusExpired},
		PaymentStatusVerifying: {PaymentStatusVerified, PaymentStatusFailed, PaymentStatusPending},
		PaymentStatusVerified:  {PaymentStatusClaimed, PaymentStatusRefunded, PaymentStatusExpired},
		PaymentStatusClaimed:   {PaymentStatusRefunded},
		PaymentStatusFailed:    {PaymentStatusVerifying, PaymentStatusPending},
		PaymentStatusExpired:   {},
		PaymentStatusRefunded:  {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("%w: unknown state '%s'", ErrInvalidStateTransition, from)
	}

	for _, state := range allowed {
		if state == to {
			return nil
		}
	}

	return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, from, to)
}

type Payment struct {
	ID                 string         `json:"id"`
	ProviderKey        string         `json:"providerKey"`
	TransactionID      string         `json:"transactionId"`
	ReferenceID        string         `json:"referenceId"`
	ReferenceType      string         `json:"referenceType"`
	Amount             float64        `json:"amount"`
	Currency           string         `json:"currency"`
	Status             PaymentStatus  `json:"status"`
	Claimed            bool           `json:"claimed"`
	ClaimedAt          *time.Time     `json:"claimedAt,omitempty"`
	ClaimedBy          string         `json:"claimedBy,omitempty"`
	APIKeyID           string         `json:"apiKeyId,omitempty"`
	APIKeyPrefix       string         `json:"apiKeyPrefix,omitempty"`
	APIKeyName         string         `json:"apiKeyName,omitempty"`
	OwnerUserID        string         `json:"ownerUserId,omitempty"`
	PayerIdentifier    string         `json:"payerIdentifier,omitempty"`
	ReceiverIdentifier string         `json:"receiverIdentifier,omitempty"`
	RawPayload         map[string]any `json:"rawPayload,omitempty"`
	ErrorMessage       string         `json:"errorMessage,omitempty"`
	ErrorCode          string         `json:"errorCode,omitempty"`
	VerifiedAt         *time.Time     `json:"verifiedAt,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type VerifyPaymentRequest struct {
	Provider       string   `json:"provider"`
	TransactionID  string   `json:"transactionId"`
	ExpectedAmount *float64 `json:"expectedAmount,omitempty"`
	ReferenceID    string   `json:"referenceId,omitempty"`
	ReferenceType  string   `json:"referenceType,omitempty"`
	AutoClaim      bool     `json:"autoClaim,omitempty"`
	ClaimedBy      string   `json:"claimedBy,omitempty"`
	UserID         string   `json:"userId,omitempty"`
}

type ClaimPaymentRequest struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transactionId"`
	ClaimedBy     string `json:"claimedBy,omitempty"`
	UserID        string `json:"userId,omitempty"`
}

type PaymentVerificationResult struct {
	Verified           bool           `json:"verified"`
	ProviderKey        string         `json:"providerKey"`
	TransactionID      string         `json:"transactionId"`
	Amount             float64        `json:"amount"`
	Currency           string         `json:"currency"`
	Status             string         `json:"status"`
	AlreadyProcessed   bool           `json:"alreadyProcessed"`
	Claimed            bool           `json:"claimed"`
	ClaimedAt          *time.Time     `json:"claimedAt,omitempty"`
	ClaimedBy          string         `json:"claimedBy,omitempty"`
	PayerIdentifier    string         `json:"payerIdentifier,omitempty"`
	ReceiverIdentifier string         `json:"receiverIdentifier,omitempty"`
	Timestamp          string         `json:"timestamp,omitempty"`
	WarningMessage     string         `json:"warningMessage,omitempty"`
	ErrorMessage       string         `json:"errorMessage,omitempty"`
	ErrorCode          string         `json:"errorCode,omitempty"`
	RawDetails         map[string]any `json:"rawDetails,omitempty"`
}

type PaymentListFilter struct {
	ProviderKey string
	Status      string
	Claimed     *bool
	APIKeyID    string
	OwnerUserID string
	Search      string
	StartDate   *time.Time
	EndDate     *time.Time
	Limit       int
	Offset      int
}

type PaymentRepository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByProviderAndTxID(ctx context.Context, provider, txID string) (*Payment, error)
	GetByReference(ctx context.Context, refID string) (*Payment, error)
	Update(ctx context.Context, p *Payment) error
	Claim(ctx context.Context, id, claimedBy string, claimedAt time.Time) (*Payment, error)
	List(ctx context.Context, filter PaymentListFilter) ([]*Payment, int64, error)
	GetStats(ctx context.Context, ownerUserID string) (*VerificationStats, error)
	ExpireUnclaimedPayments(ctx context.Context, olderThan time.Time) (int64, error)
	DeleteFailedPayments(ctx context.Context, olderThan time.Time) (int64, error)
	DeleteOldClaimedPayments(ctx context.Context, olderThan time.Time) (int64, error)
}

type VerificationStats struct {
	TotalVerifications      int64            `json:"totalVerifications"`
	SuccessfulVerifications int64            `json:"successfulVerifications"`
	FailedVerifications     int64            `json:"failedVerifications"`
	SuccessRatePercent      float64          `json:"successRatePercent"`
	TotalVerifiedAmount     float64          `json:"totalVerifiedAmount"`
	TotalClaimedAmount      float64          `json:"totalClaimedAmount"`
	Providers               map[string]int64 `json:"providers"`
}
