# Beacon Portal Plan (MVP)

Scope: single predefined benchmark ("Who are you?") with ability to view its JobSpec, execute it, monitor executions, and view diffs.

## Terminals (dev workflow)
- Terminal A: Yagna daemon
- Terminal B: Go API server (Runner) — http://localhost:8090
- Terminal C: Website Portal dev — Vite on http://localhost:5173 (proxied to :8090)
- Terminal D: Postgres + Redis — docker compose

## Milestones
- [ ] Scaffold portal app under `Website/portal/` (Vite + React + Tailwind)
- [ ] Dev proxy to Runner API and WS (http://localhost:8090)
- [ ] Pages: Dashboard, Template Viewer (Who are you?), Executions, Diffs
- [ ] API wiring: health, jobs create/execute, executions, diffs
- [ ] Basic WS indicator (follow-up)
- [ ] Netlify build integration to serve under `/portal`

## API Endpoints used
- GET `/api/v1/health`
- POST `/api/v1/jobs` (create predefined JobSpec)
- POST `/api/v1/jobs/:id/execute`
- GET `/api/v1/executions?limit=N`
- GET `/api/v1/diffs?limit=N`

## Next (post-MVP)
- Full JobSpec Builder (form + JSON editor)
- Client-side signing (WebCrypto) and provenance UI
- Execution details (logs, receipts, artifacts)
- Diff detail with side-by-side highlighter
- Catalog of multiple benchmarks
