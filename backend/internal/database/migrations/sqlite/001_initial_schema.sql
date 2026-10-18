-- SQLite Schema Migration: 001_initial_schema.sql

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'developer',
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    key_hash TEXT NOT NULL UNIQUE,
    prefix TEXT NOT NULL,
    name TEXT NOT NULL,
    owner_email TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    tier TEXT NOT NULL DEFAULT 'free',
    max_rps INTEGER NOT NULL DEFAULT 10,
    monthly_quota INTEGER NOT NULL DEFAULT 1000,
    used_this_month INTEGER NOT NULL DEFAULT 0,
    daily_volume_limit REAL NOT NULL DEFAULT 50000.0,
    verified_volume_today REAL NOT NULL DEFAULT 0.0,
    last_volume_reset_date TEXT NOT NULL DEFAULT '',
    permissions TEXT NOT NULL DEFAULT '["payment:verify"]',
    last_used_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_api_keys_owner ON api_keys(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    provider_key TEXT NOT NULL,
    transaction_id TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    reference_type TEXT NOT NULL DEFAULT 'direct_verification',
    amount REAL NOT NULL DEFAULT 0.0,
    currency TEXT NOT NULL DEFAULT 'ETB',
    status TEXT NOT NULL DEFAULT 'PENDING',
    claimed INTEGER NOT NULL DEFAULT 0,
    claimed_at TEXT,
    claimed_by TEXT,
    api_key_id TEXT,
    api_key_prefix TEXT,
    api_key_name TEXT,
    owner_user_id TEXT,
    payer_identifier TEXT,
    receiver_identifier TEXT,
    raw_payload TEXT,
    error_message TEXT,
    error_code TEXT,
    verified_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CONSTRAINT uq_payments_provider_tx UNIQUE (provider_key, transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_payments_reference ON payments(reference_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_claimed ON payments(claimed);
CREATE INDEX IF NOT EXISTS idx_payments_owner ON payments(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    service_type TEXT NOT NULL,
    tier TEXT NOT NULL,
    amount REAL NOT NULL DEFAULT 0.0,
    currency TEXT NOT NULL DEFAULT 'ETB',
    billing_period INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'pending_payment',
    payment_provider TEXT,
    payment_reference TEXT,
    paid_at TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_reference ON orders(payment_reference);

CREATE TABLE IF NOT EXISTS provisioning_jobs (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    order_id TEXT,
    user_id TEXT,
    service_type TEXT NOT NULL,
    action TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    locked_at TEXT,
    next_attempt_at TEXT,
    completed_at TEXT,
    last_error TEXT,
    payload TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_next ON provisioning_jobs(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_jobs_locked_at ON provisioning_jobs(locked_at);

CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY,
    action TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    details TEXT,
    ip_address TEXT,
    correlation_id TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_events(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_events(created_at DESC);

CREATE TABLE IF NOT EXISTS webhook_events (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_id TEXT,
    payment_reference TEXT,
    raw_payload TEXT NOT NULL,
    signature_valid INTEGER NOT NULL DEFAULT 0,
    processed INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_webhooks_provider ON webhook_events(provider, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhooks_reference ON webhook_events(payment_reference);
