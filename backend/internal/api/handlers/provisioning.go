package handlers

import (
	"net/http"

	"github.com/beecloud/paygate/internal/service"
	"github.com/go-chi/chi/v5"
)

type ProvisioningHandler struct {
	provisioning *service.ProvisioningService
}

func NewProvisioningHandler(provisioning *service.ProvisioningService) *ProvisioningHandler {
	return &ProvisioningHandler{provisioning: provisioning}
}

func (h *ProvisioningHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	limit := GetQueryInt(r, "limit", 50)
	offset := GetQueryInt(r, "offset", 0)

	jobs, total, err := h.provisioning.ListJobs(ctx, status, limit, offset)
	if err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"items":  jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *ProvisioningHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := chi.URLParam(r, "id")

	if err := h.provisioning.RetryJob(ctx, jobID); err != nil {
		WriteError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Job queued for immediate retry",
	})
}
