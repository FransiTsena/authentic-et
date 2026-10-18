package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/domain"
	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func Open(cfg *config.Config) (*DB, error) {
	dir := filepath.Dir(cfg.SQLitePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite directory: %w", err)
	}

	// Enable WAL mode, foreign keys, and busy timeout for concurrent safety
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", cfg.SQLitePath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// SQLite single-writer optimization
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) DB() *sql.DB {
	return d.db
}

func (d *DB) Migrate(ctx context.Context, migrationSQL string) error {
	_, err := d.db.ExecContext(ctx, migrationSQL)
	return err
}

func (d *DB) Repositories() *domain.Repositories {
	return &domain.Repositories{
		Payments:     &PaymentRepo{db: d.db},
		APIKeys:      &APIKeyRepo{db: d.db},
		Orders:       &OrderRepo{db: d.db},
		Provisioning: &ProvisioningRepo{db: d.db},
		Audit:        &AuditRepo{db: d.db},
		Webhooks:     &WebhookRepo{db: d.db},
		Users:        &UserRepo{db: d.db},
		TxManager:    &TxManager{db: d.db},
	}
}

// --- PaymentRepo ---
type PaymentRepo struct {
	db queryable
}

type queryable interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	rawJSON, _ := json.Marshal(p.RawPayload)
	var claimedInt int
	if p.Claimed {
		claimedInt = 1
	}

	var claimedAtStr *string
	if p.ClaimedAt != nil {
		s := p.ClaimedAt.Format(time.RFC3339)
		claimedAtStr = &s
	}

	var verifiedAtStr *string
	if p.VerifiedAt != nil {
		s := p.VerifiedAt.Format(time.RFC3339)
		verifiedAtStr = &s
	}

	query := `INSERT INTO payments (
		id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.ProviderKey, p.TransactionID, p.ReferenceID, p.ReferenceType, p.Amount, p.Currency, string(p.Status),
		claimedInt, claimedAtStr, p.ClaimedBy, p.APIKeyID, p.APIKeyPrefix, p.APIKeyName, p.OwnerUserID,
		p.PayerIdentifier, p.ReceiverIdentifier, string(rawJSON), p.ErrorMessage, p.ErrorCode, verifiedAtStr,
		p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanPayment(row)
}

func (r *PaymentRepo) GetByProviderAndTxID(ctx context.Context, provider, txID string) (*domain.Payment, error) {
	query := `SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments WHERE provider_key = ? AND transaction_id = ?`
	row := r.db.QueryRowContext(ctx, query, strings.ToLower(provider), txID)
	return scanPayment(row)
}

func (r *PaymentRepo) GetByReference(ctx context.Context, refID string) (*domain.Payment, error) {
	query := `SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments WHERE reference_id = ? ORDER BY created_at DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, refID)
	return scanPayment(row)
}

func (r *PaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	p.UpdatedAt = time.Now()
	rawJSON, _ := json.Marshal(p.RawPayload)
	var claimedInt int
	if p.Claimed {
		claimedInt = 1
	}

	var claimedAtStr *string
	if p.ClaimedAt != nil {
		s := p.ClaimedAt.Format(time.RFC3339)
		claimedAtStr = &s
	}

	var verifiedAtStr *string
	if p.VerifiedAt != nil {
		s := p.VerifiedAt.Format(time.RFC3339)
		verifiedAtStr = &s
	}

	query := `UPDATE payments SET
		status = ?, claimed = ?, claimed_at = ?, claimed_by = ?,
		payer_identifier = ?, receiver_identifier = ?, raw_payload = ?,
		error_message = ?, error_code = ?, verified_at = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		string(p.Status), claimedInt, claimedAtStr, p.ClaimedBy,
		p.PayerIdentifier, p.ReceiverIdentifier, string(rawJSON),
		p.ErrorMessage, p.ErrorCode, verifiedAtStr, p.UpdatedAt.Format(time.RFC3339),
		p.ID,
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

func (r *PaymentRepo) Claim(ctx context.Context, id, claimedBy string, claimedAt time.Time) (*domain.Payment, error) {
	claimedAtStr := claimedAt.Format(time.RFC3339)
	nowStr := time.Now().Format(time.RFC3339)

	// Atomic conditional update ensuring transaction has not already been claimed
	query := `UPDATE payments SET
		claimed = 1, claimed_at = ?, claimed_by = ?, status = 'CLAIMED', updated_at = ?
		WHERE id = ? AND claimed = 0`

	res, err := r.db.ExecContext(ctx, query, claimedAtStr, claimedBy, nowStr, id)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		p, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if p.Claimed {
			return p, domain.ErrTransactionAlreadyClaimed
		}
		return nil, domain.ErrConcurrentModification
	}

	return r.GetByID(ctx, id)
}

func (r *PaymentRepo) List(ctx context.Context, filter domain.PaymentListFilter) ([]*domain.Payment, int64, error) {
	var conditions []string
	var args []any

	if filter.ProviderKey != "" {
		conditions = append(conditions, "provider_key = ?")
		args = append(args, strings.ToLower(filter.ProviderKey))
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.Claimed != nil {
		if *filter.Claimed {
			conditions = append(conditions, "claimed = 1")
		} else {
			conditions = append(conditions, "claimed = 0")
		}
	}
	if filter.APIKeyID != "" {
		conditions = append(conditions, "api_key_id = ?")
		args = append(args, filter.APIKeyID)
	}
	if filter.OwnerUserID != "" {
		conditions = append(conditions, "owner_user_id = ?")
		args = append(args, filter.OwnerUserID)
	}
	if filter.Search != "" {
		conditions = append(conditions, "(transaction_id LIKE ? OR reference_id LIKE ? OR payer_identifier LIKE ?)")
		s := "%" + filter.Search + "%"
		args = append(args, s, s, s)
	}
	if filter.StartDate != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, filter.StartDate.Format(time.RFC3339))
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, filter.EndDate.Format(time.RFC3339))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM payments %s", whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset

	selectQuery := fmt.Sprintf(`SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, whereClause)

	listArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, 0, err
		}
		payments = append(payments, p)
	}

	return payments, total, nil
}

