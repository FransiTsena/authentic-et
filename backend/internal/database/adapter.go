package database

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/beecloud/paygate/internal/config"
	"github.com/beecloud/paygate/internal/database/postgres"
	"github.com/beecloud/paygate/internal/database/sqlite"
	"github.com/beecloud/paygate/internal/domain"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

//go:embed migrations/postgres/001_initial_schema.sql
var postgresMigrationSQL string

//go:embed migrations/sqlite/001_initial_schema.sql
var sqliteMigrationSQL string

type Closer func() error

func Initialize(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*domain.Repositories, Closer, error) {
	logger.Info("initializing database", "driver", cfg.DatabaseDriver)

	var repos *domain.Repositories
	var closer Closer

	switch cfg.DatabaseDriver {
	case "postgres":
		db, err := postgres.Open(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("postgres initialization failed: %w", err)
		}
		closer = db.Close

		logger.Info("applying postgres migrations...")
		if err := db.Migrate(ctx, postgresMigrationSQL); err != nil {
			_ = closer()
			return nil, nil, fmt.Errorf("postgres migration failed: %w", err)
		}
		repos = db.Repositories()

	case "sqlite":
		db, err := sqlite.Open(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("sqlite initialization failed: %w", err)
		}
		closer = db.Close

		logger.Info("applying sqlite migrations...")
		if err := db.Migrate(ctx, sqliteMigrationSQL); err != nil {
			_ = closer()
			return nil, nil, fmt.Errorf("sqlite migration failed: %w", err)
		}
		repos = db.Repositories()

	default:
		return nil, nil, fmt.Errorf("unsupported database driver: %s", cfg.DatabaseDriver)
	}

	// Seed default admin user if none exists
	if err := seedAdminUser(ctx, repos.Users, cfg, logger); err != nil {
		logger.Warn("failed to check or seed admin user", "error", err)
	}

	logger.Info("database initialization completed successfully", "driver", cfg.DatabaseDriver)
	return repos, closer, nil
}

func seedAdminUser(ctx context.Context, userRepo domain.UserRepository, cfg *config.Config, logger *slog.Logger) error {
	count, err := userRepo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		logger.Info("no existing users detected and no admin credentials in environment. Initial administrator registration wizard active at web dashboard or /api/v1/auth/register")
		return nil
	}

	logger.Info("no existing users detected. Seeding initial administrator from environment...", "email", cfg.AdminEmail)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	admin := &domain.User{
		ID:           uuid.NewString(),
		Email:        cfg.AdminEmail,
		PasswordHash: string(hashedPassword),
		Name:         "System Administrator",
		Role:         domain.RoleAdmin,
		Active:       true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := userRepo.Create(ctx, admin); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	logger.Info("initial administrator created successfully", "email", cfg.AdminEmail)
	return nil
}
