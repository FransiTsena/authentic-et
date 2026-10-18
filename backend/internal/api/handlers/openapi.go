package handlers

import (
	"net/http"
)

const openAPISpecJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "PayGate: Open-Source Payment Verification & Provisioning Platform",
    "version": "1.0.0",
    "description": "Production-grade, secure, multi-provider payment verification and automated service provisioning platform."
  },
  "servers": [
    { "url": "/api/v1", "description": "API v1 root" }
  ],
  "components": {
    "securitySchemes": {
      "ApiKeyAuth": {
        "type": "apiKey",
        "in": "header",
        "name": "X-API-KEY"
      },
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "schemas": {
      "VerifyPaymentRequest": {
        "type": "object",
        "required": ["provider", "transactionId"],
        "properties": {
          "provider": { "type": "string", "example": "cbe" },
          "transactionId": { "type": "string", "example": "FT262036S1GH10104" },
          "expectedAmount": { "type": "number", "example": 500.0 },
          "referenceId": { "type": "string", "example": "order_123" },
          "referenceType": { "type": "string", "example": "order_payment" },
          "autoClaim": { "type": "boolean", "example": true },
          "claimedBy": { "type": "string", "example": "user_456" }
        }
      },
      "PaymentVerificationResult": {
        "type": "object",
        "properties": {
          "verified": { "type": "boolean" },
          "providerKey": { "type": "string" },
          "transactionId": { "type": "string" },
          "amount": { "type": "number" },
          "currency": { "type": "string" },
          "status": { "type": "string" },
          "alreadyProcessed": { "type": "boolean" },
          "claimed": { "type": "boolean" },
          "payerIdentifier": { "type": "string" },
          "receiverIdentifier": { "type": "string" },
          "timestamp": { "type": "string" }
        }
      }
    }
  },
  "paths": {
    "/public/payment/verify": {
      "post": {
        "summary": "Verify payment transaction across providers",
        "security": [{ "ApiKeyAuth": [] }, {}],
        "requestBody": {
          "required": true,
          "content": { "application/json": { "schema": { "$ref": "#/components/schemas/VerifyPaymentRequest" } } }
        },
        "responses": {
          "200": {
            "description": "Verification result",
            "content": { "application/json": { "schema": { "$ref": "#/components/schemas/PaymentVerificationResult" } } }
          }
        }
      }
    },
    "/public/payment/claim": {
      "post": {
        "summary": "Claim a verified transaction to prevent double spending",
        "security": [{ "ApiKeyAuth": [] }, {}],
        "responses": { "200": { "description": "Claim result" } }
      }
    },
    "/public/providers": {
      "get": {
        "summary": "List supported payment providers",
        "responses": { "200": { "description": "Supported providers list" } }
      }
    },
    "/stats": {
      "get": {
        "summary": "Get real platform verification metrics",
        "responses": { "200": { "description": "Verification statistics" } }
      }
    }
  }
}`

func OpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(openAPISpecJSON))
}
