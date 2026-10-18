package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/beecloud/paygate/internal/api"
	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/database"
	"github.com/beecloud/paygate/internal/provider"
	"github.com/beecloud/paygate/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(t *testing.T, allowRegistration bool) http.Handler {
	cfg := &config.Config{
		DatabaseDriver:     "memory",
		Environment:        "development",
		JWTSecret:          "test-secret-key-that-is-at-least-32-chars-long",
		JWTExpiryHours:     24,
		AdminEmail:         "admin@test.local",
		AdminPassword:      "AdminPass123!",
		AllowRegistration:  allowRegistration,
		CORSAllowedOrigins: []string{"http://localhost:3000"},
		PublicRPSBurst:     5,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbMgr := database.NewManager(cfg, logger)
	err := dbMgr.Initialize(context.Background())
	require.NoError(t, err)

	repos := dbMgr.Repositories()
	registry := provider.NewRegistry()

	paymentSvc := service.NewPaymentService(cfg, repos, registry, logger)
	apiKeySvc := service.NewAPIKeyService(cfg, repos, logger)
	provSvc := service.NewProvisioningService(cfg, repos, logger)
	orderSvc := service.NewOrderService(cfg, repos, paymentSvc, provSvc, logger)
	authSvc := service.NewAuthService(cfg, repos, logger)

	return api.NewRouter(api.RouterDependencies{
		Config:       cfg,
		DBManager:    dbMgr,
		Repos:        repos,
		Payments:     paymentSvc,
		APIKeys:      apiKeySvc,
		Orders:       orderSvc,
		Provisioning: provSvc,
		Auth:         authSvc,
		Registry:     registry,
		Logger:       logger,
	})
}

func TestSecurity_HeadersPresent(t *testing.T) {
	router := setupTestRouter(t, false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "0", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "same-origin", rec.Header().Get("Cross-Origin-Opener-Policy"))
}

func TestSecurity_PublicDatabaseStatusInformationDisclosure(t *testing.T) {
	router := setupTestRouter(t, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/database/status", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()

	// Must NOT leak DSN, passwords, or internal connection parameters to unauthenticated internet
	assert.NotContains(t, body, "maskedDsn")
	assert.NotContains(t, body, "connectionError")
	assert.NotContains(t, body, "retention")
	assert.Contains(t, body, "driver")
	assert.Contains(t, body, "isEphemeral")
}

func TestSecurity_LoginRateLimiting(t *testing.T) {
	router := setupTestRouter(t, false)

	payload := `{"email":"admin@test.local","password":"wrong-password"}`

	// Fire repeated login attempts from same IP
	rateLimited := false
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.42:1234"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
	}

	assert.True(t, rateLimited, "Brute force login attempts must trigger HTTP 429 Too Many Requests")
}

func TestSecurity_OpenRegistrationDisabledByDefault(t *testing.T) {
	// 1. By default, registration is disabled once an admin exists
	routerDisabled := setupTestRouter(t, false)

	userPayload, _ := json.Marshal(map[string]string{
		"email":    "stranger@test.local",
		"password": "SecurePassword123!",
		"name":     "Public Stranger",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(userPayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	routerDisabled.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "REGISTRATION_DISABLED")

	// 2. When explicitly enabled via configuration, registration succeeds
	routerEnabled := setupTestRouter(t, true)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(userPayload))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	routerEnabled.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusCreated, rec2.Code)
}

