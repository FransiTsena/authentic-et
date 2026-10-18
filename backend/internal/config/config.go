package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               int
	Environment        string
	LogLevel           string
	CORSAllowedOrigins []string
	DatabaseDriver     string // "postgres" or "sqlite"
	DatabaseURL        string
	SQLitePath         string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    time.Duration
	JWTSecret          string
	JWTExpiryHours     int
	AdminEmail         string
	AdminPassword      string
	MerchantNames      []string
	MerchantAccounts   []string
	UnderpayTolerance  float64
	OverpayTolerance   float64
	MockMode           bool
	PublicRPSBurst     int
	WorkerEnabled      bool
	WorkerPollInterval time.Duration
	WorkerConcurrency  int
	WorkerLockDuration time.Duration
	WebhookSecret      string
	PresetAPIKeys      []string
	MasterToken        string
	DisableUI          bool
	AllowRegistration  bool
	// Data Retention & Automated Cleanup
	CleanupEnabled             bool
	CleanupIntervalHours       int
	TransactionRetentionDays   int // e.g. 90 days for verified/claimed txs (0 or -1 to disable)
	FailedRetentionDays        int // e.g. 30 days for failed transactions
	UnclaimedExpiryHours       int // e.g. 72 hours for unverified/unclaimed pending txs
	AuditRetentionDays         int // e.g. 180 days for audit trail events
	CompletedJobsRetentionDays int // e.g. 14 days for completed outbox jobs
	DryRunCleanup              bool
}

func Load() (*Config, error) {
	presetKeys := getEnvAsSlice("PRESET_API_KEYS", nil)
	if len(presetKeys) == 0 {
		presetKeys = getEnvAsSlice("API_KEY", []string{})
	}
	masterToken := getEnv("MASTER_TOKEN", getEnv("ROOT_API_KEY", ""))
	if masterToken != "" {
		presetKeys = append(presetKeys, masterToken)
	}

	cfg := &Config{
		Port:               getEnvAsInt("PORT", 5151),
		Environment:        getEnv("APP_ENV", getEnv("NODE_ENV", "development")),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5151"}),
		DatabaseDriver:     strings.ToLower(getEnv("DATABASE_DRIVER", "sqlite")),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		SQLitePath:         getEnv("SQLITE_PATH", "./data/authentic.db"),
		MaxOpenConns:       getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:       getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime:    time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 15)) * time.Minute,
		JWTSecret:          getEnv("JWT_SECRET", "authentic-change-me-in-production-secret-key-32chars"),
		JWTExpiryHours:     getEnvAsInt("JWT_EXPIRY_HOURS", 72),
		AdminEmail:         getEnv("ADMIN_EMAIL", ""),
		AdminPassword:      getEnv("ADMIN_PASSWORD", ""),
		MerchantNames: getEnvAsSlice("PAYMENT_MERCHANT_NAME", []string{
			"fransi tsena",
			"beecloud",
			"authentic-et",
		}),
		MerchantAccounts: getEnvAsSlice("PAYMENT_MERCHANT_ACCOUNTS", []string{
			"1000646339972",
			"929945151",
			"183586602",
		}),
		UnderpayTolerance:          getEnvAsFloat("UNDERPAY_TOLERANCE_ETB", 1.0),
		OverpayTolerance:           getEnvAsFloat("OVERPAY_TOLERANCE_ETB", 5.0),
		MockMode:                   getEnvAsBool("PAYMENT_VERIFIER_MOCK_MODE", true),
		PublicRPSBurst:             getEnvAsInt("PUBLIC_RPS_BURST", 5),
		WorkerEnabled:              getEnvAsBool("WORKER_ENABLED", true),
		WorkerPollInterval:         time.Duration(getEnvAsInt("WORKER_POLL_INTERVAL_SECONDS", 5)) * time.Second,
		WorkerConcurrency:          getEnvAsInt("WORKER_CONCURRENCY", 4),
		WorkerLockDuration:         time.Duration(getEnvAsInt("WORKER_LOCK_DURATION_MINUTES", 10)) * time.Minute,
		WebhookSecret:              getEnv("WEBHOOK_SECRET", "whsec_default_authentic_secret"),
		PresetAPIKeys:              presetKeys,
		MasterToken:                masterToken,
		DisableUI:                  getEnvAsBool("DISABLE_UI", false),
		AllowRegistration:          getEnvAsBool("ALLOW_REGISTRATION", false),
		CleanupEnabled:             getEnvAsBool("CLEANUP_ENABLED", true),
		CleanupIntervalHours:       getEnvAsInt("CLEANUP_INTERVAL_HOURS", 24),
		TransactionRetentionDays:   getEnvAsInt("TRANSACTION_RETENTION_DAYS", 90),
		FailedRetentionDays:        getEnvAsInt("FAILED_RETENTION_DAYS", 30),
		UnclaimedExpiryHours:       getEnvAsInt("UNCLAIMED_EXPIRY_HOURS", 72),
		AuditRetentionDays:         getEnvAsInt("AUDIT_RETENTION_DAYS", 180),
		CompletedJobsRetentionDays: getEnvAsInt("COMPLETED_JOBS_RETENTION_DAYS", 14),
		DryRunCleanup:              getEnvAsBool("DRY_RUN_CLEANUP", false),
	}

	// Intelligent driver inference if not specified
	if cfg.DatabaseDriver == "" {
		if cfg.DatabaseURL != "" {
			cfg.DatabaseDriver = "postgres"
		} else {
			cfg.DatabaseDriver = "memory"
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration error: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	switch c.DatabaseDriver {
	case "postgres", "sqlite", "memory", "none", "":
		// valid
	default:
		return fmt.Errorf("unsupported DATABASE_DRIVER '%s': must be 'postgres', 'sqlite', or 'memory'", c.DatabaseDriver)
	}
	if c.DatabaseDriver == "postgres" && c.DatabaseURL == "" {
		// When driver is postgres but URL empty, fallback to local default
		c.DatabaseURL = "postgres://authentic:authentic@localhost:5432/authentic?sslmode=disable"
	}
	if c.Environment == "production" {
		if strings.Contains(c.JWTSecret, "change-me") || len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be set to a secure unique string of at least 32 characters in production")
		}
		if c.AdminPassword != "" && (c.AdminPassword == "Authentic2026!SecureAdmin" || strings.Contains(c.AdminPassword, "change-me")) {
			return fmt.Errorf("ADMIN_PASSWORD must not use default or placeholder credentials in production")
		}
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return v
}

func getEnvAsFloat(key string, defaultVal float64) float64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return defaultVal
	}
	return v
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := strings.ToLower(os.Getenv(key))
	if valStr == "" {
		return defaultVal
	}
	return valStr == "true" || valStr == "1" || valStr == "yes"
}

func getEnvAsSlice(key string, defaultVal []string) []string {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	parts := strings.Split(valStr, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			res = append(res, strings.ToLower(trimmed))
		}
	}
	if len(res) == 0 {
		return defaultVal
	}
	return res
}