func (r *PaymentRepo) GetStats(ctx context.Context, ownerUserID string) (*domain.VerificationStats, error) {
	where := ""
	var args []any
	if ownerUserID != "" {
		where = "WHERE owner_user_id = ?"
		args = append(args, ownerUserID)
	}

	query := fmt.Sprintf(`SELECT
		COUNT(*) as total,
		COALESCE(SUM(CASE WHEN status IN ('VERIFIED', 'CLAIMED') THEN 1 ELSE 0 END), 0) as success,
		COALESCE(SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END), 0) as failed,
		COALESCE(SUM(CASE WHEN status IN ('VERIFIED', 'CLAIMED') THEN amount ELSE 0 END), 0) as total_amount,
		COALESCE(SUM(CASE WHEN claimed = 1 THEN amount ELSE 0 END), 0) as claimed_amount
		FROM payments %s`, where)

	stats := &domain.VerificationStats{
		Providers: make(map[string]int64),
	}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalVerifications,
		&stats.SuccessfulVerifications,
		&stats.FailedVerifications,
		&stats.TotalVerifiedAmount,
		&stats.TotalClaimedAmount,
	)
	if err != nil {
		return nil, err
	}

	if stats.TotalVerifications > 0 {
		stats.SuccessRatePercent = (float64(stats.SuccessfulVerifications) / float64(stats.TotalVerifications)) * 100.0
	}

	providerQuery := fmt.Sprintf(`SELECT provider_key, COUNT(*) FROM payments %s GROUP BY provider_key`, where)
	rows, err := r.db.QueryContext(ctx, providerQuery, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var c int64
			if err := rows.Scan(&k, &c); err == nil {
				stats.Providers[k] = c
			}
		}
	}

	return stats, nil
}

func (r *PaymentRepo) ExpireUnclaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `UPDATE payments SET status = 'EXPIRED', updated_at = ? WHERE status = 'PENDING' AND created_at < ?`
	nowStr := time.Now().UTC().Format(time.RFC3339)
	olderThanStr := olderThan.Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, query, nowStr, olderThanStr)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PaymentRepo) DeleteFailedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM payments WHERE status IN ('FAILED', 'EXPIRED') AND created_at < ?`
	res, err := r.db.ExecContext(ctx, query, olderThan.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PaymentRepo) DeleteOldClaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM payments WHERE claimed = 1 AND created_at < ?`
	res, err := r.db.ExecContext(ctx, query, olderThan.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPayment(scanner rowScanner) (*domain.Payment, error) {
	p := &domain.Payment{}
	var claimedInt int
	var claimedAtStr, verifiedAtStr, rawPayloadStr sql.NullString
	var claimedBy, apiKeyID, apiKeyPrefix, apiKeyName, ownerUserID sql.NullString
	var payerIdentifier, receiverIdentifier, errorMessage, errorCode sql.NullString
	var createdAtStr, updatedAtStr string
	var statusStr string

	err := scanner.Scan(
		&p.ID, &p.ProviderKey, &p.TransactionID, &p.ReferenceID, &p.ReferenceType,
		&p.Amount, &p.Currency, &statusStr, &claimedInt, &claimedAtStr, &claimedBy,
		&apiKeyID, &apiKeyPrefix, &apiKeyName, &ownerUserID,
		&payerIdentifier, &receiverIdentifier, &rawPayloadStr, &errorMessage, &errorCode,
		&verifiedAtStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	p.Status = domain.PaymentStatus(statusStr)
	p.Claimed = (claimedInt == 1)
	if claimedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, claimedAtStr.String); err == nil {
			p.ClaimedAt = &t
		}
	}
	if claimedBy.Valid {
		p.ClaimedBy = claimedBy.String
	}
	if apiKeyID.Valid {
		p.APIKeyID = apiKeyID.String
	}
	if apiKeyPrefix.Valid {
		p.APIKeyPrefix = apiKeyPrefix.String
	}
	if apiKeyName.Valid {
		p.APIKeyName = apiKeyName.String
	}
	if ownerUserID.Valid {
		p.OwnerUserID = ownerUserID.String
	}
	if payerIdentifier.Valid {
		p.PayerIdentifier = payerIdentifier.String
	}
	if receiverIdentifier.Valid {
		p.ReceiverIdentifier = receiverIdentifier.String
	}
	if errorMessage.Valid {
		p.ErrorMessage = errorMessage.String
	}
	if errorCode.Valid {
		p.ErrorCode = errorCode.String
	}
	if verifiedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, verifiedAtStr.String); err == nil {
			p.VerifiedAt = &t
		}
	}
	if rawPayloadStr.Valid && rawPayloadStr.String != "" {
		_ = json.Unmarshal([]byte(rawPayloadStr.String), &p.RawPayload)
	}
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		p.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		p.UpdatedAt = t
	}

	return p, nil
}
