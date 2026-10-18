package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

// --- APIKeyRepo ---
type APIKeyRepo struct {
	db queryable
}

func (r *APIKeyRepo) Create(ctx context.Context, k *domain.APIKey) error {
	now := time.Now().UTC()
	k.CreatedAt = now
	k.UpdatedAt = now

	permsJSON, _ := json.Marshal(k.Permissions)

	query := `INSERT INTO api_keys (
		id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	_, err := r.db.ExecContext(ctx, query,
		k.ID, k.KeyHash, k.Prefix, k.Name, k.OwnerEmail, k.OwnerUserID, k.Active, string(k.Tier),
		k.MaxRPS, k.MonthlyQuota, k.UsedThisMonth, k.DailyVolumeLimit, k.VerifiedVolumeToday,
		k.LastVolumeResetDate, string(permsJSON), k.LastUsedAt,
		k.CreatedAt, k.UpdatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *APIKeyRepo) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys WHERE key_hash = $1 AND active = true`
	row := r.db.QueryRowContext(ctx, query, keyHash)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) ListByOwner(ctx context.Context, ownerUserID string) ([]*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys WHERE owner_user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) ListAll(ctx context.Context) ([]*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) Update(ctx context.Context, k *domain.APIKey) error {
	k.UpdatedAt = time.Now().UTC()
	permsJSON, _ := json.Marshal(k.Permissions)

	query := `UPDATE api_keys SET
		name = $1, active = $2, tier = $3, max_rps = $4, monthly_quota = $5, used_this_month = $6,
		daily_volume_limit = $7, verified_volume_today = $8, last_volume_reset_date = $9,
		permissions = $10, last_used_at = $11, updated_at = $12
		WHERE id = $13`

	res, err := r.db.ExecContext(ctx, query,
		k.Name, k.Active, string(k.Tier), k.MaxRPS, k.MonthlyQuota, k.UsedThisMonth,
		k.DailyVolumeLimit, k.VerifiedVolumeToday, k.LastVolumeResetDate,
		string(permsJSON), k.LastUsedAt, k.UpdatedAt,
		k.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *APIKeyRepo) IncrementUsage(ctx context.Context, id string, amount float64, todayDate string) error {
	now := time.Now().UTC()
	query := `UPDATE api_keys SET
		used_this_month = used_this_month + 1,
		verified_volume_today = CASE WHEN last_volume_reset_date = $1 THEN verified_volume_today + $2 ELSE $3 END,
		last_volume_reset_date = $4,
		last_used_at = $5,
		updated_at = $6
		WHERE id = $7`

	_, err := r.db.ExecContext(ctx, query, todayDate, amount, amount, todayDate, now, now, id)
	return err
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `UPDATE api_keys SET active = false, updated_at = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanAPIKey(scanner rowScanner) (*domain.APIKey, error) {
	k := &domain.APIKey{}
	var tierStr, permsStr string
	var lastUsedAt *time.Time

	err := scanner.Scan(
		&k.ID, &k.KeyHash, &k.Prefix, &k.Name, &k.OwnerEmail, &k.OwnerUserID, &k.Active, &tierStr,
		&k.MaxRPS, &k.MonthlyQuota, &k.UsedThisMonth, &k.DailyVolumeLimit, &k.VerifiedVolumeToday,
		&k.LastVolumeResetDate, &permsStr, &lastUsedAt, &k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	k.Tier = domain.APIKeyTier(tierStr)
	k.LastUsedAt = lastUsedAt
	_ = json.Unmarshal([]byte(permsStr), &k.Permissions)
	return k, nil
}

// --- OrderRepo ---
type OrderRepo struct {
	db queryable
}

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now

	metaJSON, _ := json.Marshal(o.Metadata)

	query := `INSERT INTO orders (
		id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.db.ExecContext(ctx, query,
		o.ID, o.UserID, o.ServiceType, o.Tier, o.Amount, o.Currency, o.BillingPeriod, string(o.Status),
		o.PaymentProvider, o.PaymentReference, o.PaidAt, metaJSON,
		o.CreatedAt, o.UpdatedAt,
	)
	return err
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
		FROM orders WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanOrder(row)
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Order, error) {
	query := `SELECT id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
		FROM orders WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (r *OrderRepo) ListAll(ctx context.Context, limit, offset int) ([]*domain.Order, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
		FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, nil
}

func (r *OrderRepo) Update(ctx context.Context, o *domain.Order) error {
	o.UpdatedAt = time.Now().UTC()
	metaJSON, _ := json.Marshal(o.Metadata)

	query := `UPDATE orders SET
		status = $1, payment_provider = $2, payment_reference = $3, paid_at = $4, metadata = $5, updated_at = $6
		WHERE id = $7`

	res, err := r.db.ExecContext(ctx, query,
		string(o.Status), o.PaymentProvider, o.PaymentReference, o.PaidAt, metaJSON,
		o.UpdatedAt, o.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OrderRepo) MarkPaid(ctx context.Context, id, provider, reference string, paidAt time.Time) (*domain.Order, error) {
	now := time.Now().UTC()
	query := `UPDATE orders SET
		status = 'paid', payment_provider = $1, payment_reference = $2, paid_at = $3, updated_at = $4
		WHERE id = $5`

	res, err := r.db.ExecContext(ctx, query, provider, reference, paidAt, now, id)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, domain.ErrNotFound
	}

	return r.GetByID(ctx, id)
}

func scanOrder(scanner rowScanner) (*domain.Order, error) {
	o := &domain.Order{}
	var statusStr string
	var paymentProvider, paymentReference sql.NullString
	var paidAt *time.Time
	var metaBytes []byte

	err := scanner.Scan(
		&o.ID, &o.UserID, &o.ServiceType, &o.Tier, &o.Amount, &o.Currency, &o.BillingPeriod,
		&statusStr, &paymentProvider, &paymentReference, &paidAt, &metaBytes,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	o.Status = domain.OrderStatus(statusStr)
	o.PaidAt = paidAt
	if paymentProvider.Valid {
		o.PaymentProvider = paymentProvider.String
	}
	if paymentReference.Valid {
		o.PaymentReference = paymentReference.String
	}
	if len(metaBytes) > 0 {
		_ = json.Unmarshal(metaBytes, &o.Metadata)
	}

	return o, nil
}

// --- ProvisioningRepo ---
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

	query := `INSERT INTO provisioning_jobs (
		id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Key, job.OrderID, job.UserID, job.ServiceType, job.Action, string(job.Status),
		job.Attempts, job.MaxAttempts, job.LockedAt, job.NextAttemptAt, job.CompletedAt,
		job.LastError, payloadJSON, job.CreatedAt, job.UpdatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *ProvisioningRepo) GetByID(ctx context.Context, id string) (*domain.ProvisioningJob, error) {
	query := `SELECT id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
		FROM provisioning_jobs WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanJob(row)
}

func (r *ProvisioningRepo) GetByKey(ctx context.Context, key string) (*domain.ProvisioningJob, error) {
	query := `SELECT id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
		FROM provisioning_jobs WHERE key = $1`
	row := r.db.QueryRowContext(ctx, query, key)
	return scanJob(row)
}

func (r *ProvisioningRepo) ClaimPendingJobs(ctx context.Context, batchSize int, lockDuration time.Duration) ([]*domain.ProvisioningJob, error) {
	now := time.Now().UTC()
	leaseExpired := now.Add(-lockDuration)

	// In PostgreSQL, FOR UPDATE SKIP LOCKED provides concurrent job claiming across replicas
	selectQuery := `SELECT id FROM provisioning_jobs WHERE
		(status = 'pending')
		OR (status = 'failed' AND next_attempt_at <= $1)
		OR (status = 'running' AND locked_at < $2)
		ORDER BY created_at ASC LIMIT $3
		FOR UPDATE SKIP LOCKED`

	rows, err := r.db.QueryContext(ctx, selectQuery, now, leaseExpired, batchSize)
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

	var claimed []*domain.ProvisioningJob
	for _, id := range candidateIDs {
		updateQuery := `UPDATE provisioning_jobs SET
			status = 'running',
			locked_at = $1,
			attempts = attempts + 1,
			updated_at = $2
			WHERE id = $3`

		res, err := r.db.ExecContext(ctx, updateQuery, now, now, id)
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
	now := time.Now().UTC()
	query := `UPDATE provisioning_jobs SET
		status = 'completed', completed_at = $1, last_error = '', updated_at = $2
		WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, now, now, id)
	return err
}

func (r *ProvisioningRepo) MarkFailed(ctx context.Context, id, lastError string, nextAttempt *time.Time, toManualReview bool) error {
	now := time.Now().UTC()
	status := domain.JobFailed
	if toManualReview {
		status = domain.JobManualReview
	}

	query := `UPDATE provisioning_jobs SET
		status = $1, last_error = $2, next_attempt_at = $3, updated_at = $4
		WHERE id = $5`

	_, err := r.db.ExecContext(ctx, query, string(status), lastError, nextAttempt, now, id)
	return err
}

func (r *ProvisioningRepo) ResetForRetry(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `UPDATE provisioning_jobs SET
		status = 'pending', next_attempt_at = $1, locked_at = NULL, updated_at = $2
		WHERE id = $3 AND status IN ('failed', 'manual_review')`
	res, err := r.db.ExecContext(ctx, query, now, now, id)
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
		where = "WHERE status = $1"
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

	argIdx := len(args) + 1
	selectQuery := fmt.Sprintf(`SELECT id, key, order_id, user_id, service_type, action, status, attempts, max_attempts,
		locked_at, next_attempt_at, completed_at, last_error, payload, created_at, updated_at
		FROM provisioning_jobs %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

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
	query := `DELETE FROM provisioning_jobs WHERE status = 'completed' AND completed_at < $1`
	res, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanJob(scanner rowScanner) (*domain.ProvisioningJob, error) {
	j := &domain.ProvisioningJob{}
	var statusStr string
	var orderID, userID, lastError sql.NullString
	var lockedAt, nextAttemptAt, completedAt *time.Time
	var payloadBytes []byte

	err := scanner.Scan(
		&j.ID, &j.Key, &orderID, &userID, &j.ServiceType, &j.Action, &statusStr,
		&j.Attempts, &j.MaxAttempts, &lockedAt, &nextAttemptAt, &completedAt,
		&lastError, &payloadBytes, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	j.Status = domain.JobStatus(statusStr)
	j.LockedAt = lockedAt
	j.NextAttemptAt = nextAttemptAt
	j.CompletedAt = completedAt
	if orderID.Valid {
		j.OrderID = orderID.String
	}
	if userID.Valid {
		j.UserID = userID.String
	}
	if lastError.Valid {
		j.LastError = lastError.String
	}
	if len(payloadBytes) > 0 {
		_ = json.Unmarshal(payloadBytes, &j.Payload)
	}
	return j, nil
}

// --- AuditRepo ---
type AuditRepo struct {
	db queryable
}

func (r *AuditRepo) Record(ctx context.Context, e *domain.AuditEvent) error {
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}

	detailsJSON, _ := json.Marshal(e.Details)

	query := `INSERT INTO audit_events (
		id, action, actor_type, actor_id, resource_id, resource_type,
		details, ip_address, correlation_id, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		e.ID, e.Action, e.ActorType, e.ActorID, e.ResourceID, e.ResourceType,
		detailsJSON, e.IPAddress, e.CorrelationID, e.CreatedAt,
	)
	return err
}

func (r *AuditRepo) List(ctx context.Context, resourceType, resourceID string, limit, offset int) ([]*domain.AuditEvent, int64, error) {
	where := ""
	var args []any
	if resourceType != "" && resourceID != "" {
		where = "WHERE resource_type = $1 AND resource_id = $2"
		args = append(args, resourceType, resourceID)
	} else if resourceType != "" {
		where = "WHERE resource_type = $1"
		args = append(args, resourceType)
	}

	countQuery := "SELECT COUNT(*) FROM audit_events " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	argIdx := len(args) + 1
	selectQuery := fmt.Sprintf(`SELECT id, action, actor_type, actor_id, resource_id, resource_type,
		details, ip_address, correlation_id, created_at
		FROM audit_events %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	listArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		e := &domain.AuditEvent{}
		var detailsBytes []byte
		var ip, corr sql.NullString

		if err := rows.Scan(
			&e.ID, &e.Action, &e.ActorType, &e.ActorID, &e.ResourceID, &e.ResourceType,
			&detailsBytes, &ip, &corr, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}

		if len(detailsBytes) > 0 {
			_ = json.Unmarshal(detailsBytes, &e.Details)
		}
		if ip.Valid {
			e.IPAddress = ip.String
		}
		if corr.Valid {
			e.CorrelationID = corr.String
		}
		events = append(events, e)
	}

	return events, total, nil
}

func (r *AuditRepo) DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM audit_events WHERE created_at < $1`
	res, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// --- WebhookRepo ---
type WebhookRepo struct {
	db queryable
}

func (r *WebhookRepo) RecordIncoming(ctx context.Context, e *domain.WebhookEvent) error {
	now := time.Now().UTC()
	e.CreatedAt = now
	rawJSON, _ := json.Marshal(e.RawPayload)

	query := `INSERT INTO webhook_events (
		id, provider, event_type, event_id, payment_reference, raw_payload,
		signature_valid, processed, error_message, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		e.ID, e.Provider, e.EventType, e.EventID, e.PaymentReference, rawJSON,
		e.SignatureValid, e.Processed, e.ErrorMessage, e.CreatedAt,
	)
	return err
}

func (r *WebhookRepo) GetByEventID(ctx context.Context, provider, eventID string) (*domain.WebhookEvent, error) {
	query := `SELECT id, provider, event_type, event_id, payment_reference, raw_payload,
		signature_valid, processed, error_message, created_at
		FROM webhook_events WHERE provider = $1 AND event_id = $2`
	row := r.db.QueryRowContext(ctx, query, provider, eventID)
	return scanWebhook(row)
}

func (r *WebhookRepo) ListIncoming(ctx context.Context, limit, offset int) ([]*domain.WebhookEvent, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_events").Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, provider, event_type, event_id, payment_reference, raw_payload,
		signature_valid, processed, error_message, created_at
		FROM webhook_events ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.WebhookEvent
	for rows.Next() {
		e, err := scanWebhook(rows)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}

	return events, total, nil
}

func (r *WebhookRepo) MarkProcessed(ctx context.Context, id string, processErr error) error {
	processed := true
	errMsg := ""
	if processErr != nil {
		processed = false
		errMsg = processErr.Error()
	}

	query := `UPDATE webhook_events SET processed = $1, error_message = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, processed, errMsg, id)
	return err
}

func scanWebhook(scanner rowScanner) (*domain.WebhookEvent, error) {
	e := &domain.WebhookEvent{}
	var eventID, paymentRef, errMsg sql.NullString
	var rawBytes []byte

	err := scanner.Scan(
		&e.ID, &e.Provider, &e.EventType, &eventID, &paymentRef, &rawBytes,
		&e.SignatureValid, &e.Processed, &errMsg, &e.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if eventID.Valid {
		e.EventID = eventID.String
	}
	if paymentRef.Valid {
		e.PaymentReference = paymentRef.String
	}
	if errMsg.Valid {
		e.ErrorMessage = errMsg.String
	}
	if len(rawBytes) > 0 {
		_ = json.Unmarshal(rawBytes, &e.RawPayload)
	}
	return e, nil
}

// --- UserRepo ---
type UserRepo struct {
	db queryable
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `INSERT INTO users (
		id, email, password_hash, name, role, active, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, strings.ToLower(u.Email), u.PasswordHash, u.Name, string(u.Role),
		u.Active, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, role, active, created_at, updated_at
		FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, role, active, created_at, updated_at
		FROM users WHERE email = $1`
	row := r.db.QueryRowContext(ctx, query, strings.ToLower(email))
	return scanUser(row)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	u.UpdatedAt = time.Now().UTC()
	query := `UPDATE users SET
		password_hash = $1, name = $2, role = $3, active = $4, updated_at = $5
		WHERE id = $6`

	res, err := r.db.ExecContext(ctx, query,
		u.PasswordHash, u.Name, string(u.Role), u.Active, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func scanUser(scanner rowScanner) (*domain.User, error) {
	u := &domain.User{}
	var roleStr string

	err := scanner.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &roleStr,
		&u.Active, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	u.Role = domain.UserRole(roleStr)
	return u, nil
}

// --- TxManager ---
type TxManager struct {
	db *sql.DB
}

func (m *TxManager) Begin(ctx context.Context) (domain.Tx, error) {
	sqlTx, err := m.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin postgres transaction: %w", err)
	}

	return &postgresTx{
		tx: sqlTx,
		payments: &PaymentRepo{db: sqlTx},
		apiKeys: &APIKeyRepo{db: sqlTx},
		orders: &OrderRepo{db: sqlTx},
		provisioning: &ProvisioningRepo{db: sqlTx},
		audit: &AuditRepo{db: sqlTx},
		webhooks: &WebhookRepo{db: sqlTx},
	}, nil
}

func (m *TxManager) WithTx(ctx context.Context, fn func(tx domain.Tx) error) error {
	tx, err := m.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

type postgresTx struct {
	tx           *sql.Tx
	payments     *PaymentRepo
	apiKeys      *APIKeyRepo
	orders       *OrderRepo
	provisioning *ProvisioningRepo
	audit        *AuditRepo
	webhooks     *WebhookRepo
}

func (t *postgresTx) Payments() domain.PaymentRepository {
	return t.payments
}

func (t *postgresTx) APIKeys() domain.APIKeyRepository {
	return t.apiKeys
}

func (t *postgresTx) Orders() domain.OrderRepository {
	return t.orders
}

func (t *postgresTx) Provisioning() domain.ProvisioningRepository {
	return t.provisioning
}

func (t *postgresTx) Audit() domain.AuditRepository {
	return t.audit
}

func (t *postgresTx) Webhooks() domain.WebhookRepository {
	return t.webhooks
}

func (t *postgresTx) Commit() error {
	return t.tx.Commit()
}

func (t *postgresTx) Rollback() error {
	return t.tx.Rollback()
}
