package handlers

import (
	"net/http"

	"github.com/beecloud/paygate/internal/api/middleware"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/beecloud/paygate/internal/service"
	"github.com/go-chi/chi/v5"
)

type OrderHandler struct {
	orders *service.OrderService
}

func NewOrderHandler(orders *service.OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

type CreateOrderPayload struct {
	ServiceType   string         `json:"serviceType"`
	Tier          string         `json:"tier"`
	Amount        float64        `json:"amount"`
	Currency      string         `json:"currency"`
	BillingPeriod int            `json:"billingPeriod"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)

	var payload CreateOrderPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	order, err := h.orders.CreateOrder(ctx, service.CreateOrderInput{
		UserID:        claims.UserID,
		ServiceType:   payload.ServiceType,
		Tier:          payload.Tier,
		Amount:        payload.Amount,
		Currency:      payload.Currency,
		BillingPeriod: payload.BillingPeriod,
		Metadata:      payload.Metadata,
	})
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)

	userID := claims.UserID
	isAdmin := claims.Role == domain.RoleAdmin

	if isAdmin {
		if targetUser := r.URL.Query().Get("userId"); targetUser != "" {
			userID = targetUser
		} else {
			userID = "" // List all
		}
	}

	limit := GetQueryInt(r, "limit", 50)
	offset := GetQueryInt(r, "offset", 0)

	orders, total, err := h.orders.ListOrders(ctx, userID, limit, offset, isAdmin)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"items":  orders,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderID := chi.URLParam(r, "id")

	order, err := h.orders.GetOrder(ctx, orderID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, order)
}

type VerifyOrderPaymentPayload struct {
	Provider      string `json:"provider"`
	TransactionID string `json:"transactionId"`
}

func (h *OrderHandler) VerifyOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims := ctx.Value(middleware.UserKey).(*service.AuthTokenClaims)
	orderID := chi.URLParam(r, "id")

	var payload VerifyOrderPaymentPayload
	if err := ReadJSON(r, &payload); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]string{
			"code":    "INVALID_JSON",
			"message": "Malformed JSON request body",
		})
		return
	}

	result, err := h.orders.VerifyOrderPayment(ctx, orderID, claims.UserID, payload.Provider, payload.TransactionID)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, result)
}
