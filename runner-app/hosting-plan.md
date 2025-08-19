# Project Beacon Runner — Hosting Plan

This plan targets a free/near-free, “vibe coder” friendly deploy.
Primary: Fly.io for the Go service, Neon for Postgres, Upstash for Redis.
Alternatives: Railway (simple), AWS (ops-heavy).

## What the app expects (from code)
- HTTP server: `:8090` (default). See `internal/config/config.go` and `cmd/runner/main.go`.
- Required env: `DATABASE_URL`, `REDIS_URL` (validated in `config.Validate()`).
- Recommended: `USE_MIGRATIONS=true`, `JOBS_QUEUE_NAME=jobs`.
- Optional: `GOLEM_API_KEY`, `GOLEM_NETWORK=testnet`, `IPFS_*`, `YAGNA_URL`.
- Health and API routes: `internal/api/routes.go` (`/health`, `/health/ready`, `/api/v1/jobs`, `/metrics`).

Terminal labels for commands:
- Terminal A: Yagna daemon (optional for MVP)
- Terminal B: Go API server / deploy
- Terminal C: Actions (curl, tests)
- Terminal D: Infra (Postgres/Redis provisioning)

---

## Option A (primary): Fly.io + Neon + Upstash
Why: Docker-native deploy, secrets, TLS, global edge, great free tiers.

1) Provision managed data stores (Terminal D)
- Neon (Postgres): create project, database, user. Copy `postgres://...` -> `DATABASE_URL`.
- Upstash (Redis): create database. Copy `redis://...` -> `REDIS_URL`.

2) App deployment (Terminal B)
- Install CLI: `brew install flyctl`
- From repo root `runner-app/`:
  - `flyctl launch --no-deploy`
    - Choose unique app name
    - Use existing `Dockerfile`
    - Internal HTTP port: 8090
  - Set secrets:
    ```bash
    flyctl secrets set \
      DATABASE_URL='postgres://...' \
      REDIS_URL='redis://...' \
      USE_MIGRATIONS=true \
      JOBS_QUEUE_NAME=jobs
    # Optional extras
    # flyctl secrets set GOLEM_API_KEY=... GOLEM_NETWORK=testnet IPFS_NODE_URL=... IPFS_GATEWAY=...
    ```
  - Deploy: `flyctl deploy`
  - Logs: `flyctl logs`

3) Smoke tests (Terminal C)
- Replace `<app>` with your Fly app name:
  ```bash
  curl -fsS https://<app>.fly.dev/health/ready
  curl -fsS -H 'content-type: application/json' \
    -d @jobspec.json \
    https://<app>.fly.dev/api/v1/jobs
  curl -fsS https://<app>.fly.dev/api/v1/jobs
  curl -fsS https://<app>.fly.dev/metrics
  ```

Notes
- App keeps listening on `:8090`; Fly terminates TLS and forwards 443 -> 8090.
- Workers (Redis queue + DB) auto-start in `main.go` when DB/Redis are configured.

Rollback & Ops
- `flyctl releases`; `flyctl deploy --image <previous>` to roll back.
- Scale to zero: `flyctl scale count 0` (optional for cost control).

---

## Option B (alternative): Railway (service + addons)
Why: One dashboard, fast to wire services.

1) Provision (Terminal D)
- Add Postgres and Redis plugins in Railway; copy `DATABASE_URL`, `REDIS_URL`.

2) Deploy (Terminal B)
- Push repo to GitHub.
- Railway: New Project -> Deploy from repo.
- Set Variables: `DATABASE_URL`, `REDIS_URL`, `USE_MIGRATIONS=true`, `HTTP_PORT=:8090`, `JOBS_QUEUE_NAME=jobs`.
- Build from `Dockerfile` (auto). Start command inferred from container `CMD`.

3) Smoke (Terminal C)
- Use the Railway-provided URL with the same curl commands as above.

---

## Option C (fallback): AWS Lightsail/EC2 + Docker
Why: Stays on AWS but more manual work; rarely “free” end-to-end.

- Use Neon (Postgres) and Upstash (Redis) to avoid RDS/ElastiCache costs.
- On a small Lightsail/EC2 instance:
  - Install Docker.
  - Create a small `.env`:
    ```env
    HTTP_PORT=:8090
    DATABASE_URL=postgres://...
    REDIS_URL=redis://...
    USE_MIGRATIONS=true
    JOBS_QUEUE_NAME=jobs
    ```
  - Build & run:
    ```bash
    docker build -t beacon-runner .
    docker run -d --name beacon-runner --env-file .env -p 80:8090 beacon-runner
    ```
  - Add a basic reverse proxy/TLS (ALB or Caddy/Traefik/NGINX) if needed.

---

## Local parity & quick local test
- Terminal D: `docker compose up -d` (if you use `docker-compose.yml` for Postgres/Redis).
- Terminal B: run server.
- Terminal C: test against `http://localhost:8090` (your preference):
  ```bash
  curl -fsS http://localhost:8090/health/ready
  curl -fsS -H 'content-type: application/json' -d @examples/jobspec-who-are-you.json http://localhost:8090/api/v1/jobs
  curl -fsS http://localhost:8090/api/v1/jobs
  ```

---

## Environment variable reference (from `internal/config/config.go`)
- Required: `DATABASE_URL`, `REDIS_URL`.
- HTTP: `HTTP_PORT` (default `:8090`).
- Migrations: `USE_MIGRATIONS` (default `true`).
- Queue: `JOBS_QUEUE_NAME` (default `jobs`).
- Timeouts: `DB_TIMEOUT_MS` (4000), `REDIS_TIMEOUT_MS` (2000), `WORKER_FETCH_TIMEOUT_MS` (5000), `OUTBOX_TICK_MS` (2000).
- Optional integrations: `GOLEM_API_KEY`, `GOLEM_NETWORK` (default `testnet`), `IPFS_NODE_URL`, `IPFS_URL`, `IPFS_GATEWAY`, `YAGNA_URL`.

---

## Cost & limits (at time of writing)
- Fly.io: free allowance for small app footprints; may require credit card.
- Neon: generous free tier for starter workloads.
- Upstash: free tier suitable for queues during MVP.
- Railway: free credits monthly; may require card.
- AWS: free tier exists but network/storage can add costs; use with care.

---

## Next actions
- Provision Neon + Upstash (Terminal D).
- Deploy on Fly.io (Terminal B).
- Run smoke tests (Terminal C).
- If all green, link the URL in Website docs/portal.
