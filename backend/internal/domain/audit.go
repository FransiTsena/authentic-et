package domain

import (
	"context"
	"time"
)

type AuditEvent struct {
	ID            string         `json:"id"`
	Action        string         `json:"action"`    // e.g. "payment.verified", "payment.claimed", "apikey.created", "provisioning.dispatched"
	ActorType     string         `json:"actorType"` // "system", "user", "admin", "api_key"
	ActorID       string         `json:"actorId"`
	ResourceID    string         `json:"resourceId"`
	ResourceType  string         `json:"resourceType"` // "payment", "apikey", "order", "job"
	Details       map[string]any `json:"details,omitempty"`
	IPAddress     string         `json:"ipAddress,omitempty"`
	CorrelationID string         `json:"correlationId,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type AuditRepository interface {
	Record(ctx context.Context, event *AuditEvent) error
	List(ctx context.Context, resourceType, resourceID string, limit, offset int) ([]*AuditEvent, int64, error)
	DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error)
}
