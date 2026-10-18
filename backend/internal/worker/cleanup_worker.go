package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/domain"
)

type CleanupReport struct {
	ExpiredUnclaimedPayments int64         `json:"expiredUnclaimedPayments"`
	DeletedFailedPayments    int64         `json:"deletedFailedPayments"`
	DeletedClaimedPayments   int64         `json:"deletedClaimedPayments"`
	DeletedCompletedJobs     int64         `json:"deletedCompletedJobs"`
	DeletedAuditLogs         int64         `json:"deletedAuditLogs"`
	DryRun                   bool          `json:"dryRun"`
	Duration                 time.Duration `json:"duration"`
	ExecutedAt               time.Time     `json:"executedAt"`
}

type CleanupWorker struct {
	cfg    *config.Config
	repos  *domain.Repositories
	logger *slog.Logger
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewCleanupWorker(
	cfg *config.Config,
	repos *domain.Repositories,
	logger *slog.Logger,
) *CleanupWorker {
	return &CleanupWorker{
		cfg:    cfg,
		repos:  repos,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

func (w *CleanupWorker) Start(ctx context.Context) {
	if !w.cfg.CleanupEnabled {
		w.logger.Info("data retention automated cleanup worker is disabled by configuration (CLEANUP_ENABLED=false)")
		return
	}

	interval := time.Duration(w.cfg.CleanupIntervalHours) * time.Hour
	if interval < 10*time.Minute {
		interval = 10 * time.Minute
	}

	w.logger.Info("starting data retention cleanup worker",
		"intervalHours", w.cfg.CleanupIntervalHours,
		"unclaimedExpiryHours", w.cfg.UnclaimedExpiryHours,
		"failedRetentionDays", w.cfg.FailedRetentionDays,
		"transactionRetentionDays", w.cfg.TransactionRetentionDays,
		"completedJobsRetentionDays", w.cfg.CompletedJobsRetentionDays,
		"auditRetentionDays", w.cfg.AuditRetentionDays,
		"dryRun", w.cfg.DryRunCleanup,
	)

	w.wg.Add(1)
	go w.runCleanupLoop(ctx, interval)
}

func (w *CleanupWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	w.logger.Info("data retention cleanup worker stopped")
}

func (w *CleanupWorker) runCleanupLoop(ctx context.Context, interval time.Duration) {
	defer w.wg.Done()

	// Initial run after brief delay to allow system startup to settle
	select {
	case <-time.After(30 * time.Second):
		if report, err := w.RunCleanup(ctx, w.cfg.DryRunCleanup); err != nil {
			w.logger.Error("initial automated cleanup failed", "error", err)
		} else {
			w.logger.Info("initial automated data retention cleanup completed",
				"expiredUnclaimed", report.ExpiredUnclaimedPayments,
				"deletedFailed", report.DeletedFailedPayments,
				"deletedClaimed", report.DeletedClaimedPayments,
				"deletedJobs", report.DeletedCompletedJobs,
				"deletedAudit", report.DeletedAuditLogs,
				"duration", report.Duration.String(),
			)
		}
	case <-w.stopCh:
		return
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			report, err := w.RunCleanup(ctx, w.cfg.DryRunCleanup)
			if err != nil {
				w.logger.Error("scheduled automated cleanup failed", "error", err)
			} else {
				w.logger.Info("scheduled automated data retention cleanup finished",
					"expiredUnclaimed", report.ExpiredUnclaimedPayments,
					"deletedFailed", report.DeletedFailedPayments,
					"deletedClaimed", report.DeletedClaimedPayments,
					"deletedJobs", report.DeletedCompletedJobs,
					"deletedAudit", report.DeletedAuditLogs,
					"duration", report.Duration.String(),
				)
			}
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// RunCleanup performs the retention rules on payments, jobs, and audits.
func (w *CleanupWorker) RunCleanup(ctx context.Context, dryRun bool) (*CleanupReport, error) {
	start := time.Now().UTC()
	report := &CleanupReport{
		DryRun:     dryRun,
		ExecutedAt: start,
	}

	if dryRun {
		w.logger.Info("executing cleanup in DRY_RUN mode - no mutations will be committed")
		report.Duration = time.Since(start)
		return report, nil
	}

	now := time.Now().UTC()

	// 1. Expire unclaimed pending payments
	if w.cfg.UnclaimedExpiryHours > 0 {
		unclaimedThreshold := now.Add(-time.Duration(w.cfg.UnclaimedExpiryHours) * time.Hour)
		expired, err := w.repos.Payments.ExpireUnclaimedPayments(ctx, unclaimedThreshold)
		if err != nil {
			w.logger.Error("failed expiring unclaimed pending payments", "error", err)
		} else {
			report.ExpiredUnclaimedPayments = expired
		}
	}

	// 2. Delete failed / expired payments
	if w.cfg.FailedRetentionDays > 0 {
		failedThreshold := now.Add(-time.Duration(w.cfg.FailedRetentionDays) * 24 * time.Hour)
		deletedFailed, err := w.repos.Payments.DeleteFailedPayments(ctx, failedThreshold)
		if err != nil {
			w.logger.Error("failed purging expired/failed payments", "error", err)
		} else {
			report.DeletedFailedPayments = deletedFailed
		}
	}

	// 3. Delete old claimed payments
	if w.cfg.TransactionRetentionDays > 0 {
		txThreshold := now.Add(-time.Duration(w.cfg.TransactionRetentionDays) * 24 * time.Hour)
		deletedClaimed, err := w.repos.Payments.DeleteOldClaimedPayments(ctx, txThreshold)
		if err != nil {
			w.logger.Error("failed purging old claimed transactions", "error", err)
		} else {
			report.DeletedClaimedPayments = deletedClaimed
		}
	}

	// 4. Delete completed provisioning outbox jobs
	if w.cfg.CompletedJobsRetentionDays > 0 {
		jobsThreshold := now.Add(-time.Duration(w.cfg.CompletedJobsRetentionDays) * 24 * time.Hour)
		deletedJobs, err := w.repos.Provisioning.DeleteCompletedJobs(ctx, jobsThreshold)
		if err != nil {
			w.logger.Error("failed purging completed provisioning jobs", "error", err)
		} else {
			report.DeletedCompletedJobs = deletedJobs
		}
	}

	// 5. Delete old audit log events
	if w.cfg.AuditRetentionDays > 0 {
		auditThreshold := now.Add(-time.Duration(w.cfg.AuditRetentionDays) * 24 * time.Hour)
		deletedAudit, err := w.repos.Audit.DeleteOldAuditLogs(ctx, auditThreshold)
		if err != nil {
			w.logger.Error("failed purging old audit logs", "error", err)
		} else {
			report.DeletedAuditLogs = deletedAudit
		}
	}

	report.Duration = time.Since(start)
	return report, nil
}
