# Contributing to authentic-et

Thank you for your interest in contributing to **authentic-et**! We welcome bug reports, new Ethiopian banking adapters, documentation enhancements, and pull requests.

---

## 🔒 Branch Protection & Governance Rules

To guarantee financial correctness, release stability, and security:

1. **`main` Branch is Strictly Protected & Locked**:
   - Direct commits and direct pushes to `main` are disabled for all contributors.
   - All code changes must arrive via Pull Requests targeting the `main` branch.
2. **Authorized Merges Only**:
   - Only the repository owner and core maintainer (**`@FransiTsena`**) is authorized to approve and merge Pull Requests into `main`.
   - The repository uses a strict [`.github/CODEOWNERS`](.github/CODEOWNERS) policy.
3. **Automated CI Quality Gates**:
   - Every Pull Request automatically triggers the continuous integration pipeline (`.github/workflows/ci.yml`).
   - All tests, type checks, lint checks, and container builds must pass with zero failures before a Pull Request can be merged.
4. **Continuous Public Deployment Pipeline**:
   - When a Pull Request is approved and merged into `main` by `@FransiTsena`, the automated release pipeline (`.github/workflows/release.yml`) immediately builds and publishes updated multi-arch Docker images to the GitHub Container Registry (`ghcr.io/fransitsena/authentic-et`) and generates cross-compiled binary releases for public use.

---

## 🛠️ Local Development Setup

### Prerequisites
- **Go 1.22+**
- **Node.js 20+** & npm
- **Docker & Docker Compose** (optional, for containerized local testing)

### Running Backend Locally
The backend operates on standard port **`5151`**:
```bash
cd backend

# Run all unit, security, and conformance tests
go test -race -v ./...

# Run server with local embedded SQLite
DATABASE_DRIVER=sqlite SQLITE_PATH=./data/dev.db PORT=5151 go run ./cmd/server
```

### Running Frontend Locally
```bash
cd frontend
npm install
npm run dev
# Vite runs with automatic proxy to backend on port 5151
```

---

## 📐 Architecture & Correctness Principles

1. **Financial Precision & State Machine Integrity:**
   - All state transitions must adhere strictly to the Payment State Machine (`internal/domain/payment.go`).
   - Transactions must be atomic and defend against double spending via database-level uniqueness constraints and `WHERE claimed = 0` conditional updates.
2. **Standardized Port 5151:**
   - Always respect standard port **`5151`** for the authentic-et service. Avoid introducing arbitrary default ports.
3. **Database Agnostic Repositories:**
   - Both PostgreSQL and SQLite adapters must implement the exact same repository interfaces (`internal/domain/repository.go`).
   - All tests in `internal/database/conformance_test.go` must pass on any new adapter or schema alteration.
4. **Pure Go & Portability:**
   - Do not introduce CGO dependencies or external C libraries. The SQLite driver is pure Go (`modernc.org/sqlite`).
5. **No Vanity Metrics:**
   - Aggregations and metrics must reflect real database records.

---

## 🚀 Submitting a Pull Request

1. **Fork the repository** and create your branch from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. **Implement your changes** and write corresponding unit tests.
3. **Validate locally** before opening a PR:
   ```bash
   # 1. Format and test backend
   cd backend && gofmt -s -w . && go test -race -v ./...
   
   # 2. Typecheck and build frontend
   cd ../frontend && npm run build
   ```
4. **Commit with Conventional Commits**:
   - `feat: add support for Awash Bank QR receipt verification`
   - `fix: handle edge case in Telebirr Amharic payer slip regex`
   - `docs: add Vercel frontend deployment instructions`
5. **Open your Pull Request**:
   - Ensure the description explains the rationale, changes made, and links any relevant issues.
   - Wait for automated CI status checks to complete.
   - `@FransiTsena` will review and merge when ready.
