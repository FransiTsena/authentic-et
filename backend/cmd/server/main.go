package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beecloud/paygate/internal/api"
	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/database"
	"github.com/beecloud/paygate/internal/provider"
	"github.com/beecloud/paygate/internal/service"
	"github.com/beecloud/paygate/internal/worker"
)

func main() {
	// 1. Initialize structured logger
	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("starting authentic-et: Multi-Rail Payment Verification & Provisioning Platform")

	// 2. Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration loading failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize Pluggable Database Manager (with automatic in-memory fallback on failure)
	dbMgr := database.NewManager(cfg, logger)
	if err := dbMgr.Initialize(ctx); err != nil {
		logger.Error("database manager initialization error", "error", err)
	}
	defer dbMgr.Close()
	repos := dbMgr.Repositories()

	// 4. Initialize Provider Registry
	registry := provider.NewRegistry()

	// 5. Initialize Business Services
	paymentSvc := service.NewPaymentService(cfg, repos, registry, logger)
	apiKeySvc := service.NewAPIKeyService(cfg, repos, logger)
	provSvc := service.NewProvisioningService(cfg, repos, logger)
	orderSvc := service.NewOrderService(cfg, repos, paymentSvc, provSvc, logger)
	authSvc := service.NewAuthService(cfg, repos, logger)

	// 6. Start Background Worker Pool and Data Retention Worker
	bgWorker := worker.NewWorker(cfg, provSvc, logger)
	bgWorker.Start(ctx)
	defer bgWorker.Stop()

	cleanupWorker := worker.NewCleanupWorker(cfg, repos, logger)
	cleanupWorker.Start(ctx)
	defer cleanupWorker.Stop()

	// 7. Initialize HTTP API Router
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
		Cleanup:      cleanupWorker,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 8. Graceful Shutdown listener
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server listening", "port", cfg.Port, "env", cfg.Environment, "driver", cfg.DatabaseDriver)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-shutdownCh
	logger.Info("shutdown signal received, closing connections gracefully", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}

	logger.Info("PayGate server exited cleanly")
}
