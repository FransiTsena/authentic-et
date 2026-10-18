package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type ProvisioningRepo struct {
	db queryable
}

func (r *ProvisioningRepo) Create(ctx context.Context, job *domain.ProvisioningJob) error {
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	job.UpdatedAt = now
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 5
	}
	if job.Status == "" {
		job.Status = domain.JobPending
	}

	payloadJSON, _ := json.Marshal(job.Payload)
	var lockedAtStr, nextAttemptStr, completedAtStr *string
	if job.LockedAt != nil {
		s := job.LockedAt.Format(time.RFC3339)
		lockedAtStr = &s
	}
	if job.NextAttemptAt != nil {
		s := job.NextAttemptAt.Format(time.RFC3339)
		nextAttemptStr = &s
	}
	if job.CompletedAt != nil {
		s := job.CompletedAt.Format(time.RFC3339)
		completedAtStr = &s
	}

	query := `INSERT INTO provisioning_jobs (
		id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Key, job.OrderID, job.UserID, job.ServiceType, job.Action, string(job.Status),
		job.Attempts, job.MaxAttempts, lockedAtStr, nextAttemptStr, completedAtStr,
		job.LastError, string(payloadJSON), job.CreatedAt.Format(time.RFC3339), job.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *ProvisioningRepo) GetByID(ctx context.Context, id string) (*domain.ProvisioningJob, error) {
	query := `SELECT id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
		FROM provisioning_jobs WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanJob(row)
}

func (r *ProvisioningRepo) GetByKey(ctx context.Context, key string) (*domain.ProvisioningJob, error) {
	query := `SELECT id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
		FROM provisioning_jobs WHERE key = ?`
	row := r.db.QueryRowContext(ctx, query, key)
	return scanJob(row)
}

func (r *ProvisioningRepo) ClaimPendingJobs(ctx context.Context, batchSize int, lockDuration time.Duration) ([]*domain.ProvisioningJob, error) {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)
	leaseExpiredStr := now.Add(-lockDuration).Format(time.RFC3339)

	// Step 1: Find candidate jobs
	selectQuery := `SELECT id FROM provisioning_jobs WHERE
		(status = 'pending')
		OR (status = 'failed' AND next_attempt_at <= ?)
		OR (status = 'running' AND locked_at < ?)
		ORDER BY created_at ASC LIMIT ?`

	rows, err := r.db.QueryContext(ctx, selectQuery, nowStr, leaseExpiredStr, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidateIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			candidateIDs = append(candidateIDs, id)
		}
	}

	if len(candidateIDs) == 0 {
		return nil, nil
	}

	// Step 2: Claim candidates atomically
	var claimed []*domain.ProvisioningJob
	for _, id := range candidateIDs {
		updateQuery := `UPDATE provisioning_jobs SET
			status = 'running',
			locked_at = ?,
			attempts = attempts + 1,
			updated_at = ?
			WHERE id = ? AND (
				status = 'pending'
				OR (status = 'failed' AND next_attempt_at <= ?)
				OR (status = 'running' AND locked_at < ?)
			)`

		res, err := r.db.ExecContext(ctx, updateQuery, nowStr, nowStr, id, nowStr, leaseExpiredStr)
		if err != nil {
			continue
		}
		if affected, _ := res.RowsAffected(); affected > 0 {
			if job, err := r.GetByID(ctx, id); err == nil {
				claimed = append(claimed, job)
			}
		}
	}

	return claimed, nil
}

func (r *ProvisioningRepo) MarkCompleted(ctx context.Context, id string) error {
	nowStr := time.Now().Format(time.RFC3339)
	query := `UPDATE provisioning_jobs SET
		status = 'completed', completed_at = ?, last_error = '', updated_at = ?
		WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, nowStr, nowStr, id)
	return err
}

func (r *ProvisioningRepo) MarkFailed(ctx context.Context, id, lastError string, nextAttempt *time.Time, toManualReview bool) error {
	nowStr := time.Now().Format(time.RFC3339)
	status := domain.JobFailed
	if toManualReview {
		status = domain.JobManualReview
	}

	var nextAttemptStr *string
	if nextAttempt != nil {
		s := nextAttempt.Format(time.RFC3339)
		nextAttemptStr = &s
	}

	query := `UPDATE provisioning_jobs SET
		status = ?, last_error = ?, next_attempt_at = ?, updated_at = ?
		WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, string(status), lastError, nextAttemptStr, nowStr, id)
	return err
}

func (r *ProvisioningRepo) ResetForRetry(ctx context.Context, id string) error {
	nowStr := time.Now().Format(time.RFC3339)
	query := `UPDATE provisioning_jobs SET
		status = 'pending', next_attempt_at = ?, locked_at = NULL, updated_at = ?
		WHERE id = ? AND status IN ('failed', 'manual_review')`
	res, err := r.db.ExecContext(ctx, query, nowStr, nowStr, id)
	if err != nil {
		return err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ProvisioningRepo) List(ctx context.Context, status string, limit, offset int) ([]*domain.ProvisioningJob, int64, error) {
	where := ""
	var args []any
	if status != "" {
		where = "WHERE status = ?"
		args = append(args, status)
	}

	countQuery := "SELECT COUNT(*) FROM provisioning_jobs " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	selectQuery := `SELECT id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
		FROM provisioning_jobs ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	listArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*domain.ProvisioningJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, nil
}

func (r *ProvisioningRepo) DeleteCompletedJobs(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM provisioning_jobs WHERE status = 'completed' AND completed_at < ?`
	res, err := r.db.ExecContext(ctx, query, olderThan.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanJob(scanner rowScanner) (*domain.ProvisioningJob, error) {
	j := &domain.ProvisioningJob{}
	var statusStr string
	var orderID, userID, lastError, payloadStr sql.NullString
	var lockedAtStr, nextAttemptStr, completedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := scanner.Scan(
		&j.ID, &j.Key, &orderID, &userID, &j.ServiceType, &j.Action, &statusStr,
		&j.Attempts, &j.MaxAttempts, &lockedAtStr, &nextAttemptStr, &completedAtStr,
		&lastError, &payloadStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	j.Status = domain.JobStatus(statusStr)
	if orderID.Valid {
		j.OrderID = orderID.String
	}
	if userID.Valid {
		j.UserID = userID.String
	}
	if lastError.Valid {
		j.LastError = lastError.String
	}
	if lockedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, lockedAtStr.String); err == nil {
			j.LockedAt = &t
		}
	}
	if nextAttemptStr.Valid {
		if t, err := time.Parse(time.RFC3339, nextAttemptStr.String); err == nil {
			j.NextAttemptAt = &t
		}
	}
	if completedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, completedAtStr.String); err == nil {
			j.CompletedAt = &t
		}
	}
	if payloadStr.Valid && payloadStr.String != "" {
		_ = json.Unmarshal([]byte(payloadStr.String), &j.Payload)
	}
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		j.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		j.UpdatedAt = t
	}

	return j, nil
}
