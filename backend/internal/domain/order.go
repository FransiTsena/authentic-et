package domain

import (
	"context"
	"time"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "pending_payment"
	OrderStatusProcessing     OrderStatus = "processing"
	OrderStatusPaid           OrderStatus = "paid"
	OrderStatusProvisioning   OrderStatus = "provisioning"
	OrderStatusActive         OrderStatus = "active"
	OrderStatusCompleted      OrderStatus = "completed"
	OrderStatusFailed         OrderStatus = "failed"
	OrderStatusCancelled      OrderStatus = "cancelled"
	OrderStatusRefunded       OrderStatus = "refunded"
)

type Order struct {
	ID               string         `json:"id"`
	UserID           string         `json:"userId"`
	ServiceType      string         `json:"serviceType"`
	Tier             string         `json:"tier"`
	Amount           float64        `json:"amount"`
	Currency         string         `json:"currency"`
	BillingPeriod    int            `json:"billingPeriod"` // in months
	Status           OrderStatus    `json:"status"`
	PaymentProvider  string         `json:"paymentProvider,omitempty"`
	PaymentReference string         `json:"paymentReference,omitempty"`
	PaidAt           *time.Time     `json:"paidAt,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type OrderRepository interface {
	Create(ctx context.Context, o *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	ListByUser(ctx context.Context, userID string) ([]*Order, error)
	ListAll(ctx context.Context, limit, offset int) ([]*Order, int64, error)
	Update(ctx context.Context, o *Order) error
	MarkPaid(ctx context.Context, id, provider, reference string, paidAt time.Time) (*Order, error)
}
