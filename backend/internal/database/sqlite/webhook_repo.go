package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/authentic-et/authentic-et/internal/domain"
)

type WebhookRepo struct {
	db queryable
}

func (r *WebhookRepo) RecordIncoming(ctx context.Context, e *domain.WebhookEvent) error {
	now := time.Now()
	e.CreatedAt = now

	rawJSON, _ := json.Marshal(e.RawPayload)
	sigValidInt := 0
	if e.SignatureValid {
		sigValidInt = 1
	}
	processedInt := 0
	if e.Processed {
		processedInt = 1
	}

	query := `INSERT INTO webhook_events (
		id, provider, event_type, event_id, payment_reference, raw_payload,
		signature_valid, processed, error_message, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		e.ID, e.Provider, e.EventType, e.EventID, e.PaymentReference, string(rawJSON),
		sigValidInt, processedInt, e.ErrorMessage, e.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *WebhookRepo) GetByEventID(ctx context.Context, provider, eventID string) (*domain.WebhookEvent, error) {
	query := `SELECT id, provider, event_type, event_id, payment_reference, raw_payload,
		signature_valid, processed, error_message, created_at
		FROM webhook_events WHERE provider = ? AND event_id = ?`
	row := r.db.QueryRowContext(ctx, query, provider, eventID)
	return scanWebhook(row)
}

func (r *WebhookRepo) ListIncoming(ctx context.Context, limit, offset int) ([]*domain.WebhookEvent, int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_events").Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, provider, event_type, event_id, payment_reference, raw_payload,
		signature_valid, processed, error_message, created_at
		FROM webhook_events ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.WebhookEvent
	for rows.Next() {
		e, err := scanWebhook(rows)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}

	return events, total, nil
}

func (r *WebhookRepo) MarkProcessed(ctx context.Context, id string, processErr error) error {
	processed := 1
	errMsg := ""
	if processErr != nil {
		processed = 0
		errMsg = processErr.Error()
	}

	query := `UPDATE webhook_events SET processed = ?, error_message = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, processed, errMsg, id)
	return err
}

func scanWebhook(scanner rowScanner) (*domain.WebhookEvent, error) {
	e := &domain.WebhookEvent{}
	var eventID, paymentRef, errMsg sql.NullString
	var rawPayloadStr, createdAtStr string
	var sigValidInt, processedInt int

	err := scanner.Scan(
		&e.ID, &e.Provider, &e.EventType, &eventID, &paymentRef, &rawPayloadStr,
		&sigValidInt, &processedInt, &errMsg, &createdAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	if eventID.Valid {
		e.EventID = eventID.String
	}
	if paymentRef.Valid {
		e.PaymentReference = paymentRef.String
	}
	if errMsg.Valid {
		e.ErrorMessage = errMsg.String
	}
	e.SignatureValid = (sigValidInt == 1)
	e.Processed = (processedInt == 1)
	if rawPayloadStr != "" {
		_ = json.Unmarshal([]byte(rawPayloadStr), &e.RawPayload)
	}
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		e.CreatedAt = t
	}

	return e, nil
}
