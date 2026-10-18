package handlers

import (
	"net/http"
	"strings"

	"github.com/beecloud/paygate/internal/api/middleware"
	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/beecloud/paygate/internal/provider"
	"github.com/beecloud/paygate/internal/service"
	"github.com/go-chi/chi/v5"
)

type PaymentHandler struct {
	cfg         *config.Config
	payments    *service.PaymentService
	apiKeys     *service.APIKeyService
	registry    *provider.Registry
	rateLimiter *middleware.RateLimiter
}

func NewPaymentHandler(
	cfg *config.Config,
	payments *service.PaymentService,
	apiKeys *service.APIKeyService,
	registry *provider.Registry,
	rateLimiter *middleware.RateLimiter,
) *PaymentHandler {
	return &PaymentHandler{
		cfg:         cfg,
		payments:    payments,
		apiKeys:     apiKeys,
		registry:    registry,
		rateLimiter: rateLimiter,
	}
}

func (h *PaymentHandler) VerifyPublicPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawAPIKey := strings.TrimSpace(r.Header.Get("X-API-KEY"))

	var apiKeyRecord *domain.APIKey
	if rawAPIKey != "" {
		key, err := h.apiKeys.ValidateKey(ctx, rawAPIKey)
		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, map[string]string{
				"code":    "INVALID_API_KEY",
				"message": err.Error(),
			})
			return
		}
		apiKeyRecord = key

		// Enforce rate limit per API Key
		rateKey := "key_" + key.ID
		maxRPS := float64(key.MaxRPS)
		if maxRPS <= 0 {
			maxRPS = 10
		}
		if !h.rateLimiter.Allow(rateKey, maxRPS, maxRPS) {
			w.Header().Set("Retry-After", "5")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Rate limit exceeded for this API key",
			})
			return
		}
	} else {
		// Public free tier rate limiting per IP
		ip := middleware.GetClientIP(r)
		rateKey := "ip_" + ip
		burst := float64(h.cfg.PublicRPSBurst)
		if burst <= 0 {
			burst = 5
		}
		if !h.rateLimiter.Allow(rateKey, burst, 1.0) {
			w.Header().Set("Retry-After", "5")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Free anonymous testing limit exceeded. Pass an X-API-KEY header for higher throughput.",
			})
			return
		}
	}

	var req domain.VerifyPaymentRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	result, err := h.payments.VerifyTransaction(ctx, req, apiKeyRecord, false)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func (h *PaymentHandler) ClaimPublicPayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rawAPIKey := strings.TrimSpace(r.Header.Get("X-API-KEY"))
	var apiKeyRecord *domain.APIKey
	if rawAPIKey != "" {
		key, err := h.apiKeys.ValidateKey(ctx, rawAPIKey)
		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, map[string]string{
				"code":    "INVALID_API_KEY",
				"message": err.Error(),
			})
			return
		}
		apiKeyRecord = key
		rateKey := "key_" + key.ID
		if !h.rateLimiter.Allow(rateKey, float64(key.MaxRPS), float64(key.MaxRPS)) {
			w.Header().Set("Retry-After", "5")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Rate limit exceeded for this API key",
			})
			return
		}
	} else {
		// Enforce anonymous rate limiting per IP
		ip := middleware.GetClientIP(r)
		rateKey := "claim_ip_" + ip
		burst := float64(h.cfg.PublicRPSBurst)
		if burst <= 0 {
			burst = 5
		}
		if !h.rateLimiter.Allow(rateKey, burst, 1.0) {
			w.Header().Set("Retry-After", "5")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Anonymous rate limit exceeded. Pass an X-API-KEY header for higher throughput.",
			})
			return
		}
	}

	var req domain.ClaimPaymentRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	if apiKeyRecord != nil {
		req.UserID = apiKeyRecord.OwnerUserID
		if req.ClaimedBy == "" {
			req.ClaimedBy = apiKeyRecord.OwnerEmail
		}
	}

	claimedPayment, err := h.payments.ClaimTransaction(ctx, req)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"providerKey":   claimedPayment.ProviderKey,
		"transactionId": claimedPayment.TransactionID,
		"claimed":       true,
		"claimedAt":     claimedPayment.ClaimedAt,
		"claimedBy":     claimedPayment.ClaimedBy,
		"payment":       claimedPayment,
	})
}

func (h *PaymentHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	list := h.registry.List()
	WriteJSON(w, http.StatusOK, list)
}

func (h *PaymentHandler) GetVerificationStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, _ := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)

	ownerID := ""
	if claims != nil && claims.Role != domain.RoleAdmin {
		ownerID = claims.UserID
	}

	stats, err := h.payments.GetStats(ctx, ownerID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, stats)
}

func (h *PaymentHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, _ := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)

	filter := domain.PaymentListFilter{
		ProviderKey: r.URL.Query().Get("provider"),
		Status:      r.URL.Query().Get("status"),
		Search:      r.URL.Query().Get("search"),
		APIKeyID:    r.URL.Query().Get("apiKeyId"),
		Limit:       GetQueryInt(r, "limit", 50),
		Offset:      GetQueryInt(r, "offset", 0),
	}

	if claimedParam := r.URL.Query().Get("claimed"); claimedParam != "" {
		val := (claimedParam == "true" || claimedParam == "1")
		filter.Claimed = &val
	}

	// Developer can only view own transactions; Admin sees all
	if claims != nil && claims.Role != domain.RoleAdmin {
		filter.OwnerUserID = claims.UserID
	} else if targetUser := r.URL.Query().Get("userId"); targetUser != "" {
		filter.OwnerUserID = targetUser
	}

	items, total, err := h.payments.ListTransactions(ctx, filter)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *PaymentHandler) ReconcileTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, _ := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)
	paymentID := chi.URLParam(r, "id")

	adminID := "admin"
	if claims != nil {
		adminID = claims.UserID
	}

	p, err := h.payments.ReconcileTransaction(ctx, paymentID, adminID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, p)
}
