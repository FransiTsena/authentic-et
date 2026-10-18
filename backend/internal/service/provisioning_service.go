package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/google/uuid"
)

type ProvisioningHandler func(ctx context.Context, job *domain.ProvisioningJob) error

type ProvisioningService struct {
	cfg      *config.Config
	repos    *domain.Repositories
	logger   *slog.Logger
	handlers map[string]ProvisioningHandler
	client   *http.Client
}

func NewProvisioningService(cfg *config.Config, repos *domain.Repositories, logger *slog.Logger) *ProvisioningService {
	s := &ProvisioningService{
		cfg:      cfg,
		repos:    repos,
		logger:   logger,
		handlers: make(map[string]ProvisioningHandler),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}

	// Register default handlers
	s.RegisterHandler("webhook", s.handleWebhookDelivery)
	s.RegisterHandler("activate", s.handleDefaultActivation)

	return s
}

func (s *ProvisioningService) RegisterHandler(action string, h ProvisioningHandler) {
	s.handlers[action] = h
}

func (s *ProvisioningService) Dispatch(ctx context.Context, job *domain.ProvisioningJob) error {
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.MaxAttempts <= 0 {
		job.MaxAttempts = 5
	}
	job.Status = domain.JobPending

	if err := s.repos.Provisioning.Create(ctx, job); err != nil {
		if err == domain.ErrDuplicateKey {
			// Idempotent duplicate delivery: job already queued
			s.logger.Info("provisioning job already dispatched", "key", job.Key)
			return nil
		}
		return err
	}

	s.logger.Info("dispatched provisioning job", "id", job.ID, "key", job.Key, "action", job.Action)
	return nil
}

func (s *ProvisioningService) ProcessPendingBatch(ctx context.Context, batchSize int) (int, error) {
	jobs, err := s.repos.Provisioning.ClaimPendingJobs(ctx, batchSize, s.cfg.WorkerLockDuration)
	if err != nil {
		return 0, fmt.Errorf("failed to claim pending provisioning jobs: %w", err)
	}

	processed := 0
	for _, job := range jobs {
		s.executeJob(ctx, job)
		processed++
	}

	return processed, nil
}

func (s *ProvisioningService) executeJob(ctx context.Context, job *domain.ProvisioningJob) {
	s.logger.Info("executing provisioning job", "id", job.ID, "key", job.Key, "action", job.Action, "attempt", job.Attempts)

	handler, ok := s.handlers[job.Action]
	if !ok {
		// Fallback to default activation handler if specific action handler not found
		handler = s.handlers["activate"]
	}

	err := handler(ctx, job)
	if err == nil {
		if markErr := s.repos.Provisioning.MarkCompleted(ctx, job.ID); markErr != nil {
			s.logger.Error("failed to mark job completed", "id", job.ID, "error", markErr)
		} else {
			s.logger.Info("provisioning job completed successfully", "id", job.ID, "key", job.Key)
			_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
				ID:           uuid.NewString(),
				Action:       "provisioning.completed",
				ActorType:    "system",
				ActorID:      "worker",
				ResourceID:   job.ID,
				ResourceType: "job",
				Details: map[string]any{
					"key":      job.Key,
					"action":   job.Action,
					"attempts": job.Attempts,
				},
			})
		}
		return
	}

	// Handle failure with exponential backoff
	s.logger.Warn("provisioning job execution failed", "id", job.ID, "key", job.Key, "error", err, "attempt", job.Attempts)

	toManualReview := job.Attempts >= job.MaxAttempts
	var nextAttempt *time.Time
	if !toManualReview {
		backoffMinutes := math.Min(60, math.Pow(2, float64(job.Attempts)))
		t := time.Now().UTC().Add(time.Duration(backoffMinutes) * time.Minute)
		nextAttempt = &t
	}

	_ = s.repos.Provisioning.MarkFailed(ctx, job.ID, err.Error(), nextAttempt, toManualReview)
	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "provisioning.failed",
		ActorType:    "system",
		ActorID:      "worker",
		ResourceID:   job.ID,
		ResourceType: "job",
		Details: map[string]any{
			"key":          job.Key,
			"action":       job.Action,
			"attempts":     job.Attempts,
			"manualReview": toManualReview,
			"error":        err.Error(),
		},
	})
}

func (s *ProvisioningService) RetryJob(ctx context.Context, jobID string) error {
	if err := s.repos.Provisioning.ResetForRetry(ctx, jobID); err != nil {
		return err
	}
	s.logger.Info("reset provisioning job for immediate retry", "id", jobID)
	return nil
}

func (s *ProvisioningService) ListJobs(ctx context.Context, status string, limit, offset int) ([]*domain.ProvisioningJob, int64, error) {
	return s.repos.Provisioning.List(ctx, status, limit, offset)
}

func (s *ProvisioningService) handleDefaultActivation(ctx context.Context, job *domain.ProvisioningJob) error {
	// If order ID is present, update order status to active
	if job.OrderID != "" {
		order, err := s.repos.Orders.GetByID(ctx, job.OrderID)
		if err == nil && order != nil {
			order.Status = domain.OrderStatusActive
			_ = s.repos.Orders.Update(ctx, order)
		}
	}

	// If webhook URL configured in payload, dispatch downstream webhook
	if urlVal, ok := job.Payload["webhookUrl"].(string); ok && urlVal != "" {
		return s.deliverHTTPWebhook(ctx, urlVal, job.Payload)
	}

	return nil
}

func (s *ProvisioningService) handleWebhookDelivery(ctx context.Context, job *domain.ProvisioningJob) error {
	urlVal, ok := job.Payload["targetUrl"].(string)
	if !ok || urlVal == "" {
		return fmt.Errorf("targetUrl missing in webhook job payload")
	}
	return s.deliverHTTPWebhook(ctx, urlVal, job.Payload)
}

func (s *ProvisioningService) deliverHTTPWebhook(ctx context.Context, targetURL string, payload map[string]any) error {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	// Compute HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(s.cfg.WebhookSecret))
	mac.Write(bodyBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "PayGate-Provisioning-Worker/1.0")
	req.Header.Set("X-PayGate-Signature", "sha256="+signature)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook endpoint returned HTTP status %d", resp.StatusCode)
	}

	return nil
}
