package worker_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/database"
	"github.com/authentic-et/authentic-et/internal/domain"
	"github.com/authentic-et/authentic-et/internal/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupWorker_Execution(t *testing.T) {
	cfg := &config.Config{
		DatabaseDriver:             "memory",
		CleanupEnabled:             true,
		CleanupIntervalHours:       24,
		TransactionRetentionDays:   90,
		FailedRetentionDays:        30,
		UnclaimedExpiryHours:       72,
		AuditRetentionDays:         180,
		CompletedJobsRetentionDays: 14,
		DryRunCleanup:              false,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := database.NewManager(cfg, logger)
	err := mgr.Initialize(context.Background())
	require.NoError(t, err)
	defer mgr.Close()

	repos := mgr.Repositories()
	cleanupWorker := worker.NewCleanupWorker(cfg, repos, logger)

	ctx := context.Background()
	oldTime := time.Now().Add(-100 * 24 * time.Hour)

	// Seed old records
	_ = repos.Payments.Create(ctx, &domain.Payment{
		ID:            "pay_unclaimed_1",
		ProviderKey:   "cbe",
		TransactionID: "TX_UNCLAIMED_1",
		Amount:        150,
		Status:        domain.PaymentStatusPending,
		CreatedAt:     oldTime,
	})

	_ = repos.Payments.Create(ctx, &domain.Payment{
		ID:            "pay_failed_1",
		ProviderKey:   "telebirr",
		TransactionID: "TX_FAILED_1",
		Amount:        250,
		Status:        domain.PaymentStatusFailed,
		CreatedAt:     oldTime,
	})

	_ = repos.Payments.Create(ctx, &domain.Payment{
		ID:            "pay_claimed_1",
		ProviderKey:   "cbe",
		TransactionID: "TX_CLAIMED_1",
		Amount:        500,
		Status:        domain.PaymentStatusClaimed,
		Claimed:       true,
		CreatedAt:     oldTime,
	})

	_ = repos.Provisioning.Create(ctx, &domain.ProvisioningJob{
		ID:          "job_completed_1",
		Key:         "job_key_1",
		ServiceType: "app_hosting",
		Status:      domain.JobCompleted,
		CompletedAt: &oldTime,
		CreatedAt:   oldTime,
	})

	oldAuditTime := time.Now().Add(-200 * 24 * time.Hour)
	_ = repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           "audit_1",
		Action:       "test.event",
		ActorType:    "system",
		ResourceID:   "res_1",
		ResourceType: "payment",
		CreatedAt:    oldAuditTime,
	})

	// 1. Dry run should report no deletions performed
	dryReport, err := cleanupWorker.RunCleanup(ctx, true)
	require.NoError(t, err)
	assert.True(t, dryReport.DryRun)
	assert.Equal(t, int64(0), dryReport.DeletedClaimedPayments)

	// 2. Real cleanup run
	report, err := cleanupWorker.RunCleanup(ctx, false)
	require.NoError(t, err)
	assert.False(t, report.DryRun)
	assert.Equal(t, int64(1), report.ExpiredUnclaimedPayments)
	assert.Equal(t, int64(2), report.DeletedFailedPayments) // original failed + newly expired
	assert.Equal(t, int64(1), report.DeletedClaimedPayments)
	assert.Equal(t, int64(1), report.DeletedCompletedJobs)
	assert.Equal(t, int64(1), report.DeletedAuditLogs)

	// Verify records were purged
	_, err = repos.Payments.GetByID(ctx, "pay_failed_1")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	_, err = repos.Payments.GetByID(ctx, "pay_claimed_1")
	assert.ErrorIs(t, err, domain.ErrNotFound)

	_, err = repos.Provisioning.GetByID(ctx, "job_completed_1")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
