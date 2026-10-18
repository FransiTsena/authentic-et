package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type Storage struct {
	mu           sync.RWMutex
	payments     map[string]*domain.Payment
	paymentsByTx map[string]string // tx_id -> payment_id
	apiKeys      map[string]*domain.APIKey
	apiKeysByHash map[string]string // key_hash -> id
	orders       map[string]*domain.Order
	jobs         map[string]*domain.ProvisioningJob
	auditEvents  []*domain.AuditEvent
	webhooks     map[string]*domain.WebhookEvent
	users        map[string]*domain.User
	usersByEmail map[string]string // email -> user_id
}

func New() *Storage {
	return &Storage{
		payments:      make(map[string]*domain.Payment),
		paymentsByTx:  make(map[string]string),
		apiKeys:       make(map[string]*domain.APIKey),
		apiKeysByHash: make(map[string]string),
		orders:        make(map[string]*domain.Order),
		jobs:          make(map[string]*domain.ProvisioningJob),
		auditEvents:   make([]*domain.AuditEvent, 0),
		webhooks:      make(map[string]*domain.WebhookEvent),
		users:         make(map[string]*domain.User),
		usersByEmail:  make(map[string]string),
	}
}

func (s *Storage) Repositories() *domain.Repositories {
	return &domain.Repositories{
		Payments:     &PaymentRepo{s: s},
		APIKeys:      &APIKeyRepo{s: s},
		Orders:       &OrderRepo{s: s},
		Provisioning: &ProvisioningRepo{s: s},
		Audit:        &AuditRepo{s: s},
		Webhooks:     &WebhookRepo{s: s},
		Users:        &UserRepo{s: s},
		TxManager:    &TxManager{s: s},
	}
}

// --- PaymentRepo ---
type PaymentRepo struct {
	s *Storage
}

func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	lookupKey := fmt.Sprintf("%s:%s", strings.ToLower(p.ProviderKey), p.TransactionID)
	if _, exists := r.s.paymentsByTx[lookupKey]; exists {
		return domain.ErrDuplicateKey
	}

	clone := *p
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = time.Now().UTC()
	}
	clone.UpdatedAt = time.Now().UTC()

	r.s.payments[clone.ID] = &clone
	r.s.paymentsByTx[lookupKey] = clone.ID
	return nil
}

func (r *PaymentRepo) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	p, ok := r.s.payments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *p
	return &clone, nil
}

func (r *PaymentRepo) GetByProviderAndTxID(ctx context.Context, provider, txID string) (*domain.Payment, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	lookupKey := fmt.Sprintf("%s:%s", strings.ToLower(provider), txID)
	id, ok := r.s.paymentsByTx[lookupKey]
	if !ok {
		return nil, domain.ErrNotFound
	}
	p := r.s.payments[id]
	clone := *p
	return &clone, nil
}

func (r *PaymentRepo) GetByReference(ctx context.Context, refID string) (*domain.Payment, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	for _, p := range r.s.payments {
		if p.ReferenceID == refID {
			clone := *p
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *PaymentRepo) Update(ctx context.Context, p *domain.Payment) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	existing, ok := r.s.payments[p.ID]
	if !ok {
		return domain.ErrNotFound
	}

	clone := *p
	clone.UpdatedAt = time.Now().UTC()
	*existing = clone
	return nil
}

func (r *PaymentRepo) Claim(ctx context.Context, id, claimedBy string, claimedAt time.Time) (*domain.Payment, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	p, ok := r.s.payments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if p.Claimed {
		return p, domain.ErrTransactionAlreadyClaimed
	}

	p.Claimed = true
	p.ClaimedAt = &claimedAt
	p.ClaimedBy = claimedBy
	p.Status = domain.PaymentStatusClaimed
	p.UpdatedAt = time.Now().UTC()

	clone := *p
	return &clone, nil
}

func (r *PaymentRepo) List(ctx context.Context, filter domain.PaymentListFilter) ([]*domain.Payment, int64, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var matched []*domain.Payment
	for _, p := range r.s.payments {
		if filter.ProviderKey != "" && !strings.EqualFold(p.ProviderKey, filter.ProviderKey) {
			continue
		}
		if filter.Status != "" && string(p.Status) != filter.Status {
			continue
		}
		if filter.Claimed != nil && p.Claimed != *filter.Claimed {
			continue
		}
		if filter.APIKeyID != "" && p.APIKeyID != filter.APIKeyID {
			continue
		}
		if filter.OwnerUserID != "" && p.OwnerUserID != filter.OwnerUserID {
			continue
		}
		if filter.Search != "" {
			term := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(p.TransactionID), term) &&
				!strings.Contains(strings.ToLower(p.ReferenceID), term) &&
				!strings.Contains(strings.ToLower(p.PayerIdentifier), term) {
				continue
			}
		}
		clone := *p
		matched = append(matched, &clone)
	}

	total := int64(len(matched))
	start := filter.Offset
	if start > len(matched) {
		return []*domain.Payment{}, total, nil
	}
	end := start + filter.Limit
	if filter.Limit <= 0 || end > len(matched) {
		end = len(matched)
	}

	return matched[start:end], total, nil
}

