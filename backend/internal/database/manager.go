package database

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/authentic-et/authentic-et/internal/config"
	"github.com/authentic-et/authentic-et/internal/database/memory"
	"github.com/authentic-et/authentic-et/internal/database/postgres"
	"github.com/authentic-et/authentic-et/internal/database/sqlite"
	"github.com/authentic-et/authentic-et/internal/domain"
)

type DBStatus struct {
	Driver          string    `json:"driver"`         // "memory", "sqlite", "postgres"
	IsEphemeral     bool      `json:"isEphemeral"`    // true if running in-memory
	WarningMessage  string    `json:"warningMessage"` // non-empty if running ephemeral or warning
	Connected       bool      `json:"connected"`
	ConnectionError string    `json:"connectionError,omitempty"`
	MaskedDSN       string    `json:"maskedDsn,omitempty"`
	ConnectedAt     time.Time `json:"connectedAt"`
}

type Manager struct {
	mu              sync.RWMutex
	cfg             *config.Config
	logger          *slog.Logger
	currentDriver   string
	isEphemeral     bool
	warningMessage  string
	connectionError string
	rawDSN          string
	connectedAt     time.Time
	activeRepos     *domain.Repositories
	activeCloser    Closer
	proxyRepos      *domain.Repositories
}

func NewManager(cfg *config.Config, logger *slog.Logger) *Manager {
	m := &Manager{
		cfg:         cfg,
		logger:      logger,
		connectedAt: time.Now().UTC(),
	}
	m.proxyRepos = m.createProxyRepositories()
	return m
}

func (m *Manager) Status() DBStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return DBStatus{
		Driver:          m.currentDriver,
		IsEphemeral:     m.isEphemeral,
		WarningMessage:  m.warningMessage,
		Connected:       m.activeRepos != nil,
		ConnectionError: m.connectionError,
		MaskedDSN:       maskDSN(m.rawDSN),
		ConnectedAt:     m.connectedAt,
	}
}

func (m *Manager) Repositories() *domain.Repositories {
	return m.proxyRepos
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeCloser != nil {
		return m.activeCloser()
	}
	return nil
}

func (m *Manager) Initialize(ctx context.Context) error {
	m.logger.Info("initializing database manager", "preferred_driver", m.cfg.DatabaseDriver)

	targetDriver := strings.ToLower(m.cfg.DatabaseDriver)
	if targetDriver == "" {
		targetDriver = "sqlite"
	}

	// Attempt preferred driver
	var initErr error
	switch targetDriver {
	case "postgres":
		initErr = m.initPostgres(ctx, m.cfg.DatabaseURL)
	case "sqlite":
		initErr = m.initSQLite(ctx, m.cfg.SQLitePath)
	case "memory", "none":
		m.initMemory(ctx, "authentic-et is running in IN-MEMORY EPHEMERAL mode. Records will NOT persist across restarts.")
		return nil
	default:
		initErr = fmt.Errorf("unsupported database driver: %s", targetDriver)
	}

	if initErr != nil {
		// Log loud warning and fall back to memory
		warnMsg := fmt.Sprintf("⚠️  WARNING: Database connection to '%s' failed (%v). Falling back to IN-MEMORY EPHEMERAL storage mode. Server will remain fully functional, but records will NOT persist across restarts! You can point authentic-et to your existing database at any time from the Web Admin dashboard under Settings -> Database.", targetDriver, initErr)
		m.logger.Warn(warnMsg)

		m.mu.Lock()
		m.connectionError = initErr.Error()
		m.mu.Unlock()

		m.initMemory(ctx, fmt.Sprintf("Database '%s' connection failed: %v. Running in ephemeral in-memory mode.", targetDriver, initErr))
	}

	return nil
}

