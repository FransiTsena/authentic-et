# PayGate REST API Reference

The PayGate platform provides a versioned REST API rooted at `/api/v1`. An interactive OpenAPI 3.0 specification is available at `/api/v1/openapi.json`.

---

## 1. Authentication & Headers

| Header | Description | Required By |
|---|---|---|
| `Authorization: Bearer <jwt>` | Standard JWT token returned from `/api/v1/admin/auth/login` | Admin endpoints (`/api/v1/admin/*`) |
| `X-API-Key: sk_pv_...` | SHA-256 validated merchant secret key | Merchant programmatic endpoints |
| `X-Request-ID: <uuid>` | Optional client-generated correlation ID (PayGate generates one if omitted) | All endpoints |

---

## 2. Database Management & Dynamic Configuration Endpoints

### 2.1 Get Database Engine Status
Returns current active database driver, health, masked connection strings, and ephemeral status.

```http
GET /api/v1/public/database/status
```

#### Response (`200 OK`):
```json
{
  "driver": "postgres",
  "isEphemeral": false,
  "warningMessage": "",
  "connected": true,
  "maskedDsn": "postgres://paygate:******@localhost:5432/paygate?sslmode=disable",
  "connectedAt": "2026-10-18T10:00:00Z"
}
```

### 2.2 Test Database Connection
Tests connectivity and ping to an external database without switching the live database.

```http
POST /api/v1/admin/database/test
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "driver": "postgres",
  "dsn": "postgres://user:password@external-db.example.com:5432/paygate?sslmode=require"
}
```

### 2.3 Dynamically Switch / Configure Database
Connects to an external database, applies migrations on the fly, and switches the live repository pointers with zero server downtime.

```http
POST /api/v1/admin/database/configure
Authorization: Bearer <jwt>
Content-Type: application/json

{
  "driver": "postgres",
  "dsn": "postgres://user:password@external-db.example.com:5432/paygate?sslmode=require"
}
```

---

## 3. Public Payment Endpoints

### 3.1 Verify Payment
Initiates independent verification against the provider rail (CBE, Telebirr, Bank of Abyssinia, or Mock Sandbox).

```http
POST /api/v1/public/payment/verify
Content-Type: application/json

{
  "provider": "cbe",
  "tx_id": "FT240101ABCD",
  "expected_amount": 250.00,
  "currency": "ETB",
  "order_id": "ord_9901"
}
```

#### Response (`200 OK`):
```json
{
  "id": "pay_5a732dfc-8c01-443b-8105-1a3b900b7410",
  "tx_id": "FT240101ABCD",
  "provider": "cbe",
  "amount": 250.00,
  "currency": "ETB",
  "sender_name": "ALEMU KEBEDE",
  "sender_account": "1000****1234",
  "receiver_name": "ACME CLOUD SERVICES",
  "receiver_account": "1000987654321",
  "status": "VERIFIED",
  "claimed": false,
  "created_at": "2026-10-18T10:00:00Z"
}
```

---

### 2.2 Claim Payment
Atomically associates a verified payment with an order or customer entitlement.

```http
POST /api/v1/public/payment/claim
Content-Type: application/json

{
  "tx_id": "FT240101ABCD",
  "order_id": "ord_9901",
  "customer_id": "cust_1001"
}
```

#### Response (`200 OK`):
```json
{
  "status": "CLAIMED",
  "claimed_by": "ord_9901",
  "claimed_at": "2026-10-18T10:01:05Z",
  "tx_id": "FT240101ABCD"
}
```

#### Error Response (`409 Conflict`):
```json
{
  "error": "payment has already been claimed for another order",
  "code": "PAYMENT_ALREADY_CLAIMED"
}
```

---

## 3. Provisioning Webhooks & Signature Verification

When a payment is confirmed or an order is claimed, PayGate dispatches a signed HTTP POST to the configured webhook URL.

### Webhook Headers
```http
X-PayGate-Signature: 7c5417e923e2a05cf4b2d3e9112850989cb51034c4424364be73e5bf0641be2a
X-PayGate-Timestamp: 1729245600
X-PayGate-Delivery: deliv_77123
```

### Signature Verification Code Snippet (Node.js)
```typescript
import crypto from 'crypto';

export function verifyWebhook(rawPayload: string, signature: string, secret: string): boolean {
  const hmac = crypto.createHmac('sha256', secret);
  const digest = hmac.update(rawPayload).digest('hex');
  return crypto.timingSafeEqual(Buffer.from(signature, 'hex'), Buffer.from(digest, 'hex'));
}
```

### Signature Verification Code Snippet (Go)
```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

func VerifyWebhook(payload []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```
