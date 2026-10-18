---
name: Bug Report
about: Create a report to help us improve PayGate
title: '[BUG] '
labels: ['bug', 'triage']
assignees: ''
---

## Describe the Bug
A clear and concise description of what the bug is.

## Steps to Reproduce
Steps to reproduce the behavior:
1. Go to '...'
2. Send request '...'
3. See error

## Minimal Reproduction Command (cURL / Code)
```bash
curl -X POST http://localhost:5151/api/v1/public/payment/verify \
  -H "Content-Type: application/json" \
  -d '{ ... }'
```

## Expected Behavior
A clear and concise description of what you expected to happen.

## Actual Behavior
Logs or error response received.

## Environment Details:
- **PayGate Version:** (e.g. `v1.0.0` or commit hash)
- **Database Driver:** (`postgres`, `sqlite`, or `memory`)
- **OS / Architecture:** (e.g. Ubuntu 22.04 AMD64, macOS ARM64, Docker Alpine)
- **Deployment Type:** (Docker Compose, Kubernetes App Pod, Native binary)

## Additional Context
Add any other context, logs, or screenshots about the problem here.
