package handlers

import (
	"net/http"

	"github.com/beecloud/paygate/internal/domain"
)

type AuditHandler struct {
	repos *domain.Repositories
}

func NewAuditHandler(repos *domain.Repositories) *AuditHandler {
	return &AuditHandler{repos: repos}
}

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resourceType := r.URL.Query().Get("resourceType")
	resourceID := r.URL.Query().Get("resourceId")
	limit := GetQueryInt(r, "limit", 50)
	offset := GetQueryInt(r, "offset", 0)

	events, total, err := h.repos.Audit.List(ctx, resourceType, resourceID, limit, offset)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"items":  events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}
