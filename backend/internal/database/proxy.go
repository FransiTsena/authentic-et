package database

import (
	"context"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

func (m *Manager) createProxyRepositories() *domain.Repositories {
	return &domain.Repositories{
		Payments:     &proxyPaymentRepo{m: m},
		APIKeys:      &proxyAPIKeyRepo{m: m},
		Orders:       &proxyOrderRepo{m: m},
		Provisioning: &proxyProvisioningRepo{m: m},
		Audit:        &proxyAuditRepo{m: m},
		Webhooks:     &proxyWebhookRepo{m: m},
		Users:        &proxyUserRepo{m: m},
		TxManager:    &proxyTxManager{m: m},
	}
}

// --- Payment Proxy ---
type proxyPaymentRepo struct{ m *Manager }

func (p *proxyPaymentRepo) Create(ctx context.Context, pay *domain.Payment) error {
	return p.m.getActive().Payments.Create(ctx, pay)
}
func (p *proxyPaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	return p.m.getActive().Payments.GetByID(ctx, id)
}
func (p *proxyPaymentRepo) GetByProviderAndTxID(ctx context.Context, provider, txID string) (*domain.Payment, error) {
	return p.m.getActive().Payments.GetByProviderAndTxID(ctx, provider, txID)
}
func (p *proxyPaymentRepo) GetByReference(ctx context.Context, refID string) (*domain.Payment, error) {
	return p.m.getActive().Payments.GetByReference(ctx, refID)
}
func (p *proxyPaymentRepo) Update(ctx context.Context, pay *domain.Payment) error {
	return p.m.getActive().Payments.Update(ctx, pay)
}
func (p *proxyPaymentRepo) Claim(ctx context.Context, id, claimedBy string, claimedAt time.Time) (*domain.Payment, error) {
	return p.m.getActive().Payments.Claim(ctx, id, claimedBy, claimedAt)
}
func (p *proxyPaymentRepo) List(ctx context.Context, filter domain.PaymentListFilter) ([]*domain.Payment, int64, error) {
	return p.m.getActive().Payments.List(ctx, filter)
}
func (p *proxyPaymentRepo) GetStats(ctx context.Context, ownerUserID string) (*domain.VerificationStats, error) {
	return p.m.getActive().Payments.GetStats(ctx, ownerUserID)
}
func (p *proxyPaymentRepo) ExpireUnclaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	return p.m.getActive().Payments.ExpireUnclaimedPayments(ctx, olderThan)
}
func (p *proxyPaymentRepo) DeleteFailedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	return p.m.getActive().Payments.DeleteFailedPayments(ctx, olderThan)
}
func (p *proxyPaymentRepo) DeleteOldClaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	return p.m.getActive().Payments.DeleteOldClaimedPayments(ctx, olderThan)
}

// --- APIKey Proxy ---
type proxyAPIKeyRepo struct{ m *Manager }

func (p *proxyAPIKeyRepo) Create(ctx context.Context, k *domain.APIKey) error {
	return p.m.getActive().APIKeys.Create(ctx, k)
}
func (p *proxyAPIKeyRepo) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	return p.m.getActive().APIKeys.GetByID(ctx, id)
}
func (p *proxyAPIKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	return p.m.getActive().APIKeys.GetByKeyHash(ctx, keyHash)
}
func (p *proxyAPIKeyRepo) ListByOwner(ctx context.Context, ownerUserID string) ([]*domain.APIKey, error) {
	return p.m.getActive().APIKeys.ListByOwner(ctx, ownerUserID)
}
func (p *proxyAPIKeyRepo) ListAll(ctx context.Context) ([]*domain.APIKey, error) {
	return p.m.getActive().APIKeys.ListAll(ctx)
}
func (p *proxyAPIKeyRepo) Update(ctx context.Context, k *domain.APIKey) error {
	return p.m.getActive().APIKeys.Update(ctx, k)
}
func (p *proxyAPIKeyRepo) IncrementUsage(ctx context.Context, id string, amount float64, todayDate string) error {
	return p.m.getActive().APIKeys.IncrementUsage(ctx, id, amount, todayDate)
}
func (p *proxyAPIKeyRepo) Revoke(ctx context.Context, id string) error {
	return p.m.getActive().APIKeys.Revoke(ctx, id)
}

// --- Order Proxy ---
type proxyOrderRepo struct{ m *Manager }

func (p *proxyOrderRepo) Create(ctx context.Context, o *domain.Order) error {
	return p.m.getActive().Orders.Create(ctx, o)
}
func (p *proxyOrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	return p.m.getActive().Orders.GetByID(ctx, id)
}
func (p *proxyOrderRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Order, error) {
	return p.m.getActive().Orders.ListByUser(ctx, userID)
}
func (p *proxyOrderRepo) ListAll(ctx context.Context, limit, offset int) ([]*domain.Order, int64, error) {
	return p.m.getActive().Orders.ListAll(ctx, limit, offset)
}
func (p *proxyOrderRepo) Update(ctx context.Context, o *domain.Order) error {
	return p.m.getActive().Orders.Update(ctx, o)
}
func (p *proxyOrderRepo) MarkPaid(ctx context.Context, id, provider, reference string, paidAt time.Time) (*domain.Order, error) {
	return p.m.getActive().Orders.MarkPaid(ctx, id, provider, reference, paidAt)
}

