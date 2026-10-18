# Changelog

All notable changes to **PayGate** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-10-18

### Initial Open-Source Release :tada:

#### Highlights
- **Decoupled Architecture:** Replaced legacy BeeCloud NestJS/MongoDB verifier with an independent, pure Go core backend and React 18 TypeScript administrative interface.
- **Pluggable Storage Layer:** Support for **PostgreSQL 16** (production scale) and **Pure Go SQLite** (`modernc.org/sqlite`, zero CGO dependencies), with automatic in-memory fallback.
- **Ultra-Low Resource Footprint:** Idles at **~15MB RAM** (91% lower than legacy verifier), boots in **<100ms**, and ships in a **28.4MB** static Alpine container.
- **BeeCloud 1-Click & App Pods:** Native support for 1-Click deployment on BeeCloud and Kubernetes App Pod namespaces (`supportsAppPods: true`).

#### Features
- **Payment State Machine:** Strict unidirectional state transitions (`PENDING` $\to$ `VERIFIED` $\to$ `CLAIMED` $\to$ `REFUNDED` / `FAILED` / `EXPIRED`).
- **Multi-Rail Provider Support:**
  - Commercial Bank of Ethiopia (CBE) v2 API and receipt parser with compound merchant receiver verification.
  - Telebirr (Ethio Telecom) direct receipt verifier.
  - Bank of Abyssinia (BOA) online slip verification.
  - Deterministic Mock Sandbox rail with preset test fixtures.
- **Asymmetric Amount Tolerance:** Support for `-1.00 ETB` underpay and `+5.00 ETB` overpay tolerance to prevent friction from bank transaction fees.
- **Durable Provisioning Outbox:** Transactional outbox pattern with worker lease locking (`SKIP LOCKED`), exponential backoff retries, and HMAC-SHA256 signed webhooks.
- **Live Database Switching:** Point PayGate to an external PostgreSQL or SQLite database directly from the Web UI or REST API without downtime.
- **Preset API Keys & Headless Mode:** Run PayGate as a pure API microservice without web UI (`DISABLE_UI=true`) with static keys preset via environment variables (`API_KEY`, `PRESET_API_KEYS`, `MASTER_TOKEN`).
- **Administrative UI & Developer Playground:**
  - Modern React 18 + Tailwind CSS + Lucide dashboard.
  - Interactive Developer Playground with one-click copyable cURL, JavaScript, and Python code snippets.
  - Real-time Transactions, Orders, Outbox Provisioning Jobs, API Keys, and Audit Logs explorers.
  - Zero vanity metrics: all analytics aggregate directly from database queries.
- **CLI Utility (`paygate-cli`):**
  - Admin account creation.
  - Lossless database snapshot export (`export-data`) and import (`import-data`) for zero-downtime engine switching.
