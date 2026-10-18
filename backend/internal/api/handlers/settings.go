package handlers

import (
	"net/http"
	"sync"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/worker"
)

type SettingsHandler struct {
	cfg           *config.Config
	cleanupWorker *worker.CleanupWorker
	mu            sync.RWMutex
}

func NewSettingsHandler(cfg *config.Config, cleanupWorker *worker.CleanupWorker) *SettingsHandler {
	return &SettingsHandler{
		cfg:           cfg,
		cleanupWorker: cleanupWorker,
	}
}

type SettingsResponse struct {
	Retention struct {
		CleanupEnabled             bool `json:"cleanupEnabled"`
		CleanupIntervalHours       int  `json:"cleanupIntervalHours"`
		UnclaimedExpiryHours       int  `json:"unclaimedExpiryHours"`
		FailedRetentionDays        int  `json:"failedRetentionDays"`
		TransactionRetentionDays   int  `json:"transactionRetentionDays"`
		CompletedJobsRetentionDays int  `json:"completedJobsRetentionDays"`
		AuditRetentionDays         int  `json:"auditRetentionDays"`
		DryRunCleanup              bool `json:"dryRunCleanup"`
	} `json:"retention"`
	Verification struct {
		UnderpayTolerance float64  `json:"underpayTolerance"`
		OverpayTolerance  float64  `json:"overpayTolerance"`
		MockMode          bool     `json:"mockMode"`
		MerchantNames     []string `json:"merchantNames"`
		MerchantAccounts  []string `json:"merchantAccounts"`
	} `json:"verification"`
	Security struct {
		AllowRegistration bool `json:"allowRegistration"`
		JWTExpiryHours    int  `json:"jwtExpiryHours"`
	} `json:"security"`
}

