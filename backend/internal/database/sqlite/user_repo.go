package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/beecloud/paygate/internal/domain"
)

type UserRepo struct {
	db queryable
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	activeInt := 0
	if u.Active {
		activeInt = 1
	}

	query := `INSERT INTO users (
		id, email, password_hash, name, role, active, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, strings.ToLower(u.Email), u.PasswordHash, u.Name, string(u.Role),
		activeInt, u.CreatedAt.Format(time.RFC3339), u.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return domain.ErrDuplicateKey
	}
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, role, active, created_at, updated_at
		FROM users WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, role, active, created_at, updated_at
		FROM users WHERE email = ?`
	row := r.db.QueryRowContext(ctx, query, strings.ToLower(email))
	return scanUser(row)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	u.UpdatedAt = time.Now()
	activeInt := 0
	if u.Active {
		activeInt = 1
	}

	query := `UPDATE users SET
		password_hash = ?, name = ?, role = ?, active = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		u.PasswordHash, u.Name, string(u.Role), activeInt, u.UpdatedAt.Format(time.RFC3339), u.ID,
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

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func scanUser(scanner rowScanner) (*domain.User, error) {
	u := &domain.User{}
	var activeInt int
	var roleStr string
	var createdAtStr, updatedAtStr string

	err := scanner.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &roleStr,
		&activeInt, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	u.Role = domain.UserRole(roleStr)
	u.Active = (activeInt == 1)
	if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		u.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		u.UpdatedAt = t
	}

	return u, nil
}
