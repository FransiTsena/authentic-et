package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/google/uuid"
)

type OrderService struct {
	cfg          *config.Config
	repos        *domain.Repositories
	payments     *PaymentService
	provisioning *ProvisioningService
	logger       *slog.Logger
}

func NewOrderService(
	cfg *config.Config,
	repos *domain.Repositories,
	payments *PaymentService,
	provisioning *ProvisioningService,
	logger *slog.Logger,
) *OrderService {
	return &OrderService{
		cfg:          cfg,
		repos:        repos,
		payments:     payments,
		provisioning: provisioning,
		logger:       logger,
	}
}

type CreateOrderInput struct {
	UserID        string         `json:"userId"`
	ServiceType   string         `json:"serviceType"`
	Tier          string         `json:"tier"`
	Amount        float64        `json:"amount"`
	Currency      string         `json:"currency"`
	BillingPeriod int            `json:"billingPeriod"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (s *OrderService) CreateOrder(ctx context.Context, input CreateOrderInput) (*domain.Order, error) {
	if input.Currency == "" {
		input.Currency = "ETB"
	}
	if input.BillingPeriod <= 0 {
		input.BillingPeriod = 1
	}

	order := &domain.Order{
		ID:            uuid.NewString(),
		UserID:        input.UserID,
		ServiceType:   input.ServiceType,
		Tier:          input.Tier,
		Amount:        input.Amount,
		Currency:      input.Currency,
		BillingPeriod: input.BillingPeriod,
		Status:        domain.OrderStatusPendingPayment,
		Metadata:      input.Metadata,
	}

	if err := s.repos.Orders.Create(ctx, order); err != nil {
		return nil, err
	}

	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "order.created",
		ActorType:    "user",
		ActorID:      input.UserID,
		ResourceID:   order.ID,
		ResourceType: "order",
		Details: map[string]any{
			"serviceType": input.ServiceType,
			"tier":        input.Tier,
			"amount":      input.Amount,
		},
	})

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.repos.Orders.GetByID(ctx, orderID)
}

func (s *OrderService) ListOrders(ctx context.Context, userID string, limit, offset int, isAdmin bool) ([]*domain.Order, int64, error) {
	if isAdmin && userID == "" {
		return s.repos.Orders.ListAll(ctx, limit, offset)
	}
	orders, err := s.repos.Orders.ListByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return orders, int64(len(orders)), nil
}

type VerifyOrderPaymentResult struct {
	Success      bool                              `json:"success"`
	Order        *domain.Order                     `json:"order"`
	Verification *domain.PaymentVerificationResult `json:"verification,omitempty"`
	Message      string                            `json:"message"`
}

func (s *OrderService) VerifyOrderPayment(
	ctx context.Context,
	orderID, userID, providerKey, txID string,
) (*VerifyOrderPaymentResult, error) {
	order, err := s.repos.Orders.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status == domain.OrderStatusPaid || order.Status == domain.OrderStatusActive {
		return &VerifyOrderPaymentResult{
			Success: true,
			Order:   order,
			Message: "Order is already marked as paid.",
		}, nil
	}

	if order.Status == domain.OrderStatusCancelled {
		return nil, domain.NewAppError("ORDER_CANCELLED", "Order has been cancelled and cannot accept payment", 400, nil)
	}

	// 1. Verify transaction via payment service with autoClaim = true
	verification, err := s.payments.VerifyTransaction(ctx, domain.VerifyPaymentRequest{
		Provider:       providerKey,
		TransactionID:  txID,
		ExpectedAmount: &order.Amount,
		ReferenceID:    fmt.Sprintf("order_%s", order.ID),
		ReferenceType:  "order_payment",
		AutoClaim:      true,
		ClaimedBy:      fmt.Sprintf("user_%s", userID),
		UserID:         userID,
	}, nil, true)
	if err != nil {
		return nil, err
	}

	if !verification.Verified {
		errMsg := verification.WarningMessage
		if errMsg == "" {
			errMsg = verification.ErrorMessage
		}
		if errMsg == "" {
			errMsg = "Payment verification failed"
		}
		return nil, domain.NewAppError("VERIFICATION_FAILED", errMsg, 400, nil)
	}

	// 2. Atomically mark order as paid
	now := time.Now().UTC()
	updatedOrder, err := s.repos.Orders.MarkPaid(ctx, order.ID, providerKey, txID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to mark order paid: %w", err)
	}

	// 3. Dispatch durable provisioning job to outbox
	provisioningJob := &domain.ProvisioningJob{
		ID:          uuid.NewString(),
		Key:         fmt.Sprintf("order_activate_%s", order.ID),
		OrderID:     order.ID,
		UserID:      userID,
		ServiceType: order.ServiceType,
		Action:      "activate",
		Payload: map[string]any{
			"orderId":          order.ID,
			"serviceType":      order.ServiceType,
			"tier":             order.Tier,
			"amount":           order.Amount,
			"billingPeriod":    order.BillingPeriod,
			"paymentReference": txID,
			"paymentProvider":  providerKey,
			"metadata":         order.Metadata,
		},
	}
	_ = s.provisioning.Dispatch(ctx, provisioningJob)

	// 4. Record audit event
	_ = s.repos.Audit.Record(ctx, &domain.AuditEvent{
		ID:           uuid.NewString(),
		Action:       "order.paid",
		ActorType:    "user",
		ActorID:      userID,
		ResourceID:   order.ID,
		ResourceType: "order",
		Details: map[string]any{
			"paymentReference": txID,
			"paymentProvider":  providerKey,
			"amount":           order.Amount,
		},
	})

	return &VerifyOrderPaymentResult{
		Success:      true,
		Order:        updatedOrder,
		Verification: verification,
		Message:      "Payment verified successfully! Service activation is now in progress.",
	}, nil
}
