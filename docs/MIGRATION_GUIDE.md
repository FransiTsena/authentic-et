# PayGate Migration & Engine Switching Guide

This guide details how to:
1. Migrate from the legacy BeeCloud payment verifier (Node.js/Nest.js + MongoDB) to PayGate.
2. Safely switch database engines between SQLite and PostgreSQL without data loss.

---

## 1. Migrating from Legacy BeeCloud (MongoDB)

Legacy BeeCloud stored transactions in MongoDB collections `transactions`, `apikeys`, and `users`. PayGate utilizes normalized, relational SQL tables (`payments`, `api_keys`, `orders`, `provisioning_jobs`, `audit_events`, `users`).

### Schema Mapping Reference

| Legacy MongoDB Field | PayGate PostgreSQL/SQLite Field | Notes |
|---|---|---|
| `transactions.reference` | `payments.tx_id` | Unique alphanumeric transaction ID |
| `transactions.method` | `payments.provider` | `cbe`, `telebirr`, `boa` |
| `transactions.amount` | `payments.amount` | Stored as numeric/decimal with 2 decimal precision |
| `transactions.status` | `payments.status` | Maps to `VERIFIED`, `PENDING`, `FAILED`, `CLAIMED` |
| `transactions.orderId` | `payments.claimed_by` | Associated order identifier |
| `transactions.used` | `payments.claimed` | Boolean flag (0/1 in SQLite, true/false in Postgres) |
| `apikeys.keyHash` | `api_keys.key_hash` | SHA-256 digest |
| `apikeys.tier` | `api_keys.role` | Access control role |

### Migration Steps:
1. Stop incoming traffic to legacy payment verifier or place in maintenance mode.
2. Dump legacy MongoDB collection:
   ```bash
   mongoexport --uri="mongodb://localhost:27017/beecloud" --collection="transactions" --out=transactions.json
   ```
3. Use the PayGate CLI to load and validate records:
   ```bash
   paygate-cli import-legacy-mongo --file=transactions.json --target=postgres --dsn="$DATABASE_URL"
   ```
4. Verify data integrity:
   ```bash
   paygate-cli verify-integrity
   ```

---

## 2. Lossless Database Engine Switching (SQLite $\leftrightarrow$ PostgreSQL)

Switching the `DATABASE_DRIVER` environment variable without an explicit data export will result in a fresh empty database. PayGate includes a built-in CLI command to export and import complete database snapshots losslessly.

```mermaid
flowchart LR
    Source[(Source DB\ne.g. SQLite)] -->|paygate-cli export-data| Snapshot[Encrypted JSON Snapshot]
    Snapshot -->|paygate-cli import-data| Target[(Target DB\ne.g. PostgreSQL)]
```

### Scenario A: Migrating from Local SQLite to Production PostgreSQL

1. **Create Snapshot from SQLite:**
   ```bash
   paygate-cli export-data \
     --driver=sqlite \
     --sqlite-path=/data/paygate.db \
     --output=paygate-backup.json
   ```

2. **Validate Snapshot:**
   The output JSON contains transaction state hashes, row counts, and checksums.
   ```bash
   head -n 20 paygate-backup.json
   ```

3. **Import Snapshot to PostgreSQL:**
   Ensure PostgreSQL is running with migrations applied:
   ```bash
   paygate-cli import-data \
     --driver=postgres \
     --database-url="postgres://paygate:secret@postgres.internal:5432/paygate?sslmode=require" \
     --input=paygate-backup.json
   ```

4. **Verify Conformance:**
   ```bash
   paygate-cli verify-integrity --driver=postgres --database-url="..."
   ```

5. **Update Environment Variable:**
   ```bash
   DATABASE_DRIVER=postgres
   DATABASE_URL="postgres://paygate:secret@postgres.internal:5432/paygate?sslmode=require"
   ```