type UpdateSettingsRequest struct {
	Retention *struct {
		CleanupEnabled             *bool `json:"cleanupEnabled,omitempty"`
		CleanupIntervalHours       *int  `json:"cleanupIntervalHours,omitempty"`
		UnclaimedExpiryHours       *int  `json:"unclaimedExpiryHours,omitempty"`
		FailedRetentionDays        *int  `json:"failedRetentionDays,omitempty"`
		TransactionRetentionDays   *int  `json:"transactionRetentionDays,omitempty"`
		CompletedJobsRetentionDays *int  `json:"completedJobsRetentionDays,omitempty"`
		AuditRetentionDays         *int  `json:"auditRetentionDays,omitempty"`
		DryRunCleanup              *bool `json:"dryRunCleanup,omitempty"`
	} `json:"retention,omitempty"`
	Verification *struct {
		UnderpayTolerance *float64  `json:"underpayTolerance,omitempty"`
		OverpayTolerance  *float64  `json:"overpayTolerance,omitempty"`
		MockMode          *bool     `json:"mockMode,omitempty"`
		MerchantNames     *[]string `json:"merchantNames,omitempty"`
		MerchantAccounts  *[]string `json:"merchantAccounts,omitempty"`
	} `json:"verification,omitempty"`
	Security *struct {
		AllowRegistration *bool `json:"allowRegistration,omitempty"`
		JWTExpiryHours    *int  `json:"jwtExpiryHours,omitempty"`
	} `json:"security,omitempty"`
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var resp SettingsResponse
	if h.cfg != nil {
		resp.Retention.CleanupEnabled = h.cfg.CleanupEnabled
		resp.Retention.CleanupIntervalHours = h.cfg.CleanupIntervalHours
		resp.Retention.UnclaimedExpiryHours = h.cfg.UnclaimedExpiryHours
		resp.Retention.FailedRetentionDays = h.cfg.FailedRetentionDays
		resp.Retention.TransactionRetentionDays = h.cfg.TransactionRetentionDays
		resp.Retention.CompletedJobsRetentionDays = h.cfg.CompletedJobsRetentionDays
		resp.Retention.AuditRetentionDays = h.cfg.AuditRetentionDays
		resp.Retention.DryRunCleanup = h.cfg.DryRunCleanup

		resp.Verification.UnderpayTolerance = h.cfg.UnderpayTolerance
		resp.Verification.OverpayTolerance = h.cfg.OverpayTolerance
		resp.Verification.MockMode = h.cfg.MockMode
		resp.Verification.MerchantNames = h.cfg.MerchantNames
		resp.Verification.MerchantAccounts = h.cfg.MerchantAccounts

		resp.Security.AllowRegistration = h.cfg.AllowRegistration
		resp.Security.JWTExpiryHours = h.cfg.JWTExpiryHours
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var req UpdateSettingsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid request body",
		})
		return
	}

	if h.cfg != nil {
		if req.Retention != nil {
			if req.Retention.CleanupEnabled != nil {
				h.cfg.CleanupEnabled = *req.Retention.CleanupEnabled
			}
			if req.Retention.CleanupIntervalHours != nil {
				h.cfg.CleanupIntervalHours = *req.Retention.CleanupIntervalHours
			}
			if req.Retention.UnclaimedExpiryHours != nil {
				h.cfg.UnclaimedExpiryHours = *req.Retention.UnclaimedExpiryHours
			}
			if req.Retention.FailedRetentionDays != nil {
				h.cfg.FailedRetentionDays = *req.Retention.FailedRetentionDays
			}
			if req.Retention.TransactionRetentionDays != nil {
				h.cfg.TransactionRetentionDays = *req.Retention.TransactionRetentionDays
			}
			if req.Retention.CompletedJobsRetentionDays != nil {
				h.cfg.CompletedJobsRetentionDays = *req.Retention.CompletedJobsRetentionDays
			}
			if req.Retention.AuditRetentionDays != nil {
				h.cfg.AuditRetentionDays = *req.Retention.AuditRetentionDays
			}
			if req.Retention.DryRunCleanup != nil {
				h.cfg.DryRunCleanup = *req.Retention.DryRunCleanup
			}
		}

		if req.Verification != nil {
			if req.Verification.UnderpayTolerance != nil {
				h.cfg.UnderpayTolerance = *req.Verification.UnderpayTolerance
			}
			if req.Verification.OverpayTolerance != nil {
				h.cfg.OverpayTolerance = *req.Verification.OverpayTolerance
			}
			if req.Verification.MockMode != nil {
				h.cfg.MockMode = *req.Verification.MockMode
			}
			if req.Verification.MerchantNames != nil {
				h.cfg.MerchantNames = *req.Verification.MerchantNames
			}
			if req.Verification.MerchantAccounts != nil {
				h.cfg.MerchantAccounts = *req.Verification.MerchantAccounts
			}
		}

		if req.Security != nil {
			if req.Security.AllowRegistration != nil {
				h.cfg.AllowRegistration = *req.Security.AllowRegistration
			}
			if req.Security.JWTExpiryHours != nil {
				h.cfg.JWTExpiryHours = *req.Security.JWTExpiryHours
			}
		}
	}

	// Prepare updated response
	var resp SettingsResponse
	resp.Retention.CleanupEnabled = h.cfg.CleanupEnabled
	resp.Retention.CleanupIntervalHours = h.cfg.CleanupIntervalHours
	resp.Retention.UnclaimedExpiryHours = h.cfg.UnclaimedExpiryHours
	resp.Retention.FailedRetentionDays = h.cfg.FailedRetentionDays
	resp.Retention.TransactionRetentionDays = h.cfg.TransactionRetentionDays
	resp.Retention.CompletedJobsRetentionDays = h.cfg.CompletedJobsRetentionDays
	resp.Retention.AuditRetentionDays = h.cfg.AuditRetentionDays
	resp.Retention.DryRunCleanup = h.cfg.DryRunCleanup

	resp.Verification.UnderpayTolerance = h.cfg.UnderpayTolerance
	resp.Verification.OverpayTolerance = h.cfg.OverpayTolerance
	resp.Verification.MockMode = h.cfg.MockMode
	resp.Verification.MerchantNames = h.cfg.MerchantNames
	resp.Verification.MerchantAccounts = h.cfg.MerchantAccounts

	resp.Security.AllowRegistration = h.cfg.AllowRegistration
	resp.Security.JWTExpiryHours = h.cfg.JWTExpiryHours

	WriteJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"message":  "settings updated successfully",
		"settings": resp,
	})
}
