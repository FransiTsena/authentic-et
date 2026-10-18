# Stage 1: Builder
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/server ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/cli ./cmd/cli

# Stage 2: Minimal runtime
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata curl && \
    addgroup -S authentic && adduser -S authentic -G authentic

WORKDIR /home/authentic

COPY --from=builder /app/bin/server /usr/local/bin/authentic-server
COPY --from=builder /app/bin/cli /usr/local/bin/authentic-cli

# Persistent data directory for SQLite deployments
RUN mkdir -p /home/authentic/data && chown -R authentic:authentic /home/authentic

USER authentic

EXPOSE 5151

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:5151/health || exit 1

ENTRYPOINT ["/usr/local/bin/authentic-server"]
