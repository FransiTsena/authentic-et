-- PostgreSQL Schema Migration: 001_initial_schema.sql

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'developer',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(64) PRIMARY KEY,
    key_hash VARCHAR(128) NOT NULL UNIQUE,
    prefix VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    owner_email VARCHAR(255) NOT NULL,
    owner_user_id VARCHAR(64) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    tier VARCHAR(32) NOT NULL DEFAULT 'free',
    max_rps INT NOT NULL DEFAULT 10,
    monthly_quota INT NOT NULL DEFAULT 1000,
    used_this_month INT NOT NULL DEFAULT 0,
    daily_volume_limit NUMERIC(14,2) NOT NULL DEFAULT 50000.00,
    verified_volume_today NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    last_volume_reset_date VARCHAR(16) NOT NULL DEFAULT '',
    permissions TEXT NOT NULL DEFAULT '["payment:verify"]',
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_api_keys_owner ON api_keys(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active);

CREATE TABLE IF NOT EXISTS payments (
    id VARCHAR(64) PRIMARY KEY,
    provider_key VARCHAR(64) NOT NULL,
    transaction_id VARCHAR(128) NOT NULL,
    reference_id VARCHAR(128) NOT NULL,
    reference_type VARCHAR(64) NOT NULL DEFAULT 'direct_verification',
    amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(16) NOT NULL DEFAULT 'ETB',
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    claimed BOOLEAN NOT NULL DEFAULT false,
    claimed_at TIMESTAMPTZ,
    claimed_by VARCHAR(255),
    api_key_id VARCHAR(64),
    api_key_prefix VARCHAR(32),
    api_key_name VARCHAR(255),
    owner_user_id VARCHAR(64),
    payer_identifier VARCHAR(255),
    receiver_identifier VARCHAR(255),
    raw_payload JSONB,
    error_message TEXT,
    error_code VARCHAR(64),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_payments_provider_tx UNIQUE (provider_key, transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_payments_reference ON payments(reference_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_claimed ON payments(claimed);
CREATE INDEX IF NOT EXISTS idx_payments_owner ON payments(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);

CREATE TABLE IF NOT EXISTS orders (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    service_type VARCHAR(64) NOT NULL,
    tier VARCHAR(64) NOT NULL,
    amount NUMERIC(14,2) NOT NULL DEFAULT 0.00,
    currency VARCHAR(16) NOT NULL DEFAULT 'ETB',
    billing_period INT NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_payment',
    payment_provider VARCHAR(64),
    payment_reference VARCHAR(128),
    paid_at TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_reference ON orders(payment_reference);

CREATE TABLE IF NOT EXISTS provisioning_jobs (
    id VARCHAR(64) PRIMARY KEY,
    key VARCHAR(128) NOT NULL UNIQUE,
    order_id VARCHAR(64),
    user_id VARCHAR(64),
    service_type VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 5,
    locked_at TIMESTAMPTZ,
    next_attempt_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    last_error TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_next ON provisioning_jobs(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_jobs_locked_at ON provisioning_jobs(locked_at);

CREATE TABLE IF NOT EXISTS audit_events (
    id VARCHAR(64) PRIMARY KEY,
    action VARCHAR(64) NOT NULL,
    actor_type VARCHAR(32) NOT NULL,
    actor_id VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    resource_type VARCHAR(64) NOT NULL,
    details JSONB,
    ip_address VARCHAR(64),
    correlation_id VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_events(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_events(created_at DESC);

CREATE TABLE IF NOT EXISTS webhook_events (
    id VARCHAR(64) PRIMARY KEY,
    provider VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    event_id VARCHAR(128),
    payment_reference VARCHAR(128),
    raw_payload JSONB NOT NULL,
    signature_valid BOOLEAN NOT NULL DEFAULT false,
    processed BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhooks_provider ON webhook_events(provider, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhooks_reference ON webhook_events(payment_reference);
