# ==============================================================================
# authentic-et Unified All-In-One Container
# Builds React SPA + Go Backend into a single ultra-lightweight Alpine image
# Standard Port: 5151
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
    -o /build/bin/authentic-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /build/bin/authentic-cli ./cmd/cli

# Stage 3: Minimal Single-Container Runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S authentic && adduser -S authentic -G authentic

WORKDIR /home/authentic

COPY --from=backend-builder /build/bin/authentic-server /usr/local/bin/authentic-server
COPY --from=backend-builder /build/bin/authentic-cli /usr/local/bin/authentic-cli
COPY --from=frontend-builder /build/dist /usr/share/authentic/html

# Persistent data directory for SQLite deployments
RUN mkdir -p /data && chown -R authentic:authentic /data /usr/share/authentic/html

USER authentic

ENV PORT=5151
ENV STATIC_DIR=/usr/share/authentic/html
ENV SQLITE_PATH=/data/authentic.db

EXPOSE 5151

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:5151/health || exit 1

ENTRYPOINT ["/usr/local/bin/authentic-server"]
