# authentic-et Security & Threat Model

This document outlines the security architecture, trust boundaries, threat vectors, and protective countermeasures implemented in authentic-et.

---

## 1. Trust Boundaries & Actors

```mermaid
flowchart LR
    Untrusted[Untrusted Internet / Public Clients] -->|Boundary 1: Rate Limiter & Input Sanitizer| PublicAPI["Public API (/verify, /claim)"]
    Admin[Admin User] -->|Boundary 2: JWT Auth (HMAC-SHA256)| AdminAPI["Admin API (/admin/*)"]
    Merchant[Merchant Service] -->|Boundary 3: SHA-256 API Key| MerchantAPI["Merchant Integration"]
    Core[authentic-et Core] -->|Boundary 4: HMAC Webhook Signature| Downstream[Downstream Provisioning Systems]
    Core -->|Boundary 5: Isolated Adapter Interface| DB[(PostgreSQL / SQLite)]
```

---

## 2. Threat Vectors and Countermeasures

### 2.1 Double-Spending & Race Conditions
- **Threat:** An attacker issues simultaneous `/api/v1/public/payment/claim` requests using the same payment transaction reference to claim credit across multiple orders.
- **Countermeasure:**
  1. Unique database constraint on `payments.tx_id`.
  2. Single atomic transition `WHERE tx_id = ? AND claimed = 0` (or `claimed = false`).
  3. If another request attempts to claim the same transaction, 0 rows are affected and the application returns `409 Conflict` (`ErrPaymentAlreadyClaimed`).

### 2.2 Transaction ID Tampering & Provider Forgery
- **Threat:** An attacker crafts synthetic transaction strings or manipulates receipt parameters to claim unverified balances.
- **Countermeasure:**
  1. Input cleaning (`cleanTransactionID`) extracts purely canonical reference numbers from URLs or formatted slips, eliminating injection strings.
  2. Multi-rail provider verification queries bank APIs directly or scrapes provider receipts over TLS.
  3. Fuzzy and normalized merchant account verification (`isReceiverMatchingMerchant`) guarantees funds were sent to the business account, not a third-party account.

### 2.3 Webhook Interception & Replay Attacks
- **Threat:** A malicious middleman intercepts downstream provisioning notifications or replays old webhook events to re-provision cloud resources.
- **Countermeasure:**
  1. authentic-et calculates an `X-authentic-et-Signature` header using `HMAC-SHA256(payload, webhook_secret)`.
  2. authentic-et includes `X-authentic-et-Timestamp` in each webhook delivery. Downstream receivers must enforce a maximum 5-minute skew window.
  3. Every job contains an idempotent `id` and `order_id` so downstream consumers can reject duplicates.

### 2.4 Denial of Service & Brute-Force Attacks
- **Threat:** Automated scripts flood the verification endpoint to exhaust provider quotas or overload the database.
- **Countermeasure:**
  1. Token bucket in-memory rate limiter per IP address with burst control (default: 60 requests/minute).
  2. Bounded database connection pooling (`MaxOpenConns` = 25).
  3. Configurable worker concurrency pool using Go channels / semaphores preventing resource exhaustion.

### 2.5 API Key & Credential Compromise
- **Threat:** Attackers extract API keys from database backups or intercept plaintext credentials in transit.
- **Countermeasure:**
  1. Raw API keys (`sk_pv_...`) are **never stored** in the database. Only a one-way `SHA-256` digest is persisted.
  2. The plaintext key is displayed exactly once upon generation in the admin dashboard and never recoverable.
  3. Admin passwords are encrypted using `bcrypt` with cost 12.
  4. Timing-safe equality checks (`crypto/subtle.ConstantTimeCompare`) prevent timing side-channel attacks on credentials.

### 2.6 Server-Side Request Forgery (SSRF) & Receipt Impersonation
- **Threat:** An attacker crafts synthetic receipt URLs pointing to an attacker-controlled HTTP server containing mock bank response HTML.
- **Countermeasure:**
  1. Provider endpoints strictly pin external queries to hardcoded, official banking subdomains (`apps.cbe.com.et:100`, `transactioninfo.ethiotelecom.et`, `cs.bankofabyssinia.com`).
  2. Callers cannot supply custom remote URLs to fetch; queries strictly accept only alphanumeric transaction identifiers that are regex-sanitized (`^[A-Za-z0-9_\-]+$`) before making outbound requests.

### 2.7 Brute-Force Password Guessing & Unrestricted Account Creation
- **Threat:** Automated bots repeatedly guess admin passwords on `/api/v1/auth/login` or register unauthorized accounts on `/api/v1/auth/register`.
- **Countermeasure:**
  1. IP-level sliding-window rate limiting on `/auth/login` (burst 10, refill rate 1 attempt every 5 seconds).
  2. Open registration is **disabled by default** (`ALLOW_REGISTRATION=false`). Once the initial admin account is established, public registration requests are rejected with `403 Forbidden`.
  3. Non-admin users are strictly restricted to `free` API key tiers with enforced rate and quota ceilings (max 10 RPS, 5000 requests/month).

### 2.8 Unbounded Memory Exhaustion (DoS / OOM)
- **Threat:** Attackers stream gigabytes of raw JSON to API endpoints to exhaust pod memory.
- **Countermeasure:**
  1. All JSON request parsing is bounded using `io.LimitReader(r.Body, 1 << 20)` (1 MB payload cap). Requests exceeding 1 MB are terminated immediately.

### 2.9 Information Disclosure & Schema Leakage
- **Threat:** Unauthenticated attackers probe endpoints to discover database connection strings, server paths, or SQL error structures.
- **Countermeasure:**
  1. `/api/v1/public/database/status` reveals only safe boolean operational flags (`isEphemeral`, `connected`, `driver`). Detailed connection DSNs and error diagnostics are restricted to authenticated admins (`RequireAdmin`).
  2. In production, 500-level error messages return generic failure responses while logging full debugging details server-side.

### 2.10 Browser-Side Hardening & Security Headers
- **Threat:** Clickjacking, MIME-confusion, or malicious script execution in modern browsers.
- **Countermeasure:**
  1. Global middleware enforces: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 0`, `Referrer-Policy: strict-origin-when-cross-origin`, `Cross-Origin-Opener-Policy: same-origin`, and `Permissions-Policy`.
  2. CORS headers reject `AllowCredentials: true` whenever wildcard `*` origins are configured.

---

## 3. Financial Integrity Auditing

Every state change produces an immutable audit event (`internal/domain/audit.go`):
- Action performed (e.g. `PAYMENT_VERIFIED`, `PAYMENT_CLAIMED`, `API_KEY_CREATED`).
- Actor identifier (IP address, user ID, or API key).
- Metadata payload containing raw provider responses and previous vs new states.
- Audit records are append-only; update and delete operations are rejected by repository constraints.
