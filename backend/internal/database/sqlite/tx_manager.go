package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/beecloud/paygate/internal/domain"
)

type TxManager struct {
	db *sql.DB
}

func (m *TxManager) Begin(ctx context.Context) (domain.Tx, error) {
	// Use standard sql.Tx. With WAL mode & MaxOpenConns(1), SQLite serializes write transactions safely.
	sqlTx, err := m.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin sqlite transaction: %w", err)
	}

	return &sqliteTx{
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

type sqliteTx struct {
	tx           *sql.Tx
	payments     *PaymentRepo
	apiKeys      *APIKeyRepo
	orders       *OrderRepo
	provisioning *ProvisioningRepo
	audit        *AuditRepo
	webhooks     *WebhookRepo
}

func (t *sqliteTx) Payments() domain.PaymentRepository {
	return t.payments
}

func (t *sqliteTx) APIKeys() domain.APIKeyRepository {
	return t.apiKeys
}

func (t *sqliteTx) Orders() domain.OrderRepository {
	return t.orders
}

func (t *sqliteTx) Provisioning() domain.ProvisioningRepository {
	return t.provisioning
}

func (t *sqliteTx) Audit() domain.AuditRepository {
	return t.audit
}

func (t *sqliteTx) Webhooks() domain.WebhookRepository {
	return t.webhooks
}

func (t *sqliteTx) Commit() error {
	return t.tx.Commit()
}

func (t *sqliteTx) Rollback() error {
	return t.tx.Rollback()
}
