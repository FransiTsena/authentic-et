package domain

import (
	"context"
	"time"
)

type APIKeyTier string

const (
	TierFree       APIKeyTier = "free"
	TierPro        APIKeyTier = "pro"
	TierEnterprise APIKeyTier = "enterprise"
)

type APIKey struct {
	ID                  string     `json:"id"`
	KeyHash             string     `json:"-"`
	Prefix              string     `json:"prefix"`
	Name                string     `json:"name"`
	OwnerEmail          string     `json:"ownerEmail"`
	OwnerUserID         string     `json:"ownerUserId"`
	Active              bool       `json:"active"`
	Tier                APIKeyTier `json:"tier"`
	MaxRPS              int        `json:"maxRps"`
	MonthlyQuota        int        `json:"monthlyQuota"`
	UsedThisMonth       int        `json:"usedThisMonth"`
	DailyVolumeLimit    float64    `json:"dailyVolumeLimit"`
	VerifiedVolumeToday float64    `json:"verifiedVolumeToday"`
	LastVolumeResetDate string     `json:"lastVolumeResetDate"`
	Permissions         []string   `json:"permissions"`
	LastUsedAt          *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type CreateAPIKeyInput struct {
	Name             string     `json:"name"`
	OwnerEmail       string     `json:"ownerEmail"`
	OwnerUserID      string     `json:"ownerUserId"`
	Tier             APIKeyTier `json:"tier"`
	MonthlyQuota     int        `json:"monthlyQuota"`
	DailyVolumeLimit float64    `json:"dailyVolumeLimit"`
	MaxRPS           int        `json:"maxRps"`
}

type APIKeyRepository interface {
	Create(ctx context.Context, k *APIKey) error
	GetByID(ctx context.Context, id string) (*APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*APIKey, error)
	ListByOwner(ctx context.Context, ownerUserID string) ([]*APIKey, error)
	ListAll(ctx context.Context) ([]*APIKey, error)
	Update(ctx context.Context, k *APIKey) error
	IncrementUsage(ctx context.Context, id string, amount float64, todayDate string) error
	Revoke(ctx context.Context, id string) error
}
