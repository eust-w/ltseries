# Contributing to LTSeries

Thank you for helping build a practical AI production workspace.

## Before You Start

Read the [README](README.md) and [API reference](docs/frontend-api.md). Open an
issue before making a large architectural or workflow change. For small fixes,
a focused pull request is welcome.

## Local Setup

1. Configure the root `.env` from `.env.example`.
2. Start MongoDB with `./scripts/start-local.sh`.
3. Run `go run ./cmd/server` inside `backend`.
4. Run `npm ci && npm run dev` inside `LeriesStudio`.

The integration tests use a real MongoDB instance and create disposable databases.
Use `LTSERIES_TEST_MONGODB_URI` to select a dedicated test server; the default is
`mongodb://127.0.0.1:27019`.

## Required Checks

```bash
(cd backend && go test ./... && go vet ./...)
(cd LeriesStudio && npm run lint && npm run build && npm audit)
docker compose config --quiet
```

Use `gofmt` on changed Go files. Add regression coverage for behavioral fixes.
Tests for provider payloads may use HTTP test servers; shipped generation paths
must call the configured providers and must never fabricate successful results.

## Pull Requests

Describe the problem, the change, and how you verified it. Include before/after
screenshots for UI changes and update API docs when request or response contracts
change. Keep unrelated formatting and dependency upgrades out of feature PRs.

Never commit credentials, local databases, logs containing access tokens,
provider-signed media URLs, or generated user media. Use redacted request samples
and configuration examples. The existing `.gitignore` excludes runtime data.

## Helpful Contributions

- Reproducible provider compatibility fixes.
- Reliable local and container deployment.
- Asset-reference selection and shot review workflows.
- English UI, documentation and accessibility.
- Reproducible bug reports with model name, task type and sanitized errors.

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