func TestSecurity_InitialAdminSetupWizard(t *testing.T) {
	// When ADMIN_EMAIL and ADMIN_PASSWORD are empty in environment
	cfg := &config.Config{
		DatabaseDriver:     "memory",
		Environment:        "development",
		JWTSecret:          "test-secret-key-that-is-at-least-32-chars-long",
		JWTExpiryHours:     24,
		AdminEmail:         "", // empty!
		AdminPassword:      "", // empty!
		AllowRegistration:  false,
		CORSAllowedOrigins: []string{"http://localhost:3000"},
		PublicRPSBurst:     5,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbMgr := database.NewManager(cfg, logger)
	err := dbMgr.Initialize(context.Background())
	require.NoError(t, err)

	repos := dbMgr.Repositories()
	registry := provider.NewRegistry()
	paymentSvc := service.NewPaymentService(cfg, repos, registry, logger)
	apiKeySvc := service.NewAPIKeyService(cfg, repos, logger)
	provSvc := service.NewProvisioningService(cfg, repos, logger)
	orderSvc := service.NewOrderService(cfg, repos, paymentSvc, provSvc, logger)
	authSvc := service.NewAuthService(cfg, repos, logger)

	router := api.NewRouter(api.RouterDependencies{
		Config:       cfg,
		DBManager:    dbMgr,
		Repos:        repos,
		Payments:     paymentSvc,
		APIKeys:      apiKeySvc,
		Orders:       orderSvc,
		Provisioning: provSvc,
		Auth:         authSvc,
		Registry:     registry,
		Logger:       logger,
	})

	// 1. Verify /auth/setup-status indicates needsSetup = true
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/auth/setup-status", nil)
	recStatus := httptest.NewRecorder()
	router.ServeHTTP(recStatus, reqStatus)
	assert.Equal(t, http.StatusOK, recStatus.Code)

	var statusResp map[string]interface{}
	err = json.Unmarshal(recStatus.Body.Bytes(), &statusResp)
	require.NoError(t, err)
	assert.Equal(t, true, statusResp["needsSetup"])
	assert.Equal(t, float64(0), statusResp["userCount"])

	// 2. Initial registration succeeds and grants admin role
	adminPayload, _ := json.Marshal(map[string]string{
		"email":    "owner@paygate.local",
		"password": "MasterSecurePassword123!",
		"name":     "Platform Owner",
	})
	reqAdmin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(adminPayload))
	reqAdmin.Header.Set("Content-Type", "application/json")
	recAdmin := httptest.NewRecorder()
	router.ServeHTTP(recAdmin, reqAdmin)
	assert.Equal(t, http.StatusCreated, recAdmin.Code)

	var adminResp map[string]interface{}
	err = json.Unmarshal(recAdmin.Body.Bytes(), &adminResp)
	require.NoError(t, err)
	userObj := adminResp["user"].(map[string]interface{})
	assert.Equal(t, "admin", userObj["role"])

	// 3. Setup status now reflects needsSetup = false
	recStatus2 := httptest.NewRecorder()
	router.ServeHTTP(recStatus2, reqStatus)
	assert.Equal(t, http.StatusOK, recStatus2.Code)
	var statusResp2 map[string]interface{}
	err = json.Unmarshal(recStatus2.Body.Bytes(), &statusResp2)
	require.NoError(t, err)
	assert.Equal(t, false, statusResp2["needsSetup"])
	assert.Equal(t, float64(1), statusResp2["userCount"])

	// 4. Any subsequent registration attempt is locked down (403 Forbidden)
	hackerPayload, _ := json.Marshal(map[string]string{
		"email":    "attacker@bad.org",
		"password": "HackerPassword123!",
		"name":     "Attacker",
	})
	reqHacker := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(hackerPayload))
	reqHacker.Header.Set("Content-Type", "application/json")
	recHacker := httptest.NewRecorder()
	router.ServeHTTP(recHacker, reqHacker)
	assert.Equal(t, http.StatusForbidden, recHacker.Code)
	assert.Contains(t, recHacker.Body.String(), "REGISTRATION_DISABLED")
}

