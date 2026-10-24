# ltseries Go React MongoDB Implementation Plan

**Goal:** Build `ltseries` as a new full-stack project using Go, Gin, MongoDB, and React, without Python services.

**Architecture:** React/Vite frontend talks to a Go/Gin API. MongoDB stores projects, storyboard shots, assets, and task queue documents. A Go worker claims pending task documents atomically from MongoDB, runs real provider calls through environment-configured adapters, and persists success/failure history for refresh and restart recovery.

**Tech Stack:** Go 1.24, Gin, MongoDB Go Driver, React 19, Vite, TypeScript, Docker Compose.

---

## Data Model

- `projects`: title, source script, analysis result, assets, shots, timestamps.
- `tasks`: task id, type, status, progress, error, project id, payload, result, timestamps, worker id, attempt count.
- `settings`: optional persisted provider settings can be added later; initial implementation uses environment variables.

## API

- `GET /api/health`: service and MongoDB status.
- `POST /api/projects`: create a project.
- `GET /api/projects`: list projects.
- `GET /api/projects/:id`: get one project.
- `POST /api/projects/:id/analyze`: enqueue real LLM script analysis.
- `POST /api/projects/:id/assets`: enqueue asset generation task metadata.
- `POST /api/projects/:id/videos`: enqueue video generation task metadata.
- `GET /api/tasks`: list task history.
- `GET /api/tasks/:id`: get task status and result/error.

## Provider Rules

- No mock, simulation, or hardcoded generated output.
- LLM analysis uses an OpenAI-compatible HTTP endpoint configured by `LLM_BASE_URL`, `LLM_API_KEY`, and `LLM_MODEL`.
- Image/video generation tasks require real provider endpoints configured via env. If not configured, tasks fail with a persisted configuration error instead of returning fake media.

## Implementation Steps

1. Write Go integration tests for MongoDB-backed task enqueue, claim, completion, and failure.
2. Write Go API tests for project creation, analyze enqueue, and task history.
3. Implement config, Mongo connection, repositories, queue, worker, provider clients, and Gin routes.
4. Build React frontend with project creation, project detail, task history, and polling.
5. Add Docker Compose for MongoDB, backend, and frontend.
6. Run tests and start local services for acceptance.

## Acceptance Criteria

- `go test ./...` passes.
- `npm run build` and `npm run typecheck` pass in `frontend`.
- `docker compose` or local scripts can start MongoDB, backend, and frontend.
- Creating a project persists in MongoDB.
- Enqueuing analysis creates a MongoDB task and worker records `processing` then `completed` or `failed`.
- Task success/failure is visible after page refresh because the frontend reads `/api/tasks`.
