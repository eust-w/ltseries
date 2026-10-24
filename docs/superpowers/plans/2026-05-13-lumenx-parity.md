# LumenX Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the highest-impact LumenX feature gaps in ltseries while keeping the Go/Mongo/React architecture.

**Architecture:** Add missing operations to `ProjectService`, expose them through Gin routes, and connect critical UI controls in `ProjectWorkspace`. Keep long-running AI work on the existing Mongo-backed task queue.

**Tech Stack:** Go 1.24, Gin, MongoDB, React/Vite, TypeScript, FFmpeg.

---

### Task 1: Asset Variant Operations

**Files:**
- Modify: `backend/internal/services/project_service.go`
- Modify: `backend/internal/httpapi/router.go`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/components/ProjectWorkspace.tsx`
- Test: `backend/internal/httpapi/api_test.go`

- [x] Add failing API tests for delete/favorite variant.
- [x] Implement ProjectService variant delete/favorite by asset slot.
- [x] Add routes and frontend calls.
- [x] Add UI buttons on variant thumbnails.

### Task 2: Asset Video And Batch Asset Generation

**Files:**
- Modify: `backend/internal/services/project_service.go`
- Modify: `backend/internal/httpapi/router.go`
- Modify: `backend/internal/services/worker.go`
- Modify: `frontend/src/components/ProjectWorkspace.tsx`

- [x] Add API tests for `generate-all` and asset video task creation.
- [x] Add service methods for enqueueing asset video tasks and applying video variants.
- [x] Expose routes for batch asset generation, asset video generation, and asset video deletion.
- [x] Add per-asset-unit video generation controls.

### Task 3: Storyboard Parity Helpers

**Files:**
- Modify: `backend/internal/services/project_service.go`
- Modify: `backend/internal/httpapi/router.go`
- Modify: `backend/internal/services/media.go`
- Modify: `frontend/src/components/ProjectWorkspace.tsx`

- [x] Add frame lock route and UI button.
- [x] Add frame rendered-image upload route and UI control.
- [x] Add video last-frame extraction route.
- [x] Keep existing add/update/delete/copy/reorder behavior.

### Task 4: Prompt And Voice Utilities

**Files:**
- Modify: `backend/internal/httpapi/router.go`
- Modify: `backend/internal/services/project_service.go`
- Modify: `frontend/src/lib/api.ts`

- [x] Add prompt polish endpoints for storyboard, video, and r2v prompts.
- [x] Add voices list, character voice binding, and voice params endpoints.
- [x] Surface configuration and avoid hardcoded provider outputs.
