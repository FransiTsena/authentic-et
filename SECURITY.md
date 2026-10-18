# Security Policy

The authentic-et team takes security, transaction integrity, and financial data protection extremely seriously. We appreciate the responsible disclosure of vulnerabilities by security researchers and the open-source community.

---

## 1. Supported Versions

We release security patches for the following versions:

| Version | Supported |
|---|---|
| 1.0.x | :white_check_mark: Active |
| < 1.0.0 | :x: End of Life |

---

## 2. Reporting a Vulnerability

**Please do NOT file public GitHub issues for security vulnerabilities.**

If you discover a vulnerability related to payment verification, state machine bypassing, double-spending, timing attacks, authentication bypass, or data leakage, please report it via our responsible disclosure channel:

* **Email:** [security@authentic-et.dev](mailto:security@authentic-et.dev)
* **Subject:** `[SECURITY] authentic-et Vulnerability Report - <Brief Description>`
* **PGP Key:** Available upon request or via keyserver.

### What to Include in Your Report:
1. **Description:** Clear explanation of the potential vulnerability and its impact.
2. **Reproduction Steps:** Step-by-step instructions, cURL commands, or a minimal proof-of-concept (PoC) script.
3. **Environment:** Operating system, Go version, database engine (PostgreSQL, SQLite, or In-Memory), and authentic-et version.
4. **Suggested Remediation:** If known, recommendations for resolving the issue.

---

## 3. Vulnerability Response Timeline

When a report is received, the maintainers will:
* **Initial Response:** Acknowledge receipt within **24 hours**.
* **Triage & Assessment:** Verify the vulnerability and determine severity within **72 hours**.
* **Fix & Release:** Prepare and release a security advisory and patch within **7 to 14 business days** (depending on severity).
* **Credit:** Publicly credit the reporter in the release notes and security advisory (unless the reporter prefers anonymity).

---

## 4. Built-in Security Safeguards

authentic-et includes multi-layered defenses by design:

* **Anti-Double-Spending:** Unique database constraints on transaction IDs coupled with atomic conditional updates (`WHERE claimed = 0` / `FOR UPDATE SKIP LOCKED`).
* **Timing-Safe Authentication:** All API keys and HMAC signatures are validated using `crypto/subtle.ConstantTimeCompare` to mitigate timing side-channel attacks.
* **One-Way Credential Hashing:** Raw merchant API keys (`sk_pv_...`) are never stored; only one-way `SHA-256` hashes are recorded in the database.
* **Signed Outbox Webhooks:** All provisioning notifications dispatched to downstream systems are signed with `HMAC-SHA256` and include timestamp skew protection.
* **Rate Limiting:** Sliding token-bucket rate limiter per IP address with configurable burst buffers.