func (m *Manager) initPostgres(ctx context.Context, dsn string) error {
	tempCfg := *m.cfg
	tempCfg.DatabaseURL = dsn
	tempCfg.DatabaseDriver = "postgres"

	db, err := postgres.Open(&tempCfg)
	if err != nil {
		return fmt.Errorf("postgres connection error: %w", err)
	}

	m.logger.Info("applying postgres migrations...")
	if err := db.Migrate(ctx, postgresMigrationSQL); err != nil {
		_ = db.Close()
		return fmt.Errorf("postgres migration error: %w", err)
	}

	repos := db.Repositories()
	if err := seedAdminUser(ctx, repos.Users, m.cfg, m.logger); err != nil {
		m.logger.Warn("failed to check or seed admin user in postgres", "error", err)
	}

	m.mu.Lock()
	if m.activeCloser != nil {
		_ = m.activeCloser()
	}
	m.activeRepos = repos
	m.activeCloser = db.Close
	m.currentDriver = "postgres"
	m.isEphemeral = false
	m.warningMessage = ""
	m.connectionError = ""
	m.rawDSN = dsn
	m.connectedAt = time.Now().UTC()
	m.mu.Unlock()

	m.logger.Info("connected to postgres database successfully")
	return nil
}

func (m *Manager) initSQLite(ctx context.Context, path string) error {
	tempCfg := *m.cfg
	tempCfg.SQLitePath = path
	tempCfg.DatabaseDriver = "sqlite"

	db, err := sqlite.Open(&tempCfg)
	if err != nil {
		return fmt.Errorf("sqlite connection error: %w", err)
	}

	m.logger.Info("applying sqlite migrations...")
	if err := db.Migrate(ctx, sqliteMigrationSQL); err != nil {
		_ = db.Close()
		return fmt.Errorf("sqlite migration error: %w", err)
	}

	repos := db.Repositories()
	if err := seedAdminUser(ctx, repos.Users, m.cfg, m.logger); err != nil {
		m.logger.Warn("failed to check or seed admin user in sqlite", "error", err)
	}

	m.mu.Lock()
	if m.activeCloser != nil {
		_ = m.activeCloser()
	}
	m.activeRepos = repos
	m.activeCloser = db.Close
	m.currentDriver = "sqlite"
	m.isEphemeral = false
	m.warningMessage = ""
	m.connectionError = ""
	m.rawDSN = path
	m.connectedAt = time.Now().UTC()
	m.mu.Unlock()

	m.logger.Info("connected to sqlite database successfully", "path", path)
	return nil
}

func (m *Manager) initMemory(ctx context.Context, warning string) {
	memStore := memory.New()
	repos := memStore.Repositories()

	_ = seedAdminUser(ctx, repos.Users, m.cfg, m.logger)

	m.mu.Lock()
	if m.activeCloser != nil {
		_ = m.activeCloser()
	}
	m.activeRepos = repos
	m.activeCloser = func() error { return nil }
	m.currentDriver = "memory"
	m.isEphemeral = true
	m.warningMessage = warning
	m.rawDSN = "in-memory"
	m.connectedAt = time.Now().UTC()
	m.mu.Unlock()
}

func (m *Manager) TestConnection(ctx context.Context, driver, dsn string) error {
	switch strings.ToLower(driver) {
	case "postgres":
		tempCfg := *m.cfg
		tempCfg.DatabaseURL = dsn
		tempCfg.DatabaseDriver = "postgres"
		db, err := postgres.Open(&tempCfg)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.DB().PingContext(ctx)

	case "sqlite":
		tempCfg := *m.cfg
		tempCfg.SQLitePath = dsn
		tempCfg.DatabaseDriver = "sqlite"
		db, err := sqlite.Open(&tempCfg)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.DB().PingContext(ctx)

	default:
		return fmt.Errorf("unsupported driver for testing: %s", driver)
	}
}

func (m *Manager) SwitchDatabase(ctx context.Context, driver, dsn string) error {
	driver = strings.ToLower(driver)
	m.logger.Info("switching database connection", "target_driver", driver)

	switch driver {
	case "postgres":
		return m.initPostgres(ctx, dsn)
	case "sqlite":
		return m.initSQLite(ctx, dsn)
	case "memory":
		m.initMemory(ctx, "Switched to IN-MEMORY EPHEMERAL mode.")
		return nil
	default:
		return fmt.Errorf("unsupported driver: %s", driver)
	}
}

func (m *Manager) getActive() *domain.Repositories {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeRepos
}

func maskDSN(raw string) string {
	if raw == "" || raw == "in-memory" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.User != nil {
		if _, hasPass := u.User.Password(); hasPass {
			u.User = url.UserPassword(u.User.Username(), "******")
		}
	}
	return u.String()
}