func (r *PaymentRepo) GetStats(ctx context.Context, ownerUserID string) (*domain.VerificationStats, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	stats := &domain.VerificationStats{
		Providers: make(map[string]int64),
	}

	for _, p := range r.s.payments {
		if ownerUserID != "" && p.OwnerUserID != ownerUserID {
			continue
		}
		stats.TotalVerifications++
		stats.Providers[p.ProviderKey]++
		if p.Status == domain.PaymentStatusVerified || p.Status == domain.PaymentStatusClaimed {
			stats.SuccessfulVerifications++
			stats.TotalVerifiedAmount += p.Amount
		} else if p.Status == domain.PaymentStatusFailed {
			stats.FailedVerifications++
		}
		if p.Claimed {
			stats.TotalClaimedAmount += p.Amount
		}
	}

	if stats.TotalVerifications > 0 {
		stats.SuccessRatePercent = (float64(stats.SuccessfulVerifications) / float64(stats.TotalVerifications)) * 100.0
	}
	return stats, nil
}

func (r *PaymentRepo) ExpireUnclaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	var count int64
	for _, p := range r.s.payments {
		if p.Status == domain.PaymentStatusPending && p.CreatedAt.Before(olderThan) {
			p.Status = domain.PaymentStatusExpired
			p.UpdatedAt = time.Now().UTC()
			count++
		}
	}
	return count, nil
}

func (r *PaymentRepo) DeleteFailedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	var count int64
	for id, p := range r.s.payments {
		if (p.Status == domain.PaymentStatusFailed || p.Status == domain.PaymentStatusExpired) && p.CreatedAt.Before(olderThan) {
			lookupKey := fmt.Sprintf("%s:%s", strings.ToLower(p.ProviderKey), p.TransactionID)
			delete(r.s.paymentsByTx, lookupKey)
			delete(r.s.payments, id)
			count++
		}
	}
	return count, nil
}

func (r *PaymentRepo) DeleteOldClaimedPayments(ctx context.Context, olderThan time.Time) (int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	var count int64
	for id, p := range r.s.payments {
		if p.Claimed && p.CreatedAt.Before(olderThan) {
			lookupKey := fmt.Sprintf("%s:%s", strings.ToLower(p.ProviderKey), p.TransactionID)
			delete(r.s.paymentsByTx, lookupKey)
			delete(r.s.payments, id)
			count++
		}
	}
	return count, nil
}

// --- APIKeyRepo ---
type APIKeyRepo struct {
	s *Storage
}

func (r *APIKeyRepo) Create(ctx context.Context, k *domain.APIKey) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	clone := *k
	now := time.Now().UTC()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.s.apiKeys[clone.ID] = &clone
	r.s.apiKeysByHash[clone.KeyHash] = clone.ID
	return nil
}

func (r *APIKeyRepo) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	k, ok := r.s.apiKeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *k
	return &clone, nil
}

func (r *APIKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	id, ok := r.s.apiKeysByHash[keyHash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	k := r.s.apiKeys[id]
	clone := *k
	return &clone, nil
}

func (r *APIKeyRepo) ListByOwner(ctx context.Context, ownerUserID string) ([]*domain.APIKey, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.APIKey
	for _, k := range r.s.apiKeys {
		if k.OwnerUserID == ownerUserID {
			clone := *k
			res = append(res, &clone)
		}
	}
	return res, nil
}

func (r *APIKeyRepo) ListAll(ctx context.Context) ([]*domain.APIKey, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.APIKey
	for _, k := range r.s.apiKeys {
		clone := *k
		res = append(res, &clone)
	}
	return res, nil
}

func (r *APIKeyRepo) Update(ctx context.Context, k *domain.APIKey) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	existing, ok := r.s.apiKeys[k.ID]
	if !ok {
		return domain.ErrNotFound
	}
	clone := *k
	clone.UpdatedAt = time.Now().UTC()
	*existing = clone
	return nil
}

