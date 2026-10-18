package handlers

import (
	"fmt"
	"net/http"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/database"
	"github.com/beecloud/paygate/internal/worker"
)

type DatabaseHandler struct {
	dbMgr         *database.Manager
	cfg           *config.Config
	cleanupWorker *worker.CleanupWorker
}

func NewDatabaseHandler(dbMgr *database.Manager, cfg *config.Config, cleanupWorker *worker.CleanupWorker) *DatabaseHandler {
	return &DatabaseHandler{
		dbMgr:         dbMgr,
		cfg:           cfg,
		cleanupWorker: cleanupWorker,
	}
}

type RetentionInfo struct {
	Enabled                    bool `json:"enabled"`
	IntervalHours              int  `json:"intervalHours"`
	TransactionRetentionDays   int  `json:"transactionRetentionDays"`
	FailedRetentionDays        int  `json:"failedRetentionDays"`
	UnclaimedExpiryHours       int  `json:"unclaimedExpiryHours"`
	AuditRetentionDays         int  `json:"auditRetentionDays"`
	CompletedJobsRetentionDays int  `json:"completedJobsRetentionDays"`
	DryRun                     bool `json:"dryRun"`
}

// PublicStatus provides safe non-sensitive database operational status for public consumption
func (h *DatabaseHandler) PublicStatus(w http.ResponseWriter, r *http.Request) {
	if h.dbMgr == nil {
		WriteJSON(w, http.StatusOK, map[string]any{
			"database": map[string]any{
				"driver":      "unknown",
				"isEphemeral": false,
				"connected":   false,
			},
		})
		return
	}
	s := h.dbMgr.Status()
	WriteJSON(w, http.StatusOK, map[string]any{
		"database": map[string]any{
			"driver":         s.Driver,
			"isEphemeral":    s.IsEphemeral,
			"warningMessage": s.WarningMessage,
			"connected":      s.Connected,
		},
	})
}

// Status returns full database diagnostics and retention configs for authenticated administrators
func (h *DatabaseHandler) Status(w http.ResponseWriter, r *http.Request) {
	var baseStatus any
	if h.dbMgr == nil {
		baseStatus = map[string]any{
			"driver":      "unknown",
			"isEphemeral": false,
		}
	} else {
		baseStatus = h.dbMgr.Status()
	}

	var retention RetentionInfo
	if h.cfg != nil {
		retention = RetentionInfo{
			Enabled:                    h.cfg.CleanupEnabled,
			IntervalHours:              h.cfg.CleanupIntervalHours,
			TransactionRetentionDays:   h.cfg.TransactionRetentionDays,
			FailedRetentionDays:        h.cfg.FailedRetentionDays,
			UnclaimedExpiryHours:       h.cfg.UnclaimedExpiryHours,
			AuditRetentionDays:         h.cfg.AuditRetentionDays,
			CompletedJobsRetentionDays: h.cfg.CompletedJobsRetentionDays,
			DryRun:                     h.cfg.DryRunCleanup,
		}
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"database":  baseStatus,
		"retention": retention,
	})
}

type DBTestRequest struct {
	Driver string `json:"driver"` // "postgres" or "sqlite"
	DSN    string `json:"dsn"`    // Database URL or SQLite file path
}

func (h *DatabaseHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	var req DBTestRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
		return
	}

	if req.Driver == "" || req.DSN == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "driver and dsn are required",
		})
		return
	}

	err := h.dbMgr.TestConnection(r.Context(), req.Driver, req.DSN)
	if err != nil {
		WriteJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "connection successful and database ping responded",
	})
}

type DBConfigureRequest struct {
	Driver string `json:"driver"` // "postgres" or "sqlite" or "memory"
	DSN    string `json:"dsn"`
}

func (h *DatabaseHandler) Configure(w http.ResponseWriter, r *http.Request) {
	var req DBConfigureRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
		return
	}

	if req.Driver != "memory" && req.DSN == "" {
		WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "dsn is required for postgres or sqlite",
		})
		return
	}

	err := h.dbMgr.SwitchDatabase(r.Context(), req.Driver, req.DSN)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("database switch failed: %v", err),
		})
		return
	}

	newStatus := h.dbMgr.Status()
	WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "database switched and migrations applied successfully",
		"status":  newStatus,
	})
}

type RunCleanupRequest struct {
	DryRun *bool `json:"dryRun,omitempty"`
}

func (h *DatabaseHandler) RunCleanup(w http.ResponseWriter, r *http.Request) {
	if h.cleanupWorker == nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": "cleanup worker is not initialized",
		})
		return
	}

	var req RunCleanupRequest
	_ = ReadJSON(r, &req)

	dryRun := false
	if h.cfg != nil {
		dryRun = h.cfg.DryRunCleanup
	}
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}

	report, err := h.cleanupWorker.RunCleanup(r.Context(), dryRun)
	if err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("cleanup execution failed: %v", err),
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "database cleanup completed",
		"report":  report,
	})
}
