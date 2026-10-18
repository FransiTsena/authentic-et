package handlers

import (
	"net/http"

	"github.com/beecloud/paygate/internal/api/middleware"
	"github.com/beecloud/paygate/internal/service"
)

type AuthHandler struct {
	auth        *service.AuthService
	rateLimiter *middleware.RateLimiter
}

func NewAuthHandler(auth *service.AuthService, rateLimiter *middleware.RateLimiter) *AuthHandler {
	return &AuthHandler{
		auth:        auth,
		rateLimiter: rateLimiter,
	}
}

type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Rate limit login attempts per IP to prevent brute-force attacks
	if h.rateLimiter != nil {
		ip := middleware.GetClientIP(r)
		if !h.rateLimiter.Allow("login_ip_"+ip, 5, 0.1) {
			w.Header().Set("Retry-After", "60")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many login attempts. Please wait 60 seconds and try again.",
			})
			return
		}
	}

	var payload LoginPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	// Also rate limit per email account to mitigate distributed credential stuffing
	if h.rateLimiter != nil && payload.Email != "" {
		if !h.rateLimiter.Allow("login_email_"+payload.Email, 5, 0.1) {
			w.Header().Set("Retry-After", "60")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many login attempts for this account. Please wait 60 seconds.",
			})
			return
		}
	}

	result, err := h.auth.Login(r.Context(), payload.Email, payload.Password)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Rate limit registration requests per IP to prevent automated account creation
	if h.rateLimiter != nil {
		ip := middleware.GetClientIP(r)
		if !h.rateLimiter.Allow("register_ip_"+ip, 5, 0.05) {
			w.Header().Set("Retry-After", "60")
			WriteJSON(w, http.StatusTooManyRequests, map[string]string{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many registration attempts. Please wait a moment before trying again.",
			})
			return
		}
	}

	var payload RegisterPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	result, err := h.auth.Register(r.Context(), payload.Email, payload.Password, payload.Name)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, result)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(middleware.UserKey).(*service.AuthTokenClaims)
	user, err := h.auth.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.auth.GetSetupStatus(r.Context())
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, status)
}
