<div align="center">

# authentic-et ⚡
### High-Performance, Multi-Rail Payment Verification & Automated Provisioning Platform

[![CI](https://github.com/beecloud/paygate/actions/workflows/ci.yml/badge.svg)](https://github.com/beecloud/paygate/actions/workflows/ci.yml)
[![Release](https://img.shields.io/badge/release-v1.0.0-blue.svg)](https://github.com/beecloud/paygate/releases)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/beecloud/paygate)](https://goreportcard.com/report/github.com/beecloud/paygate)
[![Docker Image Size](https://img.shields.io/badge/docker%20image-28.4%20MB-success)](https://ghcr.io/beecloud/paygate)
[![RAM Consumption](https://img.shields.io/badge/idle%20RAM-~15%20MB-blueviolet)](docs/BENCHMARKS.md)
[![Pure Go SQLite](https://img.shields.io/badge/SQLite-Pure%20Go%20(No%20CGO)-blueviolet)](https://modernc.org/sqlite)
[![App Pod Ready](https://img.shields.io/badge/Kubernetes-App%20Pod%20Ready-326ce5.svg)](docs/DEPLOYMENT.md)

**authentic-et** is a standalone, production-grade payment verification and automated service-provisioning platform built with **Go**, **React**, **TypeScript**, and **PostgreSQL / SQLite**. Designed with financial correctness and zero external infrastructure requirements as top priorities, authentic-et eliminates vendor lock-in, prevents double-spending, and guarantees delivery of downstream service activation webhooks.

[Quick Start](#-quick-start) • [Architecture](docs/ARCHITECTURE.md) • [REST API Reference](docs/API.md) • [Deployment Guide](docs/DEPLOYMENT.md) • [Threat Model](docs/THREAT_MODEL.md) • [Benchmarks](docs/BENCHMARKS.md) • [Roadmap](docs/ROADMAP.md)

</div>

---

## 💡 Why PayGate?

Traditional payment verifiers are often tightly coupled to monolithic billing stacks, require heavy runtime containers (Node/Java/Python), demand external MongoDB/Redis servers, and lack strict ACID state machines—leading to race conditions and double-spending vulnerabilities.

| Feature | Legacy Payment Verifiers | **PayGate Platform** |
|---|---|---|
| **Runtime Architecture** | Heavy Node.js / NestJS / JVM | **Static Pure Go Binary** |
| **Idle Memory (RAM)** | ~185 MB – 500 MB | **~15 MB RAM** (91% lower) |
| **Database Requirement** | Mandatory MongoDB + Redis | **Pluggable: PostgreSQL, SQLite (No CGO), or In-Memory** |
| **Container Image Size** | ~380 MB – 800 MB | **28.4 MB Static Alpine Image** |
| **Double-Spend Protection** | In-memory cache / prone to races | **Database-enforced ACID conditional updates** |
| **Provisioning Guarantee** | Fire-and-forget API calls | **Transactional Durable Outbox with lease locks** |
| **Deployment Modes** | Complex multi-pod setups | **1-Click BeeCloud, Kubernetes App Pods, or Standalone** |
| **Headless Microservice** | Web UI tightly coupled | **Optional UI (`DISABLE_UI=true`) with preset env keys** |

---

## ✨ Core Features

- 🛡️ **Guaranteed Financial Correctness:** Strict state machine prevents double-spending and race conditions with database-level uniqueness constraints and atomic conditional updates (`WHERE claimed = 0` / `FOR UPDATE SKIP LOCKED`).
- 🚆 **Multi-Rail Provider Support (6 Live Rails + Sandbox):** Production-tested verifiers for:
  - **Commercial Bank of Ethiopia (CBE):** v2 API & receipt slip parser with compound receiver name validation.
  - **Telebirr (Ethio Telecom):** Direct receipt validator with bilingual regex engine and proxy support.
  - **Bank of Abyssinia (BOA):** Online slip verification with automated customer and merchant identity matching.
  - **Dashen Bank:** Live HTTP JSON verification against official Dashen receipt services (`receipts.dashenbanksc.com`).
  - **Awash Bank (Awash Birr Pro):** Online receipt portal verification parser on high-speed banking gateways.
  - **M-Pesa Safaricom Ethiopia:** Native B2C/C2B transaction receipt parser with pure Go zlib stream decompression.
  - **Mock Sandbox Rail:** Deterministic test rails with customizable latency, test fixtures, and error simulation.
- 🔌 **Pluggable Storage Layer:**
  - **Pure Go SQLite (Default):** Embedded zero-dependency database with WAL mode (`modernc.org/sqlite`, 100% CGO-free). Persisted in `./data/paygate.db`.
  - **PostgreSQL 16:** Production-scale clustered storage with connection pooling and horizontal worker concurrency.
  - **In-Memory Ephemeral Mode:** Automatic zero-crash fallback with safety warnings if requested.
  - **Live Database Switching:** Point PayGate to an external database directly from the Web Admin UI at runtime without server restarts!
- 📬 **Durable Provisioning Outbox:** Transactional outbox worker guarantees zero lost orders between payment confirmation and cloud service provisioning, complete with lease locking, exponential backoff retries, and HMAC-SHA256 signed webhooks.
- ⚙️ **Preset Keys & Headless Mode:** Run as a pure API microservice without UI overhead (`DISABLE_UI=true`) and configure static keys via environment variables (`API_KEY`, `PRESET_API_KEYS`, `MASTER_TOKEN`).
- 💻 **Modern Administrative & Developer Console:**
  - High-end dark theme inspired by Stripe, Svix, and Resend.
  - **Global Quick Verify Modal (`Cmd+K` / `Ctrl+K`):** Instantly test any bank slip or reference from anywhere in the console.
  - **Slide-Over Receipt Inspector Drawer:** Inspect full routing, settled amounts, counterparty avatars, claim status, and raw JSON slip payload.
  - **Interactive Developer Playground:** Multi-rail presets, live latency stopwatch, and one-click copyable code snippets in **cURL**, **TypeScript**, **Python**, and **Go**.
  - **Settlement Ledger & Queue:** Full visibility into Transactions, Provisioning Jobs, API Keys, Orders, and Immutable Audit Trails.

---

## 🏗️ Architecture & State Machine

```mermaid
flowchart TD
    Client["Client / Merchant Service"] -->|HTTPS / REST| Ingress["Ingress / Nginx / Direct (:5151)"]
    
    subgraph Core ["PayGate Core Engine (Go)"]
        Ingress --> MW["Middleware: RateLimiter, ReqID, Auth, CORS"]
        MW --> Router["Chi HTTP Router"]
        Router --> PS["PaymentService\n(State Machine, Claim, Tolerance)"]
        Router --> PrS["ProvisioningService\n(Durable Outbox Queue)"]
        Router --> KS["APIKeyService\n(SHA-256 Hashes, Quotas)"]
        
        PS --> Providers["Provider Registry\n(CBE, Telebirr, BOA, Mock)"]
        PrS --> Worker["Background Outbox Worker\n(Lease Locks, Exponential Retries)"]
    end
    
    subgraph Storage ["Pluggable ACID Storage Layer"]
        PS & PrS & KS --> TxM["TxManager"]
        TxM --> PG[("PostgreSQL 16\nFOR UPDATE SKIP LOCKED")]
        TxM --> SL[("SQLite WAL\nPure Go / No CGO")]
        TxM --> Mem[("In-Memory\nEphemeral Fallback")]
    end
    
    Worker -->|Signed HMAC-SHA256| Downstream["Merchant Provisioning Webhook / API"]
```

### Unidirectional Payment Lifecycle:
```mermaid
stateDiagram-v2
    [*] --> PENDING: Verification Requested
    PENDING --> VERIFIED: Provider Confirmed
    PENDING --> FAILED: Provider Rejected / Expired
    VERIFIED --> CLAIMED: Order Associated & Outbox Dispatched
    VERIFIED --> EXPIRED: Unclaimed After Window
    CLAIMED --> REFUNDED: Admin / Provider Reversal
    FAILED --> [*]
    EXPIRED --> [*]
    REFUNDED --> [*]
```

---

## 🚀 Quick Start

### 1. Standalone Instant Start (Zero Database Server Needed)

Run PayGate out-of-the-box with **zero database configuration**. It automatically uses embedded pure Go SQLite (or runs in-memory with a safety warning) and serves both the React Web UI and API:

```bash
# Clone the repository
git clone https://github.com/beecloud/paygate.git
cd paygate

# Start standalone stack
docker compose up --build -d
```

- **Web Dashboard:** `http://localhost:3000` (or `http://localhost:5151`)
- **Default Login:** `admin@paygate.dev` / `admin-password-change-me`
- **REST API:** `http://localhost:5151/api/v1`

> 💡 **Connect Your Existing Database:** Once the dashboard opens, navigate to the **Database** tab to connect PayGate to your existing PostgreSQL or SQLite database dynamically!

---

### 2. Full Enterprise Production Stack (`docker-compose-full.yml`)

Start the production stack with dedicated PostgreSQL 16 Alpine, connection pooling, persistent outbox queues, and report archives:

```bash
docker compose -f docker-compose-full.yml up --build -d
```

---

### 3. Independent Backend Microservice

Run authentic-et purely as a headless payment validation engine:

```bash
docker run -d \
  --name authentic-backend \
  -p 5151:5151 \
  -e DATABASE_DRIVER=sqlite \
  -e SQLITE_PATH=/data/authentic.db \
  -e JWT_SECRET=your-secret-key-min-32-chars \
  -v authentic_data:/data \
  ghcr.io/fransitsena/authentic-backend:latest
```

Or build the static Go binary directly:
```bash
cd backend
go build -o authentic-server ./cmd/server
./authentic-server
```

---

### 4. Independent Frontend Deployment

Deploy the React SPA independently to Vercel, Netlify, Cloudflare Pages, or AWS S3 / CloudFront:
1. Root directory: `frontend`
2. Build command: `npm run build`
3. Output directory: `dist`
4. Environment variable: `VITE_API_BASE_URL=https://api.yourdomain.com/api/v1`

---

### 5. Local Native Development (Go & Node.js)

```bash
# 1. Run Backend with embedded SQLite
cd backend
DATABASE_DRIVER=sqlite SQLITE_PATH=./dev.db go run ./cmd/server

# 2. In another terminal, run Frontend
cd ../frontend
npm install
npm run dev
```

Visit `http://localhost:5173` to explore the dashboard and interactive developer playground.

---

## ⚡ 60-Second Payment Verification Walkthrough

Verify a Commercial Bank of Ethiopia (CBE), Telebirr, or Bank of Abyssinia payment with a single cURL call:

```bash
curl -X POST http://localhost:5151/api/v1/public/payment/verify \
  -H "Content-Type: application/json" \
  -H "X-API-KEY: sk_pv_live_prod_key_123456" \
  -d '{
    "provider": "cbe",
    "transactionId": "FT240101ABCD",
    "expectedAmount": 250.00,
    "currency": "ETB",
    "autoClaim": true,
    "claimedBy": "order_9901"
  }'
```

#### JSON Response (`200 OK`):
```json
{
  "verified": true,
  "providerKey": "cbe",
  "transactionId": "FT240101ABCD",
  "amount": 250.00,
  "currency": "ETB",
  "status": "CLAIMED",
  "alreadyProcessed": false,
  "claimed": true,
  "claimedBy": "order_9901",
  "payerIdentifier": "ALEMU KEBEDE",
  "receiverIdentifier": "ACME CLOUD SERVICES"
}
```

---

## 🛠️ Command-Line Interface (`paygate-cli`)

PayGate includes a companion CLI for administration and zero-downtime database engine migrations:

```bash
cd backend && go build -o paygate-cli ./cmd/cli
```

### Create Administrator
```bash
./paygate-cli create-admin --email=ops@example.com --password="SecurePassword123!"
```

### Lossless Data Export & Import (SQLite $\leftrightarrow$ PostgreSQL)
```bash
# Export complete snapshot from SQLite
./paygate-cli export-data --driver=sqlite --sqlite-path=/data/paygate.db --output=backup.json

# Import into production PostgreSQL
./paygate-cli import-data --driver=postgres --database-url="postgres://user:pass@host:5432/db" --input=backup.json
```

---

## ⚙️ Configuration Reference

| Environment Variable | Default | Description |
|---|---|---|
| `PORT` | `5151` | Port for the HTTP API server |
| `DATABASE_DRIVER` | `postgres` | Persistence driver: `postgres`, `sqlite`, or `memory` |
| `DATABASE_URL` | - | PostgreSQL connection DSN (required if `DATABASE_DRIVER=postgres`) |
| `SQLITE_PATH` | `/data/paygate.db` | File path for SQLite database (required if `DATABASE_DRIVER=sqlite`) |
| `API_KEY` / `PRESET_API_KEYS` | - | Preset static API keys for headless usage without web UI |
| `MASTER_TOKEN` | - | Static administrator access token for scripts and automation |
| `DISABLE_UI` | `false` | Run in pure headless API mode without serving React frontend |
| `JWT_SECRET` | - | HMAC-SHA256 key for signing admin JWT tokens (min 32 bytes) |
| `ADMIN_EMAIL` | `admin@paygate.dev` | Initial admin user email (seeded automatically on startup) |
| `ADMIN_PASSWORD` | `admin-password-change-me` | Initial admin user password |
| `PAYMENT_PROVIDERS_ENABLED` | `mock,cbe,telebirr,boa` | Active payment provider rails |
| `UNDERPAY_TOLERANCE_ETB` | `1.0` | Allowed underpayment tolerance (-1.00 ETB for bank fees) |
| `OVERPAY_TOLERANCE_ETB` | `5.0` | Allowed overpayment tolerance (+5.00 ETB) |
| `WORKER_CONCURRENCY` | `5` | Maximum concurrent background provisioning dispatchers |
| `WORKER_POLL_INTERVAL` | `3s` | Interval for polling the outbox queue |
| `CLEANUP_ENABLED` | `true` | Automated data retention background cleaner enabled |
| `CLEANUP_INTERVAL_HOURS` | `24` | Interval (in hours) between automated retention purges |
| `TRANSACTION_RETENTION_DAYS` | `90` | Days to retain claimed payments (`0` or `-1` to retain indefinitely) |
| `FAILED_RETENTION_DAYS` | `30` | Days to retain failed/expired transaction records |
| `UNCLAIMED_EXPIRY_HOURS` | `72` | Hours after which stale pending payments transition to `EXPIRED` |
| `AUDIT_RETENTION_DAYS` | `180` | Days to retain immutable financial audit logs (`0` for indefinite) |
| `COMPLETED_JOBS_RETENTION_DAYS` | `14` | Days to retain successfully executed provisioning outbox jobs |
| `DRY_RUN_CLEANUP` | `false` | When true, previews prunable records without performing deletes |

### 🧹 Automated Data Retention & Safety Guarantees

PayGate includes an enterprise data retention lifecycle engine designed around financial correctness:

1. **Active Verification Window Safety:** Pending payments are never prematurely purged. Payments exceeding `UNCLAIMED_EXPIRY_HOURS` (default 72h) are transitioned to `EXPIRED` before becoming eligible for deletion via `FAILED_RETENTION_DAYS`.
2. **Batch & Engine Portability:** Cleanup queries execute cleanly across PostgreSQL, SQLite, and In-Memory without locking transaction tables or blocking WAL readers.
3. **Audit Trail Immutability:** Audit records are preserved for the full `AUDIT_RETENTION_DAYS` duration (default 180 days) for regulatory compliance.
4. **On-Demand Admin Execution & Dry Run:** Administrators can inspect prunable records with `POST /api/v1/database/cleanup` (`{"dryRun": true}`) or trigger an immediate purge via the Web UI Database page.

---

## 📚 Documentation Index

- [Existing System Assessment & Migration Roadmap](docs/ASSESSMENT.md)
- [Architecture & Design Invariants](docs/ARCHITECTURE.md)
- [REST API & Webhook Signature Reference](docs/API.md)
- [Deployment Guide (BeeCloud, K8s, Docker, Systemd)](docs/DEPLOYMENT.md)
- [Security Threat Model & Mitigations](docs/THREAT_MODEL.md)
- [Database Switching & MongoDB Migration Guide](docs/MIGRATION_GUIDE.md)
- [Performance & Concurrency Benchmarks](docs/BENCHMARKS.md)
- [Project Roadmap](docs/ROADMAP.md)

---

## 🤝 Contributing

We welcome contributions! Please review our [Contributing Guidelines](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).

1. Fork the repo and create your feature branch: `git checkout -b feat/my-new-feature`
2. Run backend conformance tests: `cd backend && go test -v ./...`
3. Run frontend typechecks and build: `cd frontend && npm run build`
4. Submit a Pull Request adhering to Conventional Commits.

---

## 🔒 Security

For security vulnerabilities, please refer to our [Security Policy](SECURITY.md) and contact [security@paygate.dev](mailto:security@paygate.dev).

---

## 📄 License

PayGate is open-source software licensed under the [Apache License 2.0](LICENSE).
