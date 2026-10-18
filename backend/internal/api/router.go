package api

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/beecloud/paygate/internal/api/handlers"
	"github.com/beecloud/paygate/internal/api/middleware"
	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/database"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/beecloud/paygate/internal/provider"
	"github.com/beecloud/paygate/internal/service"
	"github.com/beecloud/paygate/internal/worker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type RouterDependencies struct {
	Config       *config.Config
	DBManager    *database.Manager
	Repos        *domain.Repositories
	Payments     *service.PaymentService
	APIKeys      *service.APIKeyService
	Orders       *service.OrderService
	Provisioning *service.ProvisioningService
	Auth         *service.AuthService
	Registry     *provider.Registry
	Logger       *slog.Logger
	Cleanup      *worker.CleanupWorker
}

func NewRouter(deps RouterDependencies) http.Handler {
	r := chi.NewRouter()

	// Rate limiter
	rateLimiter := middleware.NewRateLimiter()

	// Global Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.Recoverer(deps.Logger))

	// CORS configuration - W3C specification disallows AllowCredentials with wildcard "*"
	allowCredentials := true
	for _, o := range deps.Config.CORSAllowedOrigins {
		if o == "*" {
			allowCredentials = false
			break
		}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.Config.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-KEY", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: allowCredentials,
		MaxAge:           300,
	}))

	// Handlers
	healthH := handlers.NewHealthHandler(deps.Config, deps.Repos)
	paymentH := handlers.NewPaymentHandler(deps.Config, deps.Payments, deps.APIKeys, deps.Registry, rateLimiter)
	apiKeyH := handlers.NewAPIKeyHandler(deps.APIKeys)
	orderH := handlers.NewOrderHandler(deps.Orders)
	provH := handlers.NewProvisioningHandler(deps.Provisioning)
	authH := handlers.NewAuthHandler(deps.Auth, rateLimiter)
	auditH := handlers.NewAuditHandler(deps.Repos)
	dbH := handlers.NewDatabaseHandler(deps.DBManager, deps.Config, deps.Cleanup)
	settingsH := handlers.NewSettingsHandler(deps.Config, deps.Cleanup)

	// Health Endpoints
	r.Get("/health", healthH.Health)
	r.Get("/ready", healthH.Ready)

	// API v1 Routes
	r.Route("/api/v1", func(v1 chi.Router) {
		// Public Endpoints
		v1.Get("/health", healthH.Health)
		v1.Get("/ready", healthH.Ready)
		v1.Get("/openapi.json", handlers.OpenAPI)
		v1.Get("/public/database/status", dbH.PublicStatus)
		v1.Get("/auth/setup-status", authH.SetupStatus)
		v1.Post("/auth/login", authH.Login)
		v1.Post("/auth/register", authH.Register)

		v1.Post("/public/payment/verify", paymentH.VerifyPublicPayment)
		v1.Post("/public/payment/claim", paymentH.ClaimPublicPayment)
		v1.Get("/public/providers", paymentH.ListProviders)
		v1.With(middleware.OptionalAuth(deps.Config, deps.Auth)).Get("/stats", paymentH.GetVerificationStats)

		// Authenticated Routes
		v1.Group(func(auth chi.Router) {
			auth.Use(middleware.RequireAuth(deps.Config, deps.Auth))

			auth.Get("/auth/me", authH.Me)

			// Settings & Retention Configuration
			auth.With(middleware.RequireAdmin).Get("/settings", settingsH.Get)
			auth.With(middleware.RequireAdmin).Put("/settings", settingsH.Update)

			// Database Management & Dynamic Reconfiguration
			auth.With(middleware.RequireAdmin).Get("/database/status", dbH.Status)
			auth.With(middleware.RequireAdmin).Post("/database/test", dbH.TestConnection)
			auth.With(middleware.RequireAdmin).Post("/database/configure", dbH.Configure)
			auth.With(middleware.RequireAdmin).Post("/database/cleanup", dbH.RunCleanup)

			// API Keys
			auth.Get("/api-keys", apiKeyH.List)
			auth.Post("/api-keys", apiKeyH.Create)
			auth.Delete("/api-keys/{id}", apiKeyH.Revoke)
			auth.Post("/api-keys/{id}/regenerate", apiKeyH.Regenerate)

			// Transactions Explorer
			auth.Get("/transactions", paymentH.ListTransactions)
			auth.With(middleware.RequireAdmin).Post("/transactions/{id}/reconcile", paymentH.ReconcileTransaction)

			// Orders & Entitlements
			auth.Get("/orders", orderH.List)
			auth.Post("/orders", orderH.Create)
			auth.Get("/orders/{id}", orderH.Get)
			auth.Post("/orders/{id}/verify-transaction", orderH.VerifyOrder)

			// Provisioning Jobs
			auth.With(middleware.RequireAdmin).Get("/provisioning/jobs", provH.ListJobs)
			auth.With(middleware.RequireAdmin).Post("/provisioning/jobs/{id}/retry", provH.RetryJob)

			// Audit Logs
			auth.With(middleware.RequireAdmin).Get("/audit/logs", auditH.List)
		})
	})

	// Legacy backward compatibility route aliases for existing BeeCloud consumers
	r.Route("/payment-verifier", func(legacy chi.Router) {
		legacy.Post("/v1/public/payment/verify", paymentH.VerifyPublicPayment)
		legacy.Post("/v1/public/payment/claim", paymentH.ClaimPublicPayment)
		legacy.Get("/providers", paymentH.ListProviders)
		legacy.Get("/stats", paymentH.GetVerificationStats)

		legacy.Group(func(auth chi.Router) {
			auth.Use(middleware.RequireAuth(deps.Config, deps.Auth))
			auth.Get("/api-keys", apiKeyH.List)
			auth.Post("/api-keys", apiKeyH.Create)
			auth.Delete("/api-keys/{id}", apiKeyH.Revoke)
			auth.Get("/transactions", paymentH.ListTransactions)
		})
	})

	// Headless vs SPA serving
	if deps.Config.DisableUI {
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			handlers.WriteJSON(w, http.StatusOK, map[string]any{
				"service": "paygate",
				"version": "1.0.0",
				"mode":    "headless",
				"api":     "/api/v1",
				"health":  "/health",
				"ready":   "/ready",
			})
		})
	} else {
		// Single-Container App Pod Support: Serve static SPA frontend if STATIC_DIR is provided
		staticDir := os.Getenv("STATIC_DIR")
		if staticDir != "" {
			if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
				fs := http.FileServer(http.Dir(staticDir))
				r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
					path := filepath.Join(staticDir, filepath.Clean(req.URL.Path))
					if _, err := os.Stat(path); os.IsNotExist(err) {
						http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
						return
					}
					fs.ServeHTTP(w, req)
				})
			}
		}
	}

	return r
}
