# Project Beacon – Railway Deployment Debrief

Date: 2025-09-22
Service: Project-Beacon (Hybrid Router)
Region: europe-west4
Public URL: https://project-beacon-production.up.railway.app

## Summary
- Goal: Get the Hybrid Router to pass Railway health checks and serve endpoints reliably.
- Outcome: Successful deployment. Readiness and health endpoints working; CI and local smoke tests added to prevent regressions.

## Timeline (key points)
- Added unit tests and local Docker smoke test to validate `/health` and startup.
- Identified uvicorn CLI issue with `--port ${PORT:-8000}` (string literal not parsed to int) causing crashes.
- Switched to Python entrypoint `python3 hybrid_router_new.py` which reads `PORT` via `get_port()`.
- Added lightweight readiness endpoint `/ready` and set Railway healthcheck to use it.
- Fixed import path issue in `hybrid_router/api/maps.py` and made the maps router optional at import to avoid startup failures if that module breaks.
- Cleaned up `.railwayignore` and standardized on `Website/Dockerfile` via `Website/railway.json`.
- Redeployed; readiness returned 200 and health stabilised.

## Root Causes
- Uvicorn CLI does not expand shell-style defaults; `--port ${PORT:-8000}` passed a non-integer string, causing uvicorn to exit.
- `hybrid_router/api/maps.py` imported `get_google_maps_api_key` indirectly via a non-existent `hybrid_router.api.config`. This caused module import crashes at app startup.
- Readiness and health conflated. Health occasionally slow due to provider background checks.

## Fixes Implemented
- Start command: `Website/railway.json`
  - from: `python3 -m uvicorn hybrid_router.main:app --host 0.0.0.0 --port ${PORT:-8000} ...`
  - to: `python3 hybrid_router_new.py` (PORT handled in code)
- Readiness endpoint: `Website/hybrid_router/api/health.py`
  - Added `GET /ready` (fast 200) for platform probes.
  - Kept `GET /health` for service telemetry (non-blocking provider checks).
- Optional maps router:
  - `Website/hybrid_router/api/__init__.py`: wrap maps import in try/except.
  - `Website/hybrid_router/main.py`: include `maps_router` only if available.
  - `Website/hybrid_router/api/maps.py`: import fixed to `from ..config import get_google_maps_api_key`.
- Docker/Healthcheck:
  - `Website/Dockerfile` HEALTHCHECK now probes `/ready`.
- Config-as-code for Railway: `Website/railway.json`
  - `build.builder = DOCKERFILE`
  - `build.dockerfilePath = Website/Dockerfile`
  - `deploy.healthcheckPath = /ready`
  - `deploy.restartPolicyType = ON_FAILURE`
- `.railwayignore` tuned to whitelist only required app files.

## Tests & CI
- Unit tests (FastAPI in-process): `Website/tests/`
  - `test_app_boot.py`: app imports; `/health` returns 200.
  - `test_health_schema.py`: validates `/health` JSON shape and latency.
- Local smoke test: `Website/Makefile`
  - `make test` and `make smoke` (builds `Website/Dockerfile`, runs uvicorn in container, polls `/health`).
- CI: `.github/workflows/router-ci.yml`
  - Runs unit tests, builds the Docker image, runs container with uvicorn, polls `/health`, and smokes `/providers`.

## Lessons Learned
- Do not pass shell-style defaults to uvicorn CLI; resolve env vars in code.
- Separate readiness (`/ready`) from health (`/health`) for reliable platform checks.
- Make optional/ancillary routers non-fatal to app startup.
- Pin Railway build context and Dockerfile path via `railway.json` in monorepos.
- Gate deployments with unit + container smoke tests.

## Outstanding/Related Services
- `fabulous-renewal` (Backend Diffs): Failing due to missing `Dockerfile.railway`. If not needed, disable auto-deploy or delete service. If needed, reconfigure build to `Website/` + `Dockerfile`.
- `backend-diffs` (legacy): Consider disabling/deleting to reduce noise and cost.

## Next Steps
- Add an automated check that `Website/railway.json` startCommand never contains `${...}`.
- Add structured startup logs and basic metrics.
- Expand tests to cover websockets (`/ws`) and provider list semantics.
- Monitor Railway metrics; add alerts for readiness/health failures.

## Quick Verification
```bash
curl -fsS https://project-beacon-production.up.railway.app/ready | jq
curl -fsS https://project-beacon-production.up.railway.app/health | jq
curl -fsS https://project-beacon-production.up.railway.app/providers | jq
```
