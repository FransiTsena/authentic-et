package domain

import (
	"context"
	"time"
)

type JobStatus string

const (
	JobPending      JobStatus = "pending"
	JobRunning      JobStatus = "running"
	JobCompleted    JobStatus = "completed"
	JobFailed       JobStatus = "failed"
	JobManualReview JobStatus = "manual_review"
)

type ProvisioningJob struct {
	ID            string         `json:"id"`
	Key           string         `json:"key"` // Unique idempotency key
	OrderID       string         `json:"orderId,omitempty"`
	UserID        string         `json:"userId,omitempty"`
	ServiceType   string         `json:"serviceType"`
	Action        string         `json:"action"` // e.g. "activate", "renew", "suspend", "deprovision"
	Status        JobStatus      `json:"status"`
	Attempts      int            `json:"attempts"`
	MaxAttempts   int            `json:"maxAttempts"`
	LockedAt      *time.Time     `json:"lockedAt,omitempty"`
	NextAttemptAt *time.Time     `json:"nextAttemptAt,omitempty"`
	CompletedAt   *time.Time     `json:"completedAt,omitempty"`
	LastError     string         `json:"lastError,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

type ProvisioningRepository interface {
	Create(ctx context.Context, job *ProvisioningJob) error
	GetByID(ctx context.Context, id string) (*ProvisioningJob, error)
	GetByKey(ctx context.Context, key string) (*ProvisioningJob, error)
	ClaimPendingJobs(ctx context.Context, batchSize int, lockDuration time.Duration) ([]*ProvisioningJob, error)
	MarkCompleted(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id, lastError string, nextAttempt *time.Time, toManualReview bool) error
	ResetForRetry(ctx context.Context, id string) error
	List(ctx context.Context, status string, limit, offset int) ([]*ProvisioningJob, int64, error)
	DeleteCompletedJobs(ctx context.Context, olderThan time.Time) (int64, error)
}
