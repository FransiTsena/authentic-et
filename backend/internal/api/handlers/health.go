package handlers

import (
	"net/http"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/domain"
)

type HealthHandler struct {
	cfg   *config.Config
	repos *domain.Repositories
	start time.Time
}

func NewHealthHandler(cfg *config.Config, repos *domain.Repositories) *HealthHandler {
	return &HealthHandler{
		cfg:   cfg,
		repos: repos,
		start: time.Now(),
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "authentic-et",
		"uptime":  time.Since(h.start).String(),
	})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// Ping database by counting users or performing quick query
	ctx := r.Context()
	_, err := h.repos.Users.Count(ctx)
	if err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":   "unready",
			"database": "unhealthy",
			"error":    err.Error(),
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "ready",
		"database": "connected",
		"driver":   h.cfg.DatabaseDriver,
		"uptime":   time.Since(h.start).String(),
	})
}
