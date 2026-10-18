package domain_test

import (
	"testing"

	"github.com/beecloud/paygate/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPaymentStateTransitions(t *testing.T) {
	// Valid transitions
	assert.NoError(t, domain.ValidatePaymentTransition(domain.PaymentStatusPending, domain.PaymentStatusVerifying))
	assert.NoError(t, domain.ValidatePaymentTransition(domain.PaymentStatusVerifying, domain.PaymentStatusVerified))
	assert.NoError(t, domain.ValidatePaymentTransition(domain.PaymentStatusVerified, domain.PaymentStatusClaimed))
	assert.NoError(t, domain.ValidatePaymentTransition(domain.PaymentStatusClaimed, domain.PaymentStatusRefunded))
	assert.NoError(t, domain.ValidatePaymentTransition(domain.PaymentStatusVerifying, domain.PaymentStatusFailed))
	assert.NoError(t, domain.ValidatePaymentTransition(domain.PaymentStatusPending, domain.PaymentStatusFailed))

	// Invalid transitions
	assert.Error(t, domain.ValidatePaymentTransition(domain.PaymentStatusClaimed, domain.PaymentStatusPending))
	assert.Error(t, domain.ValidatePaymentTransition(domain.PaymentStatusClaimed, domain.PaymentStatusVerified))
	assert.Error(t, domain.ValidatePaymentTransition(domain.PaymentStatusExpired, domain.PaymentStatusVerified))
	assert.Error(t, domain.ValidatePaymentTransition(domain.PaymentStatusRefunded, domain.PaymentStatusClaimed))
}