func TestPublicHealthAndReadyEndpoints(t *testing.T) {
	router := setupTestRouter(t, false)

	// Check /health
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	router.ServeHTTP(recHealth, reqHealth)
	assert.Equal(t, http.StatusOK, recHealth.Code)
	var healthResp map[string]interface{}
	err := json.Unmarshal(recHealth.Body.Bytes(), &healthResp)
	require.NoError(t, err)
	assert.Equal(t, "ok", healthResp["status"])
	assert.Equal(t, "authentic-et", healthResp["service"])

	// Check /api/v1/health
	reqHealthV1 := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recHealthV1 := httptest.NewRecorder()
	router.ServeHTTP(recHealthV1, reqHealthV1)
	assert.Equal(t, http.StatusOK, recHealthV1.Code)

	// Check /ready
	reqReady := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recReady := httptest.NewRecorder()
	router.ServeHTTP(recReady, reqReady)
	assert.Equal(t, http.StatusOK, recReady.Code)

	// Check /api/v1/ready
	reqReadyV1 := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	recReadyV1 := httptest.NewRecorder()
	router.ServeHTTP(recReadyV1, reqReadyV1)
	assert.Equal(t, http.StatusOK, recReadyV1.Code)
}

func TestSettingsConfiguration_AdminOnly(t *testing.T) {
	router := setupTestRouter(t, false)

	// Unauthenticated request to /api/v1/settings should be rejected with 401
	reqUnauth := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	recUnauth := httptest.NewRecorder()
	router.ServeHTTP(recUnauth, reqUnauth)
	assert.Equal(t, http.StatusUnauthorized, recUnauth.Code)

	// Login as admin
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    "admin@test.local",
		"password": "AdminPass123!",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	router.ServeHTTP(recLogin, reqLogin)
	require.Equal(t, http.StatusOK, recLogin.Code)

	var loginResp map[string]interface{}
	err := json.Unmarshal(recLogin.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	token := loginResp["token"].(string)

	// GET /api/v1/settings with admin token
	reqSettings := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	reqSettings.Header.Set("Authorization", "Bearer "+token)
	recSettings := httptest.NewRecorder()
	router.ServeHTTP(recSettings, reqSettings)
	assert.Equal(t, http.StatusOK, recSettings.Code)

	var settingsResp map[string]interface{}
	err = json.Unmarshal(recSettings.Body.Bytes(), &settingsResp)
	require.NoError(t, err)
	retention := settingsResp["retention"].(map[string]interface{})
	assert.NotNil(t, retention)

	// PUT /api/v1/settings to update retention and tolerance rules
	updatePayload, _ := json.Marshal(map[string]interface{}{
		"retention": map[string]interface{}{
			"cleanupEnabled":           false,
			"cleanupIntervalHours":     48,
			"unclaimedExpiryHours":     96,
			"failedRetentionDays":      60,
			"transactionRetentionDays": 365,
			"dryRunCleanup":            true,
		},
		"verification": map[string]interface{}{
			"underpayTolerance": 2.5,
			"overpayTolerance":  10.0,
			"mockMode":          false,
		},
	})
	reqUpdate := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(updatePayload))
	reqUpdate.Header.Set("Authorization", "Bearer "+token)
	reqUpdate.Header.Set("Content-Type", "application/json")
	recUpdate := httptest.NewRecorder()
	router.ServeHTTP(recUpdate, reqUpdate)
	assert.Equal(t, http.StatusOK, recUpdate.Code)

	var updateResp map[string]interface{}
	err = json.Unmarshal(recUpdate.Body.Bytes(), &updateResp)
	require.NoError(t, err)
	assert.Equal(t, true, updateResp["success"])

	updatedSettings := updateResp["settings"].(map[string]interface{})
	updatedRetention := updatedSettings["retention"].(map[string]interface{})
	assert.Equal(t, false, updatedRetention["cleanupEnabled"])
	assert.Equal(t, float64(48), updatedRetention["cleanupIntervalHours"])
	assert.Equal(t, float64(96), updatedRetention["unclaimedExpiryHours"])
	assert.Equal(t, float64(60), updatedRetention["failedRetentionDays"])
	assert.Equal(t, float64(365), updatedRetention["transactionRetentionDays"])
	assert.Equal(t, true, updatedRetention["dryRunCleanup"])

	updatedVerification := updatedSettings["verification"].(map[string]interface{})
	assert.Equal(t, 2.5, updatedVerification["underpayTolerance"])
	assert.Equal(t, 10.0, updatedVerification["overpayTolerance"])
	assert.Equal(t, false, updatedVerification["mockMode"])
}

