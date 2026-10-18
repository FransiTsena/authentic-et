package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type APIKeyRepo struct {
	db queryable
}

func (r *APIKeyRepo) Create(ctx context.Context, k *domain.APIKey) error {
	now := time.Now()
	k.CreatedAt = now
	k.UpdatedAt = now

	permsJSON, _ := json.Marshal(k.Permissions)
	activeInt := 0
	if k.Active {
		activeInt = 1
	}

	var lastUsedAtStr *string
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.Format(time.RFC3339)
		lastUsedAtStr = &s
	}

	query := `INSERT INTO api_keys (
		id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		k.ID, k.KeyHash, k.Prefix, k.Name, k.OwnerEmail, k.OwnerUserID, activeInt, string(k.Tier),
		k.MaxRPS, k.MonthlyQuota, k.UsedThisMonth, k.DailyVolumeLimit, k.VerifiedVolumeToday,
		k.LastVolumeResetDate, string(permsJSON), lastUsedAtStr,
		k.CreatedAt.Format(time.RFC3339), k.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *APIKeyRepo) GetByID(ctx context.Context, id string) (*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys WHERE key_hash = ? AND active = 1`
	row := r.db.QueryRowContext(ctx, query, keyHash)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) ListByOwner(ctx context.Context, ownerUserID string) ([]*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys WHERE owner_user_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) ListAll(ctx context.Context) ([]*domain.APIKey, error) {
	query := `SELECT id, key_hash, prefix, name, owner_email, owner_user_id, active, tier,
		max_rps, monthly_quota, used_this_month, daily_volume_limit, verified_volume_today,
		last_volume_reset_date, permissions, last_used_at, created_at, updated_at
		FROM api_keys ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) Update(ctx context.Context, k *domain.APIKey) error {
	k.UpdatedAt = time.Now()
	permsJSON, _ := json.Marshal(k.Permissions)
	activeInt := 0
	if k.Active {
		activeInt = 1
	}

	var lastUsedAtStr *string
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.Format(time.RFC3339)
		lastUsedAtStr = &s
	}

	query := `UPDATE api_keys SET
		name = ?, active = ?, tier = ?, max_rps = ?, monthly_quota = ?, used_this_month = ?,
		daily_volume_limit = ?, verified_volume_today = ?, last_volume_reset_date = ?,
		permissions = ?, last_used_at = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		k.Name, activeInt, string(k.Tier), k.MaxRPS, k.MonthlyQuota, k.UsedThisMonth,
		k.DailyVolumeLimit, k.VerifiedVolumeToday, k.LastVolumeResetDate,
		string(permsJSON), lastUsedAtStr, k.UpdatedAt.Format(time.RFC3339),
		k.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *APIKeyRepo) IncrementUsage(ctx context.Context, id string, amount float64, todayDate string) error {
	nowStr := time.Now().Format(time.RFC3339)

	// Lazy daily volume reset if date changed, atomic increment of monthly counter
	query := `UPDATE api_keys SET
		used_this_month = used_this_month + 1,
		verified_volume_today = CASE WHEN last_volume_reset_date = ? THEN verified_volume_today + ? ELSE ? END,
		last_volume_reset_date = ?,
		last_used_at = ?,
		updated_at = ?
		WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, todayDate, amount, amount, todayDate, nowStr, nowStr, id)
	return err
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	nowStr := time.Now().Format(time.RFC3339)
	query := `UPDATE api_keys SET active = 0, updated_at = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, nowStr, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanAPIKey(scanner rowScanner) (*domain.APIKey, error) {
	k := &domain.APIKey{}
	var activeInt int
	var tierStr, permsStr string
	var lastUsedAtStr sql.NullString
	var createdAtStr, updatedAtStr string

	err := scanner.Scan(
		&k.ID, &k.KeyHash, &k.Prefix, &k.Name, &k.OwnerEmail, &k.OwnerUserID, &activeInt, &tierStr,
		&k.MaxRPS, &k.MonthlyQuota, &k.UsedThisMonth, &k.DailyVolumeLimit, &k.VerifiedVolumeToday,
		&k.LastVolumeResetDate, &permsStr, &lastUsedAtStr, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	k.Active = (activeInt == 1)
	k.Tier = domain.APIKeyTier(tierStr)
	_ = json.Unmarshal([]byte(permsStr), &k.Permissions)

	if lastUsedAtStr.Valid {
		if t, err := time.Parse(time.RFC3339, lastUsedAtStr.String); err == nil {
			k.LastUsedAt = &t
		}
	}
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		k.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		k.UpdatedAt = t
	}

	return k, nil
}
