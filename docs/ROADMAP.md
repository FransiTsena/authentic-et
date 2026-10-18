# authentic-et Project Roadmap

This document outlines the planned architectural milestones, provider rails, and feature expansions for **authentic-et**.

---

## 📍 Version 1.0.0 (Current Release) :white_check_mark:
- [x] Pure Go core backend with Chi router, context timeouts, and structured JSON logging.
- [x] Pluggable database architecture supporting **PostgreSQL 16** (`pgx/v5`) and **Pure Go SQLite** (`modernc.org/sqlite`, zero CGO).
- [x] Automatic graceful fallback to In-Memory Ephemeral storage with startup safety warnings.
- [x] Dynamic runtime database configuration from the Web Admin client (`/database`).
- [x] Multi-rail payment verifiers for **Commercial Bank of Ethiopia (CBE)**, **Telebirr**, **Bank of Abyssinia (BOA)**, and **Mock Sandbox**.
- [x] Asymmetric amount tolerance (`-1.00 ETB` / `+5.00 ETB`).
- [x] Transactional durable outbox worker with lease locking (`SKIP LOCKED`) and HMAC-SHA256 signed webhooks.
- [x] Preset static API keys (`API_KEY`, `PRESET_API_KEYS`, `MASTER_TOKEN`) and headless mode (`DISABLE_UI=true`).
- [x] Modern React 18 + TypeScript admin dashboard with interactive Developer Playground.
- [x] `authentic-cli` utility for lossless data export/import and admin seeding.
- [x] Full Docker Compose (`docker-compose-full.yml`), embedded SQLite compose, and all-in-one single container image.
- [x] Native BeeCloud 1-Click & Kubernetes App Pod deployment integration (`supportsAppPods: true`).

---

## 🚀 Version 1.1.0 (Q1 2027)
- [ ] **Additional Ethiopian Bank Rails:**
  - Awash Bank (Internet Banking slip scraper & QR).
  - Dashen Bank / Amole verifier.
  - Hibret Bank & Cooperative Bank of Oromia (Coop).
- [ ] **Prometheus Metrics Exporter:**
  - Expose `/metrics` with transaction counts, verification latencies, outbox retry rates, and provider uptime metrics.
- [ ] **OpenTelemetry (OTel) Tracing:**
  - Distributed trace IDs propagating across incoming requests, provider outbound calls, and webhook dispatches.
- [ ] **Automated Daily Settlement Reports:**
  - Scheduled CSV and PDF export of verified amounts and reconciliation logs.

---

## 🔮 Version 1.2.0 (Q2 2027)
- [ ] **Global Gateway Adapters:**
  - Pluggable fallback rails for Stripe, LemonSqueezy, and PayPal for international customers.
- [ ] **Webhook Dead Letter Queue (DLQ) Management:**
  - Visual inspect & replay for webhooks that exceeded max retries.
- [ ] **Multi-Tenant Organizations:**
  - Sub-merchant account isolation with scoped API keys and role-based permissions.
- [ ] **Native Mobile Admin Companion App:**
  - Push notifications for high-value confirmed payments and provider slip anomalies.

---

## Proposing Features
Have an idea or want to contribute a provider integration?
Please submit a [Feature Request](https://github.com/beecloud/paygate/issues/new?template=feature_request.md) or open a discussion on [GitHub Discussions](https://github.com/beecloud/paygate/discussions).
