package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type AuditRepo struct {
	db queryable
}

func (r *AuditRepo) Record(ctx context.Context, e *domain.AuditEvent) error {
	now := time.Now().UTC()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}

	detailsJSON, _ := json.Marshal(e.Details)

	query := `INSERT INTO audit_events (
		id, action, actor_type, actor_id, resource_id, resource_type,
		details, ip_address, correlation_id, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		e.ID, e.Action, e.ActorType, e.ActorID, e.ResourceID, e.ResourceType,
		string(detailsJSON), e.IPAddress, e.CorrelationID, e.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (r *AuditRepo) List(ctx context.Context, resourceType, resourceID string, limit, offset int) ([]*domain.AuditEvent, int64, error) {
	where := ""
	var args []any
	if resourceType != "" && resourceID != "" {
		where = "WHERE resource_type = ? AND resource_id = ?"
		args = append(args, resourceType, resourceID)
	} else if resourceType != "" {
		where = "WHERE resource_type = ?"
		args = append(args, resourceType)
	}

	countQuery := "SELECT COUNT(*) FROM audit_events " + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	selectQuery := `SELECT id, action, actor_type, actor_id, resource_id, resource_type,
		details, ip_address, correlation_id, created_at
		FROM audit_events ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	listArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		e := &domain.AuditEvent{}
		var detailsStr, ip, corr sql.NullString
		var createdAtStr string

		if err := rows.Scan(
			&e.ID, &e.Action, &e.ActorType, &e.ActorID, &e.ResourceID, &e.ResourceType,
			&detailsStr, &ip, &corr, &createdAtStr,
		); err != nil {
			return nil, 0, err
		}

		if detailsStr.Valid && detailsStr.String != "" {
			_ = json.Unmarshal([]byte(detailsStr.String), &e.Details)
		}
		if ip.Valid {
			e.IPAddress = ip.String
		}
		if corr.Valid {
			e.CorrelationID = corr.String
		}
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			e.CreatedAt = t
		}
		events = append(events, e)
	}

	return events, total, nil
}

func (r *AuditRepo) DeleteOldAuditLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM audit_events WHERE created_at < ?`
	res, err := r.db.ExecContext(ctx, query, olderThan.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
