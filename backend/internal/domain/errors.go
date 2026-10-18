package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound                   = errors.New("record not found")
	ErrTransactionNotFound        = errors.New("transaction not found with provider")
	ErrTransactionAlreadyClaimed  = errors.New("transaction has already been claimed")
	ErrAmountMismatch             = errors.New("payment amount does not match expected amount")
	ErrInsufficientAmount         = errors.New("payment amount is insufficient")
	ErrRecipientMismatch          = errors.New("payment recipient does not match merchant account")
	ErrProviderUnavailable        = errors.New("payment provider is temporarily unavailable")
	ErrProviderTimeout            = errors.New("payment provider request timed out")
	ErrInvalidProvider            = errors.New("unsupported or invalid payment provider")
	ErrRateLimitExceeded          = errors.New("rate limit exceeded")
	ErrUnauthorized               = errors.New("unauthorized")
	ErrInvalidAPIKey              = errors.New("invalid or inactive API key")
	ErrQuotaExceeded              = errors.New("monthly verification quota exceeded")
	ErrDailyVolumeExceeded        = errors.New("daily transaction volume limit exceeded")
	ErrInvalidStateTransition     = errors.New("invalid payment state transition")
	ErrConcurrentModification     = errors.New("concurrent modification detected")
	ErrDuplicateKey               = errors.New("duplicate key violation")
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, statusCode int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}
