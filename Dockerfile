# ==============================================================================
# PayGate Unified All-In-One Container
# Builds React SPA + Go Backend into a single ultra-lightweight Alpine image
# Perfect for Kubernetes App Pods and 1-Click Deployments (Port 8080)
# ==============================================================================

# Stage 1: Build React Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.22-alpine AS backend-builder
WORKDIR /build
RUN apk add --no-cache git ca-certificates tzdata
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /build/bin/paygate-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /build/bin/paygate-cli ./cmd/cli

# Stage 3: Minimal Single-Container Runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S paygate && adduser -S paygate -G paygate

WORKDIR /home/paygate

COPY --from=backend-builder /build/bin/paygate-server /usr/local/bin/paygate-server
COPY --from=backend-builder /build/bin/paygate-cli /usr/local/bin/paygate-cli
COPY --from=frontend-builder /build/dist /usr/share/paygate/html

# Persistent data directory for SQLite deployments
RUN mkdir -p /data && chown -R paygate:paygate /data /usr/share/paygate/html

USER paygate

ENV PORT=5151
ENV STATIC_DIR=/usr/share/paygate/html
ENV SQLITE_PATH=/data/paygate.db

EXPOSE 5151

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:5151/health || exit 1

ENTRYPOINT ["/usr/local/bin/paygate-server"]
