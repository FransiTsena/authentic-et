## Description
Briefly describe the changes made in this Pull Request and the issue or requirement it addresses.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Performance improvement / Refactoring

## Financial Correctness & Security Checklist
- [ ] State machine transitions adhere strictly to `internal/domain/payment.go`.
- [ ] No regression on anti-double-spending protections (atomic updates / uniqueness constraints).
- [ ] No plaintext storage of secrets or API keys.
- [ ] All database queries work across both PostgreSQL and SQLite.

## Testing
- [ ] Unit tests added or updated.
- [ ] `go test -v ./...` in `backend` passes.
- [ ] `npm run build` in `frontend` passes.
