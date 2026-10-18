package domain

import (
	"context"
)

type Tx interface {
	Payments() PaymentRepository
	APIKeys() APIKeyRepository
	Orders() OrderRepository
	Provisioning() ProvisioningRepository
	Audit() AuditRepository
	Webhooks() WebhookRepository
	Commit() error
	Rollback() error
}

type TxManager interface {
	Begin(ctx context.Context) (Tx, error)
	WithTx(ctx context.Context, fn func(tx Tx) error) error
}

type Repositories struct {
	Payments     PaymentRepository
	APIKeys      APIKeyRepository
	Orders       OrderRepository
	Provisioning ProvisioningRepository
	Audit        AuditRepository
	Webhooks     WebhookRepository
	Users        UserRepository
	TxManager    TxManager
}
