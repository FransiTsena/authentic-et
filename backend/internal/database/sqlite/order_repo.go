package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type OrderRepo struct {
	db queryable
}

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	now := time.Now()
	o.CreatedAt = now
	o.UpdatedAt = now

	metaJSON, _ := json.Marshal(o.Metadata)
	var paidAtStr *string
	if o.PaidAt != nil {
		s := o.PaidAt.Format(time.RFC3339)
		paidAtStr = &s
	}

	query := `INSERT INTO orders (
		id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		o.ID, o.UserID, o.ServiceType, o.Tier, o.Amount, o.Currency, o.BillingPeriod, string(o.Status),
		o.PaymentProvider, o.PaymentReference, paidAtStr, string(metaJSON),
		o.CreatedAt.Format(time.RFC3339), o.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
		FROM orders WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanOrder(row)
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Order, error) {
	query := `SELECT id, user_id, service_type, tier, amount, currency, billing_period, status,
		payment_provider, payment_reference, paid_at, metadata, created_at, updated_at
		FROM orders WHERE user_id = ? ORDER BY created_at DESC`
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
		FROM orders ORDER BY created_at DESC LIMIT ? OFFSET ?`
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
	o.UpdatedAt = time.Now()
	metaJSON, _ := json.Marshal(o.Metadata)

	var paidAtStr *string
	if o.PaidAt != nil {
		s := o.PaidAt.Format(time.RFC3339)
		paidAtStr = &s
	}

	query := `UPDATE orders SET
		status = ?, payment_provider = ?, payment_reference = ?, paid_at = ?, metadata = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		string(o.Status), o.PaymentProvider, o.PaymentReference, paidAtStr, string(metaJSON),
		o.UpdatedAt.Format(time.RFC3339), o.ID,
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
	paidAtStr := paidAt.Format(time.RFC3339)
	nowStr := time.Now().Format(time.RFC3339)

	query := `UPDATE orders SET
		status = 'paid', payment_provider = ?, payment_reference = ?, paid_at = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query, provider, reference, paidAtStr, nowStr, id)
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
	var paymentProvider, paymentReference, paidAtStr, metadataStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := scanner.Scan(
		&o.ID, &o.UserID, &o.ServiceType, &o.Tier, &o.Amount, &o.Currency, &o.BillingPeriod,
		&statusStr, &paymentProvider, &paymentReference, &paidAtStr, &metadataStr,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	o.Status = domain.OrderStatus(statusStr)
	if paymentProvider.Valid {
		o.PaymentProvider = paymentProvider.String
	}
	if paymentReference.Valid {
		o.PaymentReference = paymentReference.String
	}
	if paidAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, paidAtStr.String); err == nil {
			o.PaidAt = &t
		}
	}
	if metadataStr.Valid && metadataStr.String != "" {
		_ = json.Unmarshal([]byte(metadataStr.String), &o.Metadata)
	}
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		o.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		o.UpdatedAt = t
	}

	return o, nil
}
