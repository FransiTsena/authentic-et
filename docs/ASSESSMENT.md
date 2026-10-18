# Assessment of the Existing BeeCloud Payment Verifier & Provisioning System

## 1. Executive Summary

This assessment evaluates the payment verification, billing, and provisioning architecture in the BeeCloud codebase (`apps/backend` and `apps/frontend`). The existing system connects customer payments across Ethiopian payment rails (Commercial Bank of Ethiopia, Telebirr, Bank of Abyssinia) to automated service activation.

The objective is to extract, decouple, and transform this system into a standalone, production-grade, open-source platform written in **Go**, **React**, **TypeScript**, and **PostgreSQL** (with optional **SQLite** support), named **PayGate**.

---

## 2. Existing Architecture & Dependencies

| Area | Current Implementation | Target Open-Source Platform (PayGate) |
|---|---|---|
| **Runtime & Language** | NestJS (TypeScript / Node.js 20+) | Go 1.22+ (Multi-stage minimal binary, ~25MB image) |
| **Database** | MongoDB / Mongoose ODM (`ObjectId`, schema hooks) | Pluggable SQL: PostgreSQL default (`pgx/v5`), SQLite (`modernc.org/sqlite`) |
| **Transaction Boundaries** | No multi-document ACID transactions; ad-hoc updates | Strict ACID database transactions for payment + outbox jobs |
| **State Machine** | Implicit status strings (`pending`, `paid`, `failed`) | Explicit, validated state machine with formal transition guards |
| **Background Processing** | In-process polling & event emitter (`Subject<OrderPaidEvent>`) | Durable transactional outbox (`provisioning_jobs`), lease locks, backoff |
| **Rate Limiting & Quotas** | In-memory `Map<string, {count, resetAt}>` (lost on reboot) | Atomic token bucket / sliding window in database or Redis-compatible adapter |
| **Frontend** | Monolithic React/Vite app mixed with VPS/Hosting/SMS | Dedicated React/Vite SPA with typed client, playground, and admin console |

---

## 3. Critical Workflows to Preserve

### 3.1 Multi-Provider Verification
1. **Commercial Bank of Ethiopia (CBE)**:
   - Modern Public Mb v2 API: `https://Mb.cbe.com.et/api/v1/transactions/public/transaction-detail/{token}` with headers `X-App-ID` and `X-App-Version`.
   - Legacy Receipt Fallback: `https://apps.cbe.com.et:100/?id={FT_NUMBER}{ACCOUNT_SUFFIX}` supporting HTML slip parsing and PDF invoice parsing.
2. **Telebirr (Ethio Telecom)**:
   - Receipt Web Verification: `https://transactioninfo.ethiotelecom.et/receipt/{id}` with resilient layout regex parsing.
   - Payment Session creation and webhook callbacks.
3. **Bank of Abyssinia (BOA)**:
   - Online Slip API: `https://cs.bankofabyssinia.com/api/onlineSlip/getDetails/?id={id}`.
4. **Mock / Sandbox Engine**:
   - Deterministic test transactions (`TEST_SUCCESS`, `TEST_CLAIMED`, `TEST_NOTFOUND`, `TEST_INSUFFICIENT`) for local development, automated testing, and developer integration without burning bank funds.

### 3.2 Transaction Cleaning & Merchant Verification
- **Receipt normalization**: Parsing raw text, URLs (e.g. `mbreciept.cbe.com.et/v2-xxx` or `cs.bankofabyssinia.com/slip/?trx=FTxxx`), extracting query parameters and tokens.
- **Amount tolerance**: Tolerates -1.00 ETB (for banking apps without cent support) and +5.00 ETB overpayment.
- **Merchant verification**: Compound name matching preventing false positives, configurable via environment variables (`PAYMENT_MERCHANT_NAME`, `PAYMENT_VALID_RECEIVERS`, `BANK_TRANSFER_ACCOUNT_NAME`).

### 3.3 Idempotency & Double-Spending Safeguards
- Single-claim constraint: Once a transaction is verified and claimed, it cannot be claimed by another order or user.
- Atomic claim execution: Conditional update `WHERE claimed = false` preventing concurrent double claims.

### 3.4 Durable Service Provisioning (Transactional Outbox)
- When payment is confirmed, a provisioning job is atomically written to the outbox.
- Workers claim jobs using a lease lock (`locked_at`), execute the downstream provisioning handler, retry with exponential backoff on failure, and mark `manual_review` if maximum attempts (5) are exhausted.

---

## 4. Identified Defects, Security Weaknesses & Tech Debt

1. **Artificial Metrics Inflation**:
   In `PaymentGatewayService.getVerificationStats()`, statistics were artificially inflated (`totalVerifications + 1250`, `successfulVerifications + 1235`). In PayGate, all statistics will be genuine, computed from actual database records.
2. **In-Memory Rate Limiting**:
   The existing system used `Map<string, ...>` for rate limiting and daily quotas. Multiple server instances did not share quota state, and restarts wiped out usage counters.
3. **Tight Coupling to MongoDB**:
   Direct use of `Types.ObjectId`, `$or`, `$set`, and Mongoose models prevented deploying on relational databases.
4. **Monolithic Entanglement**:
   The payment verifier was coupled to internal BeeCloud VPS, DNS, and support services.
5. **Missing Database-Enforced Locking**:
   Concurrent verification and claim requests could execute race conditions without explicit transaction locks.

---

## 5. Migration & Decoupling Plan

1. **Backend Decoupling**:
   - Build Go backend with clean domain boundaries (`domain`, `provider`, `service`, `database`, `api`, `worker`).
   - Define repository contracts that abstract persistence from business logic.
   - Implement PostgreSQL (using `pgx/v5`) and SQLite (using `modernc.org/sqlite` purego) adapters.
2. **Durable Outbox & Webhooks**:
   - Implement transactional job dispatcher with worker lease locks and exponential backoff.
   - Implement webhook notifications with HMAC-SHA256 signatures for downstream services.
3. **Frontend Rebuilding**:
   - Recreate the payment verifier administrative and developer portal in React + TypeScript + Vite.
   - Include Dashboard, Explorer, Details, Live Playground, API Keys, Provisioning Jobs, Webhook Logs, and Database Health.
4. **Containerization & Release Automation**:
   - Docker Compose with PostgreSQL (standard) and SQLite (lightweight).
   - Comprehensive documentation (`README.md`, `ARCHITECTURE.md`, `API.md`, `THREAT_MODEL.md`, `MIGRATION_GUIDE.md`).
