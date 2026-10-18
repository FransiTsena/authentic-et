package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/google/uuid"
)

type APIKeyService struct {
	cfg    *config.Config
	repos  *domain.Repositories
	logger *slog.Logger
}

func NewAPIKeyService(cfg *config.Config, repos *domain.Repositories, logger *slog.Logger) *APIKeyService {
	return &APIKeyService{
		cfg:    cfg,
		repos:  repos,
		logger: logger,
	}
}

type GeneratedKey struct {
	RawKey string         `json:"key"`
	Record *domain.APIKey `json:"record"`
}

func (s *APIKeyService) CreateKey(ctx context.Context, input domain.CreateAPIKeyInput) (*GeneratedKey, error) {
	randomBytes := make([]byte, 24)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate secure random key: %w", err)
	}

	rawKey := fmt.Sprintf("sk_pv_%s", hex.EncodeToString(randomBytes))
	keyHash := HashKey(rawKey)
	prefix := rawKey[:16]

	tier := input.Tier
	if tier == "" {
		tier = domain.TierFree
	}

	maxRPS := input.MaxRPS
	if maxRPS <= 0 {
		switch tier {
		case domain.TierEnterprise:
			maxRPS = 500
		case domain.TierPro:
			maxRPS = 100
		default:
			maxRPS = 10
		}
	}

	monthlyQuota := input.MonthlyQuota
	if monthlyQuota <= 0 {
		switch tier {
		case domain.TierEnterprise:
			monthlyQuota = 1000000
		case domain.TierPro:
			monthlyQuota = 50000
		default:
			monthlyQuota = 1000
		}
	}

	dailyVolume := input.DailyVolumeLimit
	if dailyVolume == 0 {
		switch tier {
		case domain.TierEnterprise:
			dailyVolume = -1 // unlimited
		case domain.TierPro:
			dailyVolume = 150000
		default:
			dailyVolume = 50000
		}
	}

	apiKey := &domain.APIKey{
		ID:                  uuid.NewString(),
		KeyHash:             keyHash,
		Prefix:              prefix,
		Name:                input.Name,
		OwnerEmail:          input.OwnerEmail,
		OwnerUserID:         input.OwnerUserID,
		Active:              true,
		Tier:                tier,
		MaxRPS:              maxRPS,
		MonthlyQuota:        monthlyQuota,
		UsedThisMonth:       0,
		DailyVolumeLimit:    dailyVolume,
		VerifiedVolumeToday: 0,
		LastVolumeResetDate: time.Now().UTC().Format("2006-01-02"),
		Permissions:         []string{"payment:verify", "payment:claim"},
	}

	if err := s.repos.APIKeys.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "apikey.created",
		ActorType:    "user",
		ActorID:      input.OwnerUserID,
		ResourceID:   apiKey.ID,
		ResourceType: "apikey",
		Details: map[string]any{
			"prefix": prefix,
			"tier":   tier,
			"name":   input.Name,
		},
	})

	return &GeneratedKey{
		RawKey: rawKey,
		Record: apiKey,
	}, nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*domain.APIKey, error) {
	if rawKey == "" {
		return nil, domain.ErrInvalidAPIKey
	}

	cleanKey := strings.TrimSpace(rawKey)

	// 1. Check against environment preset keys or master token
	if s.cfg != nil {
		for _, preset := range s.cfg.PresetAPIKeys {
			if subtle.ConstantTimeCompare([]byte(cleanKey), []byte(preset)) == 1 {
				prefix := cleanKey
				if len(prefix) > 16 {
					prefix = prefix[:16]
				}
				return &domain.APIKey{
					ID:                  "preset_env_key",
					KeyHash:             HashKey(cleanKey),
					Prefix:              prefix,
					Name:                "Environment Preset Key",
					OwnerEmail:          s.cfg.AdminEmail,
					OwnerUserID:         "system_admin",
					Active:              true,
					Tier:                domain.TierEnterprise,
					MaxRPS:              1000,
					MonthlyQuota:        -1, // unlimited
					DailyVolumeLimit:    -1, // unlimited
					Permissions:         []string{"payment:verify", "payment:claim", "admin"},
					CreatedAt:           time.Now().UTC(),
					UpdatedAt:           time.Now().UTC(),
				}, nil
			}
		}
	}

	// 2. Database lookup
	keyHash := HashKey(cleanKey)
	apiKey, err := s.repos.APIKeys.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, domain.ErrInvalidAPIKey
	}

	if !apiKey.Active {
		return nil, domain.ErrInvalidAPIKey
	}

	// Monthly Quota validation (-1 for unlimited)
	if apiKey.MonthlyQuota > 0 && apiKey.UsedThisMonth >= apiKey.MonthlyQuota {
		return nil, domain.ErrQuotaExceeded
	}

	// Daily Volume Limit validation (-1 for unlimited)
	today := time.Now().UTC().Format("2006-01-02")
	if apiKey.LastVolumeResetDate == today && apiKey.DailyVolumeLimit > 0 && apiKey.VerifiedVolumeToday >= apiKey.DailyVolumeLimit {
		return nil, domain.ErrDailyVolumeExceeded
	}

	return apiKey, nil
}

func (s *APIKeyService) ListKeys(ctx context.Context, ownerUserID string, isAdmin bool) ([]*domain.APIKey, error) {
	if isAdmin && ownerUserID == "" {
		return s.repos.APIKeys.ListAll(ctx)
	}
	return s.repos.APIKeys.ListByOwner(ctx, ownerUserID)
}

func (s *APIKeyService) RevokeKey(ctx context.Context, keyID, actorUserID string, isAdmin bool) error {
	k, err := s.repos.APIKeys.GetByID(ctx, keyID)
	if err != nil {
		return err
	}
	if !isAdmin && k.OwnerUserID != actorUserID {
		return domain.ErrUnauthorized
	}

	if err := s.repos.APIKeys.Revoke(ctx, keyID); err != nil {
		return err
	}

	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "apikey.revoked",
		ActorType:    "user",
		ActorID:      actorUserID,
		ResourceID:   keyID,
		ResourceType: "apikey",
		Details: map[string]any{
			"prefix": k.Prefix,
		},
	})

	return nil
}

func (s *APIKeyService) RegenerateKey(ctx context.Context, keyID, actorUserID string, isAdmin bool) (*GeneratedKey, error) {
	k, err := s.repos.APIKeys.GetByID(ctx, keyID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && k.OwnerUserID != actorUserID {
		return nil, domain.ErrUnauthorized
	}

	// Revoke old key
	_ = s.repos.APIKeys.Revoke(ctx, keyID)

	// Create new key preserving tier and limits
	return s.CreateKey(ctx, domain.CreateAPIKeyInput{
		Name:             k.Name,
		OwnerEmail:       k.OwnerEmail,
		OwnerUserID:      k.OwnerUserID,
		Tier:             k.Tier,
		MonthlyQuota:     k.MonthlyQuota,
		DailyVolumeLimit: k.DailyVolumeLimit,
		MaxRPS:           k.MaxRPS,
	})
}

func HashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
