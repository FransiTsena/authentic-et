package handlers

import (
	"net/http"

	"github.com/beecloud/paygate/internal/api/middleware"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/beecloud/paygate/internal/service"
	"github.com/go-chi/chi/v5"
)

type APIKeyHandler struct {
	apiKeys *service.APIKeyService
}

func NewAPIKeyHandler(apiKeys *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{apiKeys: apiKeys}
}

type CreateKeyPayload struct {
	Name             string            `json:"name"`
	Tier             domain.APIKeyTier `json:"tier"`
	MonthlyQuota     int               `json:"monthlyQuota"`
	DailyVolumeLimit float64           `json:"dailyVolumeLimit"`
	MaxRPS           int               `json:"maxRps"`
	TargetUserID     string            `json:"targetUserId,omitempty"`
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)

	var payload CreateKeyPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	ownerUserID := claims.UserID
	ownerEmail := claims.Email
	if claims.Role == domain.RoleAdmin && payload.TargetUserID != "" {
		ownerUserID = payload.TargetUserID
	}

	tier := payload.Tier
	monthlyQuota := payload.MonthlyQuota
	dailyVolumeLimit := payload.DailyVolumeLimit
	maxRPS := payload.MaxRPS

	// Enforce tier and quota ceilings for standard non-admin users
	if claims.Role != domain.RoleAdmin {
		tier = domain.TierFree
		if maxRPS <= 0 || maxRPS > 10 {
			maxRPS = 10
		}
		if monthlyQuota <= 0 || monthlyQuota > 5000 {
			monthlyQuota = 5000
		}
		if dailyVolumeLimit <= 0 || dailyVolumeLimit > 100000 {
			dailyVolumeLimit = 100000
		}
	}

	result, err := h.apiKeys.CreateKey(ctx, domain.CreateAPIKeyInput{
		Name:             payload.Name,
		OwnerEmail:       ownerEmail,
		OwnerUserID:      ownerUserID,
		Tier:             tier,
		MonthlyQuota:     monthlyQuota,
		DailyVolumeLimit: dailyVolumeLimit,
		MaxRPS:           maxRPS,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, result)
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)

	ownerID := claims.UserID
	isAdmin := claims.Role == domain.RoleAdmin

	if isAdmin {
		if targetUser := r.URL.Query().Get("userId"); targetUser != "" {
			ownerID = targetUser
		} else {
			ownerID = "" // List all
		}
	}

	keys, err := h.apiKeys.ListKeys(ctx, ownerID, isAdmin)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, keys)
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)
	keyID := chi.URLParam(r, "id")

	isAdmin := claims.Role == domain.RoleAdmin
	if err := h.apiKeys.RevokeKey(ctx, keyID, claims.UserID, isAdmin); err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *APIKeyHandler) Regenerate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)
	keyID := chi.URLParam(r, "id")

	isAdmin := claims.Role == domain.RoleAdmin
	result, err := h.apiKeys.RegenerateKey(ctx, keyID, claims.UserID, isAdmin)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}
