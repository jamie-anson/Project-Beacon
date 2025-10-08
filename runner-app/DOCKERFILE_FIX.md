# Dockerfile Fix - Wrong Binary Deployed

## Issue
Portal was getting 404 errors for `/api/v1/questions` endpoint because the Dockerfile was building the wrong binary.

## Root Cause
- **Dockerfile line 16** was building: `./cmd/server`
- **Should build:** `./cmd/runner`

## Difference Between Binaries

### `cmd/server/main.go` (Minimal Cross-Region API)
- Limited endpoints for cross-region execution
- Missing portal-required endpoints like `/questions`, `/jobs`, `/executions`
- Only has: `/health`, `/ws`, `/api/v1/jobs/cross-region`, `/api/v2/jobs/:jobId/bias-analysis`

### `cmd/runner/main.go` (Full Runner API)
- Complete API with all portal endpoints
- Includes: `/api/v1/questions`, `/api/v1/jobs`, `/api/v1/executions`, etc.
- Job processing workers (OutboxPublisher, JobRunner)
- Full RBAC, signature verification, transparency layer

## Fix Applied
Changed Dockerfile line 16:
```dockerfile
# Before
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -a -installsuffix cgo -o server ./cmd/server

# After  
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -a -installsuffix cgo -o server ./cmd/runner
```

## Verification
After redeployment:
```bash
# Should return questions data (not 404)
curl https://beacon-runner-production.fly.dev/api/v1/questions

# Should return jobs list
curl https://beacon-runner-production.fly.dev/api/v1/jobs?limit=5
```

## Status
- ✅ Dockerfile fixed
- 🔄 Redeploying to beacon-runner-production
- ⏳ Portal will work once deployment completes
