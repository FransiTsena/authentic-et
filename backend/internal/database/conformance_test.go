package database_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/database"
	"github.com/authentic-et/authentic-et/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestSQLite(t *testing.T) (*domain.Repositories, database.Closer) {
	tempDir, err := os.MkdirTemp("", "authentic_test_*")
	require.NoError(t, err)

	cfg := &config.Config{
		DatabaseDriver: "sqlite",
		SQLitePath:     filepath.Join(tempDir, "test.db"),
		AdminEmail:     "testadmin@authentic-et.local",
		AdminPassword:  "SecureTestPassword123!",
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	repos, closer, err := database.Initialize(context.Background(), cfg, logger)
	require.NoError(t, err)

	cleanup := func() error {
		_ = closer()
		return os.RemoveAll(tempDir)
	}

	return repos, cleanup
}

func setupTestMemory(t *testing.T) (*domain.Repositories, database.Closer) {
	cfg := &config.Config{
		DatabaseDriver: "memory",
		AdminEmail:     "testadmin@authentic-et.local",
		AdminPassword:  "SecureTestPassword123!",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mgr := database.NewManager(cfg, logger)
	err := mgr.Initialize(context.Background())
	require.NoError(t, err)

	return mgr.Repositories(), mgr.Close
}

func TestDatabaseConformance(t *testing.T) {
	t.Run("SQLite Engine", func(t *testing.T) {
		repos, cleanup := setupTestSQLite(t)
		defer cleanup()
		runConformanceSuite(t, repos)
	})

	t.Run("In-Memory Ephemeral Engine", func(t *testing.T) {
		repos, cleanup := setupTestMemory(t)
		defer cleanup()
		runConformanceSuite(t, repos)
	})
}

func runConformanceSuite(t *testing.T, repos *domain.Repositories) {
	ctx := context.Background()

	// 1. Payment Create & Claim
	t.Run("Payment lifecycle and atomic claim", func(t *testing.T) {
		p := &domain.Payment{
			ID:            uuid.NewString(),
			ProviderKey:   "cbe",
			TransactionID: "FT11223344",
			ReferenceID:   "ref_001",
			Amount:        250.0,
			Currency:      "ETB",
			Status:        domain.PaymentStatusVerified,
			Claimed:       false,
		}

		err := repos.Payments.Create(ctx, p)
		require.NoError(t, err)

		// Duplicate create fails with ErrDuplicateKey
		dupErr := repos.Payments.Create(ctx, p)
		assert.ErrorIs(t, dupErr, domain.ErrDuplicateKey)

		// Fetch
		fetched, err := repos.Payments.GetByID(ctx, p.ID)
		require.NoError(t, err)
		assert.Equal(t, "FT11223344", fetched.TransactionID)
		assert.False(t, fetched.Claimed)

		// Claim
		now := time.Now().UTC()
		claimed, err := repos.Payments.Claim(ctx, p.ID, "user_123", now)
		require.NoError(t, err)
		assert.True(t, claimed.Claimed)
		assert.Equal(t, "user_123", claimed.ClaimedBy)
		assert.Equal(t, domain.PaymentStatusClaimed, claimed.Status)

		// Second claim must fail with ErrTransactionAlreadyClaimed (idempotency & double-spend prevention)
		_, errSecondClaim := repos.Payments.Claim(ctx, p.ID, "user_456", now)
		assert.ErrorIs(t, errSecondClaim, domain.ErrTransactionAlreadyClaimed)
	})

	// 2. API Key Quota and Usage
	t.Run("API Key creation and usage increment", func(t *testing.T) {
		k := &domain.APIKey{
			ID:                  uuid.NewString(),
			KeyHash:             "sample_sha256_hash",
			Prefix:              "sk_pv_12345678",
			Name:                "Test Dev Key",
			OwnerEmail:          "dev@example.com",
			OwnerUserID:         "user_dev_1",
			Active:              true,
			Tier:                domain.TierFree,
			MaxRPS:              10,
			MonthlyQuota:        1000,
			DailyVolumeLimit:    50000,
			LastVolumeResetDate: time.Now().UTC().Format("2006-01-02"),
		}

		err := repos.APIKeys.Create(ctx, k)
		require.NoError(t, err)

		today := time.Now().UTC().Format("2006-01-02")
		err = repos.APIKeys.IncrementUsage(ctx, k.ID, 500.0, today)
		require.NoError(t, err)

		fetched, err := repos.APIKeys.GetByID(ctx, k.ID)
		require.NoError(t, err)
		assert.Equal(t, 1, fetched.UsedThisMonth)
		assert.Equal(t, 500.0, fetched.VerifiedVolumeToday)
	})

	// 3. Provisioning Outbox Jobs and Lease Locking
	t.Run("Provisioning Outbox lease claim and status", func(t *testing.T) {
		job := &domain.ProvisioningJob{
			ID:          uuid.NewString(),
			Key:         "job_key_order_777",
			ServiceType: "vps",
			Action:      "activate",
			Status:      domain.JobPending,
			MaxAttempts: 5,
		}

		err := repos.Provisioning.Create(ctx, job)
		require.NoError(t, err)

		// Duplicate dispatch is idempotent
		dupErr := repos.Provisioning.Create(ctx, job)
		assert.ErrorIs(t, dupErr, domain.ErrDuplicateKey)

		// Worker claim
		claimedBatch, err := repos.Provisioning.ClaimPendingJobs(ctx, 10, 5*time.Minute)
		require.NoError(t, err)
		require.Len(t, claimedBatch, 1)
		assert.Equal(t, job.ID, claimedBatch[0].ID)
		assert.Equal(t, domain.JobRunning, claimedBatch[0].Status)
		assert.Equal(t, 1, claimedBatch[0].Attempts)

		// Mark completed
		err = repos.Provisioning.MarkCompleted(ctx, job.ID)
		require.NoError(t, err)

		updated, err := repos.Provisioning.GetByID(ctx, job.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.JobCompleted, updated.Status)
	})

	// 4. Transaction Manager ACID rollback
	t.Run("TxManager rollback", func(t *testing.T) {
		err := repos.TxManager.WithTx(ctx, func(tx domain.Tx) error {
			_ = tx.Payments().Create(ctx, &domain.Payment{
				ID:            "tx_rollback_test",
				ProviderKey:   "telebirr",
				TransactionID: "TB998877",
				ReferenceID:   "ref_rb",
				Amount:        100,
				Status:        domain.PaymentStatusVerified,
			})
			return domain.ErrRateLimitExceeded // Trigger rollback
		})

		assert.Error(t, err)

		// Verify record does NOT exist after rollback
		_, getErr := repos.Payments.GetByID(ctx, "tx_rollback_test")
		assert.ErrorIs(t, getErr, domain.ErrNotFound)
	})

	// 5. Data Retention & Cleanup Policies
	t.Run("Data retention and cleanup execution", func(t *testing.T) {
		oldTime := time.Now().Add(-100 * 24 * time.Hour)
		threshold := time.Now().Add(-50 * 24 * time.Hour)

		// A. Pending payments older than threshold should expire
		pendingOld := &domain.Payment{
			ID:            "pay_pending_old_" + uuid.NewString(),
			ProviderKey:   "cbe",
			TransactionID: "CBE_CLEANUP_PENDING",
			ReferenceID:   "ref_cleanup_1",
			Amount:        200,
			Status:        domain.PaymentStatusPending,
			CreatedAt:     oldTime,
		}
		require.NoError(t, repos.Payments.Create(ctx, pendingOld))

		expiredCount, err := repos.Payments.ExpireUnclaimedPayments(ctx, threshold)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, expiredCount, int64(1))

		reloaded, err := repos.Payments.GetByID(ctx, pendingOld.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusExpired, reloaded.Status)

		// B. Deleting failed/expired payments
		deletedFailed, err := repos.Payments.DeleteFailedPayments(ctx, threshold)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, deletedFailed, int64(1))

		_, err = repos.Payments.GetByID(ctx, pendingOld.ID)
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// C. Old Claimed Payments
		claimedOld := &domain.Payment{
			ID:            "pay_claimed_old_" + uuid.NewString(),
			ProviderKey:   "telebirr",
			TransactionID: "TB_CLEANUP_CLAIMED",
			ReferenceID:   "ref_cleanup_2",
			Amount:        350,
			Status:        domain.PaymentStatusClaimed,
			Claimed:       true,
			CreatedAt:     oldTime,
		}
		require.NoError(t, repos.Payments.Create(ctx, claimedOld))

		deletedClaimed, err := repos.Payments.DeleteOldClaimedPayments(ctx, threshold)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, deletedClaimed, int64(1))

		_, err = repos.Payments.GetByID(ctx, claimedOld.ID)
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// D. Completed Outbox Jobs
		completedJob := &domain.ProvisioningJob{
			ID:          "job_completed_" + uuid.NewString(),
			Key:         "job_key_" + uuid.NewString(),
			ServiceType: "app_hosting",
			Action:      "activate",
			Status:      domain.JobCompleted,
			CompletedAt: &oldTime,
			CreatedAt:   oldTime,
		}
		require.NoError(t, repos.Provisioning.Create(ctx, completedJob))

		deletedJobs, err := repos.Provisioning.DeleteCompletedJobs(ctx, threshold)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, deletedJobs, int64(1))

		_, err = repos.Provisioning.GetByID(ctx, completedJob.ID)
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// E. Old Audit Logs
		auditOld := &domain.AuditEvent{
			ID:           "audit_old_" + uuid.NewString(),
			Action:       "payment.cleaned",
			ActorType:    "system",
			ResourceID:   "res_1",
			ResourceType: "payment",
			CreatedAt:    oldTime,
		}
		require.NoError(t, repos.Audit.Record(ctx, auditOld))

		deletedAudit, err := repos.Audit.DeleteOldAuditLogs(ctx, threshold)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, deletedAudit, int64(1))
	})
}
