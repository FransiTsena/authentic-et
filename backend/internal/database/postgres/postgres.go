package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	db *sql.DB
}

func Open(cfg *config.Config) (*DB, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
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

type queryable interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// --- PaymentRepo ---
type PaymentRepo struct {
	db queryable
}

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now

	rawJSON, _ := json.Marshal(p.RawPayload)

	query := `INSERT INTO payments (
		id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.ProviderKey, p.TransactionID, p.ReferenceID, p.ReferenceType, p.Amount, p.Currency, string(p.Status),
		p.Claimed, p.ClaimedAt, p.ClaimedBy, p.APIKeyID, p.APIKeyPrefix, p.APIKeyName, p.OwnerUserID,
		p.PayerIdentifier, p.ReceiverIdentifier, rawJSON, p.ErrorMessage, p.ErrorCode, p.VerifiedAt,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanPayment(row)
}

func (r *PaymentRepo) GetByProviderAndTxID(ctx context.Context, provider, txID string) (*domain.Payment, error) {
	query := `SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments WHERE provider_key = $1 AND transaction_id = $2`
	row := r.db.QueryRowContext(ctx, query, strings.ToLower(provider), txID)
	return scanPayment(row)
}

func (r *PaymentRepo) GetByReference(ctx context.Context, refID string) (*domain.Payment, error) {
	query := `SELECT id, provider_key, transaction_id, reference_id, reference_type, amount, currency, status,
		claimed, claimed_at, claimed_by, api_key_id, api_key_prefix, api_key_name, owner_user_id,
		payer_identifier, receiver_identifier, raw_payload, error_message, error_code, verified_at,
		created_at, updated_at FROM payments WHERE reference_id = $1 ORDER BY created_at DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, refID)
	return scanPayment(row)
}

func (r *PaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	p.UpdatedAt = time.Now().UTC()
	rawJSON, _ := json.Marshal(p.RawPayload)

	query := `UPDATE payments SET
		status = $1, claimed = $2, claimed_at = $3, claimed_by = $4,
		payer_identifier = $5, receiver_identifier = $6, raw_payload = $7,
		error_message = $8, error_code = $9, verified_at = $10, updated_at = $11
		WHERE id = $12`

	res, err := r.db.ExecContext(ctx, query,
		string(p.Status), p.Claimed, p.ClaimedAt, p.ClaimedBy,
		p.PayerIdentifier, p.ReceiverIdentifier, rawJSON,
		p.ErrorMessage, p.ErrorCode, p.VerifiedAt, p.UpdatedAt,
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
	now := time.Now().UTC()

	// Atomic conditional update ensuring transaction is not already claimed
	query := `UPDATE payments SET
		claimed = true, claimed_at = $1, claimed_by = $2, status = 'CLAIMED', updated_at = $3
		WHERE id = $4 AND claimed = false`

	res, err := r.db.ExecContext(ctx, query, claimedAt, claimedBy, now, id)
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
	argIdx := 1

	if filter.ProviderKey != "" {
		conditions = append(conditions, fmt.Sprintf("provider_key = $%d", argIdx))
		args = append(args, strings.ToLower(filter.ProviderKey))
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Claimed != nil {
		conditions = append(conditions, fmt.Sprintf("claimed = $%d", argIdx))
		args = append(args, *filter.Claimed)
		argIdx++
	}
	if filter.APIKeyID != "" {
		conditions = append(conditions, fmt.Sprintf("api_key_id = $%d", argIdx))
		args = append(args, filter.APIKeyID)
		argIdx++
	}
	if filter.OwnerUserID != "" {
		conditions = append(conditions, fmt.Sprintf("owner_user_id = $%d", argIdx))
		args = append(args, filter.OwnerUserID)
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(transaction_id ILIKE $%d OR reference_id ILIKE $%d OR payer_identifier ILIKE $%d)", argIdx, argIdx, argIdx))
		s := "%" + filter.Search + "%"
		args = append(args, s)
		argIdx++
	}
	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, filter.EndDate)
		argIdx++
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
		created_at, updated_at FROM payments %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

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
		where = "WHERE owner_user_id = $1"
		args = append(args, ownerUserID)
	}

	query := fmt.Sprintf(`SELECT
		COUNT(*) as total,
		COALESCE(SUM(CASE WHEN status IN ('VERIFIED', 'CLAIMED') THEN 1 ELSE 0 END), 0) as success,
		COALESCE(SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END), 0) as failed,
		COALESCE(SUM(CASE WHEN status IN ('VERIFIED', 'CLAIMED') THEN amount ELSE 0 END), 0) as total_amount,
		COALESCE(SUM(CASE WHEN claimed = true THEN amount ELSE 0 END), 0) as claimed_amount
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
	query := `UPDATE payments SET status = 'EXPIRED', updated_at = $1 WHERE status = 'PENDING' AND created_at < $2`
	res, err := r.db.ExecContext(ctx, query, time.Now().UTC(), olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PaymentRepo) DeleteFailedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM payments WHERE status IN ('FAILED', 'EXPIRED') AND created_at < $1`
	res, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PaymentRepo) DeleteOldClaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM payments WHERE claimed = true AND created_at < $1`
	res, err := r.db.ExecContext(ctx, query, olderThan)
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
	var claimedAt, verifiedAt *time.Time
	var claimedBy, apiKeyID, apiKeyPrefix, apiKeyName, ownerUserID sql.NullString
	var payerIdentifier, receiverIdentifier, errorMessage, errorCode sql.NullString
	var rawPayloadBytes []byte
	var statusStr string

	err := scanner.Scan(
		&p.ID, &p.ProviderKey, &p.TransactionID, &p.ReferenceID, &p.ReferenceType,
		&p.Amount, &p.Currency, &statusStr, &p.Claimed, &claimedAt, &claimedBy,
		&apiKeyID, &apiKeyPrefix, &apiKeyName, &ownerUserID,
		&payerIdentifier, &receiverIdentifier, &rawPayloadBytes, &errorMessage, &errorCode,
		&verifiedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	p.Status = domain.PaymentStatus(statusStr)
	p.ClaimedAt = claimedAt
	p.VerifiedAt = verifiedAt
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
	if len(rawPayloadBytes) > 0 {
		_ = json.Unmarshal(rawPayloadBytes, &p.RawPayload)
	}

	return p, nil
}