// --- Provisioning Proxy ---
type proxyProvisioningRepo struct{ m *Manager }

func (p *proxyProvisioningRepo) Create(ctx context.Context, job *domain.ProvisioningJob) error {
	return p.m.getActive().Provisioning.Create(ctx, job)
}
func (p *proxyProvisioningRepo) GetByID(ctx context.Context, id string) (*domain.ProvisioningJob, error) {
	return p.m.getActive().Provisioning.GetByID(ctx, id)
}
func (p *proxyProvisioningRepo) GetByKey(ctx context.Context, key string) (*domain.ProvisioningJob, error) {
	return p.m.getActive().Provisioning.GetByKey(ctx, key)
}
func (p *proxyProvisioningRepo) ClaimPendingJobs(ctx context.Context, batchSize int, lockDuration time.Duration) ([]*domain.ProvisioningJob, error) {
	return p.m.getActive().Provisioning.ClaimPendingJobs(ctx, batchSize, lockDuration)
}
func (p *proxyProvisioningRepo) MarkCompleted(ctx context.Context, id string) error {
	return p.m.getActive().Provisioning.MarkCompleted(ctx, id)
}
func (p *proxyProvisioningRepo) MarkFailed(ctx context.Context, id, lastError string, nextAttempt *time.Time, toManualReview bool) error {
	return p.m.getActive().Provisioning.MarkFailed(ctx, id, lastError, nextAttempt, toManualReview)
}
func (p *proxyProvisioningRepo) ResetForRetry(ctx context.Context, id string) error {
	return p.m.getActive().Provisioning.ResetForRetry(ctx, id)
}
func (p *proxyProvisioningRepo) List(ctx context.Context, status string, limit, offset int) ([]*domain.ProvisioningJob, int64, error) {
	return p.m.getActive().Provisioning.List(ctx, status, limit, offset)
}
func (p *proxyProvisioningRepo) DeleteCompletedJobs(ctx context.Context, olderThan time.Time) (int64, error) {
	return p.m.getActive().Provisioning.DeleteCompletedJobs(ctx, olderThan)
}

// --- Audit Proxy ---
type proxyAuditRepo struct{ m *Manager }

func (p *proxyAuditRepo) Record(ctx context.Context, event *domain.AuditEvent) error {
	return p.m.getActive().Audit.Record(ctx, event)
}
func (p *proxyAuditRepo) List(ctx context.Context, resourceType, resourceID string, limit, offset int) ([]*domain.AuditEvent, int64, error) {
	return p.m.getActive().Audit.List(ctx, resourceType, resourceID, limit, offset)
}
func (p *proxyAuditRepo) DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	return p.m.getActive().Audit.DeleteOldAuditLogs(ctx, olderThan)
}

// --- Webhook Proxy ---
type proxyWebhookRepo struct{ m *Manager }

func (p *proxyWebhookRepo) RecordIncoming(ctx context.Context, event *domain.WebhookEvent) error {
	return p.m.getActive().Webhooks.RecordIncoming(ctx, event)
}
func (p *proxyWebhookRepo) GetByEventID(ctx context.Context, provider, eventID string) (*domain.WebhookEvent, error) {
	return p.m.getActive().Webhooks.GetByEventID(ctx, provider, eventID)
}
func (p *proxyWebhookRepo) ListIncoming(ctx context.Context, limit, offset int) ([]*domain.WebhookEvent, int64, error) {
	return p.m.getActive().Webhooks.ListIncoming(ctx, limit, offset)
}
func (p *proxyWebhookRepo) MarkProcessed(ctx context.Context, id string, err error) error {
	return p.m.getActive().Webhooks.MarkProcessed(ctx, id, err)
}

// --- User Proxy ---
type proxyUserRepo struct{ m *Manager }

func (p *proxyUserRepo) Create(ctx context.Context, u *domain.User) error {
	return p.m.getActive().Users.Create(ctx, u)
}
func (p *proxyUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return p.m.getActive().Users.GetByID(ctx, id)
}
func (p *proxyUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return p.m.getActive().Users.GetByEmail(ctx, email)
}
func (p *proxyUserRepo) Update(ctx context.Context, u *domain.User) error {
	return p.m.getActive().Users.Update(ctx, u)
}
func (p *proxyUserRepo) Count(ctx context.Context) (int64, error) {
	return p.m.getActive().Users.Count(ctx)
}

// --- TxManager Proxy ---
type proxyTxManager struct{ m *Manager }

func (p *proxyTxManager) Begin(ctx context.Context) (domain.Tx, error) {
	return p.m.getActive().TxManager.Begin(ctx)
}
func (p *proxyTxManager) WithTx(ctx context.Context, fn func(tx domain.Tx) error) error {
	return p.m.getActive().TxManager.WithTx(ctx, fn)
}
