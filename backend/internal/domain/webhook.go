package domain

import (
	"context"
	"time"
)

type WebhookEvent struct {
	ID               string         `json:"id"`
	Provider         string         `json:"provider"`
	EventType        string         `json:"eventType"`
	EventID          string         `json:"eventId,omitempty"`
	PaymentReference string         `json:"paymentReference,omitempty"`
	RawPayload       map[string]any `json:"rawPayload"`
	SignatureValid   bool           `json:"signatureValid"`
	Processed        bool           `json:"processed"`
	ErrorMessage     string         `json:"errorMessage,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
}

type WebhookRepository interface {
	RecordIncoming(ctx context.Context, event *WebhookEvent) error
	GetByEventID(ctx context.Context, provider, eventID string) (*WebhookEvent, error)
	ListIncoming(ctx context.Context, limit, offset int) ([]*WebhookEvent, int64, error)
	MarkProcessed(ctx context.Context, id string, err error) error
}