func (r *APIKeyRepo) IncrementUsage(ctx context.Context, id string, amount float64, todayDate string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	k, ok := r.s.apiKeys[id]
	if !ok {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	k.LastUsedAt = &now
	k.UsedThisMonth++
	if k.LastVolumeResetDate == todayDate {
		k.VerifiedVolumeToday += amount
	} else {
		k.LastVolumeResetDate = todayDate
		k.VerifiedVolumeToday = amount
	}
	return nil
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	k, ok := r.s.apiKeys[id]
	if !ok {
		return domain.ErrNotFound
	}
	k.Active = false
	k.UpdatedAt = time.Now().UTC()
	return nil
}

// --- OrderRepo ---
type OrderRepo struct {
	s *Storage
}

func (r *OrderRepo) Create(ctx context.Context, o *domain.Order) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	clone := *o
	now := time.Now().UTC()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.s.orders[clone.ID] = &clone
	return nil
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	o, ok := r.s.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *o
	return &clone, nil
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Order, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.Order
	for _, o := range r.s.orders {
		if o.UserID == userID {
			clone := *o
			res = append(res, &clone)
		}
	}
	return res, nil
}

func (r *OrderRepo) ListAll(ctx context.Context, limit, offset int) ([]*domain.Order, int64, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.Order
	for _, o := range r.s.orders {
		clone := *o
		res = append(res, &clone)
	}
	total := int64(len(res))
	if offset > len(res) {
		return []*domain.Order{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(res) {
		end = len(res)
	}
	return res[offset:end], total, nil
}

func (r *OrderRepo) Update(ctx context.Context, o *domain.Order) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	existing, ok := r.s.orders[o.ID]
	if !ok {
		return domain.ErrNotFound
	}
	clone := *o
	clone.UpdatedAt = time.Now().UTC()
	*existing = clone
	return nil
}

func (r *OrderRepo) MarkPaid(ctx context.Context, id, provider, reference string, paidAt time.Time) (*domain.Order, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	o, ok := r.s.orders[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	o.Status = domain.OrderStatusPaid
	o.PaymentProvider = provider
	o.PaymentReference = reference
	o.PaidAt = &paidAt
	o.UpdatedAt = time.Now().UTC()
	clone := *o
	return &clone, nil
}

// --- ProvisioningRepo ---
type ProvisioningRepo struct {
	s *Storage
}

func (r *ProvisioningRepo) Create(ctx context.Context, job *domain.ProvisioningJob) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	for _, j := range r.s.jobs {
		if j.ID == job.ID || (job.Key != "" && j.Key == job.Key) {
			return domain.ErrDuplicateKey
		}
	}

	clone := *job
	now := time.Now().UTC()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.s.jobs[clone.ID] = &clone
	return nil
}

func (r *ProvisioningRepo) GetByID(ctx context.Context, id string) (*domain.ProvisioningJob, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	j, ok := r.s.jobs[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *j
	return &clone, nil
}

func (r *ProvisioningRepo) GetByKey(ctx context.Context, key string) (*domain.ProvisioningJob, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	for _, j := range r.s.jobs {
		if j.Key == key {
			clone := *j
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ProvisioningRepo) ClaimPendingJobs(ctx context.Context, batchSize int, lockDuration time.Duration) ([]*domain.ProvisioningJob, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	now := time.Now().UTC()
	var claimed []*domain.ProvisioningJob

	for _, j := range r.s.jobs {
		if len(claimed) >= batchSize {
			break
		}
		if (j.Status == domain.JobPending || (j.Status == domain.JobRunning && j.LockedAt != nil && j.LockedAt.Add(lockDuration).Before(now))) {
			if j.NextAttemptAt != nil && j.NextAttemptAt.After(now) {
				continue
			}
			j.Status = domain.JobRunning
			j.LockedAt = &now
			j.Attempts++
			j.UpdatedAt = now
			clone := *j
			claimed = append(claimed, &clone)
		}
	}
	return claimed, nil
}

func (r *ProvisioningRepo) MarkCompleted(ctx context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	j, ok := r.s.jobs[id]
	if !ok {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	j.Status = domain.JobCompleted
	j.CompletedAt = &now
	j.LockedAt = nil
	j.UpdatedAt = now
	return nil
}

func (r *ProvisioningRepo) MarkFailed(ctx context.Context, id, lastError string, nextAttempt *time.Time, toManualReview bool) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	j, ok := r.s.jobs[id]
	if !ok {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	if toManualReview {
		j.Status = domain.JobManualReview
	} else {
		j.Status = domain.JobPending
	}
	j.LastError = lastError
	j.NextAttemptAt = nextAttempt
	j.LockedAt = nil
	j.UpdatedAt = now
	return nil
}

func (r *ProvisioningRepo) ResetForRetry(ctx context.Context, id string) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	j, ok := r.s.jobs[id]
	if !ok {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	j.Status = domain.JobPending
	j.Attempts = 0
	j.LastError = ""
	j.NextAttemptAt = nil
	j.LockedAt = nil
	j.UpdatedAt = now
	return nil
}

func (r *ProvisioningRepo) List(ctx context.Context, status string, limit, offset int) ([]*domain.ProvisioningJob, int64, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.ProvisioningJob
	for _, j := range r.s.jobs {
		if status != "" && string(j.Status) != status {
			continue
		}
		clone := *j
		res = append(res, &clone)
	}
	total := int64(len(res))
	if offset > len(res) {
		return []*domain.ProvisioningJob{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(res) {
		end = len(res)
	}
	return res[offset:end], total, nil
}

func (r *ProvisioningRepo) DeleteCompletedJobs(ctx context.Context, olderThan time.Time) (int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	var count int64
	for id, j := range r.s.jobs {
		if j.Status == domain.JobCompleted && j.CompletedAt != nil && j.CompletedAt.Before(olderThan) {
			delete(r.s.jobs, id)
			count++
		}
	}
	return count, nil
}

// --- AuditRepo ---
type AuditRepo struct {
	s *Storage
}

func (r *AuditRepo) Record(ctx context.Context, event *domain.AuditEvent) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	clone := *event
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = time.Now().UTC()
	}
	r.s.auditEvents = append([]*domain.AuditEvent{&clone}, r.s.auditEvents...)
	return nil
}

func (r *AuditRepo) List(ctx context.Context, resourceType, resourceID string, limit, offset int) ([]*domain.AuditEvent, int64, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.AuditEvent
	for _, e := range r.s.auditEvents {
		if resourceType != "" && e.ResourceType != resourceType {
			continue
		}
		if resourceID != "" && e.ResourceID != resourceID {
			continue
		}
		clone := *e
		res = append(res, &clone)
	}
	total := int64(len(res))
	if offset > len(res) {
		return []*domain.AuditEvent{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(res) {
		end = len(res)
	}
	return res[offset:end], total, nil
}

func (r *AuditRepo) DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	var retained []*domain.AuditEvent
	var count int64
	for _, e := range r.s.auditEvents {
		if e.CreatedAt.Before(olderThan) {
			count++
		} else {
			retained = append(retained, e)
		}
	}
	r.s.auditEvents = retained
	return count, nil
}

// --- WebhookRepo ---
type WebhookRepo struct {
	s *Storage
}

func (r *WebhookRepo) RecordIncoming(ctx context.Context, event *domain.WebhookEvent) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	clone := *event
	if clone.CreatedAt.IsZero() {
		clone.CreatedAt = time.Now().UTC()
	}
	r.s.webhooks[clone.ID] = &clone
	return nil
}

func (r *WebhookRepo) GetByEventID(ctx context.Context, provider, eventID string) (*domain.WebhookEvent, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	for _, w := range r.s.webhooks {
		if w.Provider == provider && w.EventID == eventID {
			clone := *w
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *WebhookRepo) ListIncoming(ctx context.Context, limit, offset int) ([]*domain.WebhookEvent, int64, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	var res []*domain.WebhookEvent
	for _, w := range r.s.webhooks {
		clone := *w
		res = append(res, &clone)
	}
	total := int64(len(res))
	if offset > len(res) {
		return []*domain.WebhookEvent{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > len(res) {
		end = len(res)
	}
	return res[offset:end], total, nil
}

func (r *WebhookRepo) MarkProcessed(ctx context.Context, id string, err error) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	w, ok := r.s.webhooks[id]
	if !ok {
		return domain.ErrNotFound
	}
	w.Processed = true
	if err != nil {
		w.ErrorMessage = err.Error()
	}
	return nil
}

// --- UserRepo ---
type UserRepo struct {
	s *Storage
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	emailKey := strings.ToLower(u.Email)
	if _, exists := r.s.usersByEmail[emailKey]; exists {
		return domain.ErrDuplicateKey
	}
	clone := *u
	now := time.Now().UTC()
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.s.users[clone.ID] = &clone
	r.s.usersByEmail[emailKey] = clone.ID
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	u, ok := r.s.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *u
	return &clone, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	id, ok := r.s.usersByEmail[strings.ToLower(email)]
	if !ok {
		return nil, domain.ErrNotFound
	}
	u := r.s.users[id]
	clone := *u
	return &clone, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()

	existing, ok := r.s.users[u.ID]
	if !ok {
		return domain.ErrNotFound
	}
	clone := *u
	clone.UpdatedAt = time.Now().UTC()
	*existing = clone
	return nil
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	r.s.mu.RLock()
	defer r.s.mu.RUnlock()

	return int64(len(r.s.users)), nil
}

func (s *Storage) clone() *Storage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := New()
	for k, v := range s.payments {
		cp := *v
		c.payments[k] = &cp
	}
	for k, v := range s.paymentsByTx {
		c.paymentsByTx[k] = v
	}
	for k, v := range s.apiKeys {
		ck := *v
		c.apiKeys[k] = &ck
	}
	for k, v := range s.apiKeysByHash {
		c.apiKeysByHash[k] = v
	}
	for k, v := range s.orders {
		co := *v
		c.orders[k] = &co
	}
	for k, v := range s.jobs {
		cj := *v
		c.jobs[k] = &cj
	}
	for _, a := range s.auditEvents {
		ca := *a
		c.auditEvents = append(c.auditEvents, &ca)
	}
	for k, v := range s.webhooks {
		cw := *v
		c.webhooks[k] = &cw
	}
	for k, v := range s.users {
		cu := *v
		c.users[k] = &cu
	}
	for k, v := range s.usersByEmail {
		c.usersByEmail[k] = v
	}
	return c
}

func (s *Storage) apply(c *Storage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments = c.payments
	s.paymentsByTx = c.paymentsByTx
	s.apiKeys = c.apiKeys
	s.apiKeysByHash = c.apiKeysByHash
	s.orders = c.orders
	s.jobs = c.jobs
	s.auditEvents = c.auditEvents
	s.webhooks = c.webhooks
	s.users = c.users
	s.usersByEmail = c.usersByEmail
}

// --- TxManager ---
type TxManager struct {
	s *Storage
}

func (m *TxManager) Begin(ctx context.Context) (domain.Tx, error) {
	return &memTx{parent: m.s, staged: m.s.clone()}, nil
}

func (m *TxManager) WithTx(ctx context.Context, fn func(tx domain.Tx) error) error {
	tx := &memTx{parent: m.s, staged: m.s.clone()}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type memTx struct {
	parent *Storage
	staged *Storage
}

func (t *memTx) Payments() domain.PaymentRepository         { return &PaymentRepo{s: t.staged} }
func (t *memTx) APIKeys() domain.APIKeyRepository           { return &APIKeyRepo{s: t.staged} }
func (t *memTx) Orders() domain.OrderRepository             { return &OrderRepo{s: t.staged} }
func (t *memTx) Provisioning() domain.ProvisioningRepository { return &ProvisioningRepo{s: t.staged} }
func (t *memTx) Audit() domain.AuditRepository               { return &AuditRepo{s: t.staged} }
func (t *memTx) Webhooks() domain.WebhookRepository         { return &WebhookRepo{s: t.staged} }
func (t *memTx) Commit() error {
	if t.staged != nil && t.parent != nil {
		t.parent.apply(t.staged)
	}
	return nil
}
func (t *memTx) Rollback() error {
	t.staged = nil
	return nil
}
