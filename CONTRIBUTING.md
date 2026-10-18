# Contributing to PayGate

Thank you for your interest in contributing to **PayGate**! We welcome bug reports, feature requests, documentation enhancements, and pull requests.

## Development Environment Setup

### Prerequisites
- **Go 1.22+**
- **Node.js 20+** & npm
- **Docker & Docker Compose** (optional, for containerized local testing)
- **Make** or bash

### Running Backend Locally
```bash
cd backend
# Run tests
go test -v ./...

# Run server with local SQLite (no Docker needed)
DATABASE_DRIVER=sqlite SQLITE_PATH=./dev.db go run ./cmd/server
```

### Running Frontend Locally
```bash
cd frontend
npm install
npm run dev
# Visit http://localhost:5173
```

## Architecture Principles
1. **Financial Precision & State Machine Integrity:**
   - All state transitions must adhere strictly to the Payment State Machine (`internal/domain/payment.go`).
   - Transactions must be atomic and protect against double spending via database-level uniqueness constraints and `claimed` flags.
2. **Database Agnostic Repositories:**
   - Both PostgreSQL and SQLite adapters must implement the exact same repository interfaces (`internal/domain/repository.go`).
   - All tests in `internal/database/conformance_test.go` must pass on any new adapter or schema alteration.
3. **Pure Go & Portability:**
   - Do not introduce CGO dependencies or external C libraries. The SQLite driver is pure Go (`modernc.org/sqlite`).
4. **Zero Vanity Metrics:**
   - Never artificially inflate statistics. Aggregations must reflect real database records.

## Submitting a Pull Request
1. Fork the repository and create your branch from `main`:
   ```bash
   git checkout -b feature/my-new-feature
   ```
2. Write unit tests for any new business logic or endpoints.
3. Ensure formatting and tests pass:
   ```bash
   cd backend && gofmt -s -w . && go test -v ./...
   cd ../frontend && npm run build
   ```
4. Commit your changes with clear, descriptive commit messages adhering to Conventional Commits:
   - `feat: add support for Telebirr QR validation`
   - `fix: prevent negative amount in tolerance matcher`
   - `docs: update migration guide for Postgres 16`
5. Open a Pull Request against `main`. CI will automatically validate your build and tests.
