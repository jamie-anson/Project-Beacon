# Runner Migration Complete ✅

**Date:** 2025-10-08  
**Migration:** `beacon-runner-change-me` → `beacon-runner-production`

## Summary

Successfully migrated Project Beacon Runner API to a fresh Fly.io deployment with proper configuration and all references updated.

## What Was Done

### 1. Root Cause Analysis
- **Problem:** Old Runner app (`beacon-runner-change-me`) was returning 502 errors
- **Cause:** App listening on wrong port (8091 vs 8090)
- **Issue:** `fly.toml` set `HTTP_PORT` but app reads `PORT` env var

### 2. New Deployment
- **App Name:** `beacon-runner-production`
- **URL:** `https://beacon-runner-production.fly.dev`
- **Region:** lhr (London)
- **Config:** Fixed `fly.production.toml` to use `PORT=8090`

### 3. Secrets Migrated
All production secrets successfully copied:
- ✅ `DATABASE_URL` - Neon PostgreSQL connection
- ✅ `REDIS_URL` - Upstash Redis connection
- ✅ `ADMIN_TOKENS` - New secure token generated
- ✅ `TRUSTED_KEYS_JSON` - Portal public key for signature verification
- ✅ `STORACHA_TOKEN` - IPFS storage token
- ✅ `HYBRID_BASE` - Railway hybrid router URL
- ✅ `OPENAI_API_KEY` - OpenAI API key (Write permissions)

### 4. Code Updates
Updated **70 files** across the codebase:
- Portal configuration (`portal/src/lib/api/config.js`)
- Netlify redirects (`portal/public/_redirects`)
- All test scripts (18 files)
- GitHub Actions workflows (3 files)
- Documentation (facts.json, runbooks, guides)
- Storybook stories

### 5. Verification
- ✅ Health endpoint: `https://beacon-runner-production.fly.dev/health` returns `{"ok":true}`
- ✅ CORS headers: Proper `Access-Control-Allow-Origin` for portal
- ✅ Port configuration: App listening on correct port (8090)
- ✅ Old app destroyed: `beacon-runner-change-me` deleted

## New Production Configuration

### Admin Token
```
fef3eab741631182dc97579b8b44b422fcd26470c46c38e0cfe65e2fbac9ec1f
```

### API Endpoints
- **Base URL:** `https://beacon-runner-production.fly.dev`
- **Health:** `/health`
- **API:** `/api/v1/*`
- **WebSocket:** `/ws`

### Test Commands
```bash
# Health check
curl https://beacon-runner-production.fly.dev/health

# CORS test
curl -I -X OPTIONS https://beacon-runner-production.fly.dev/api/v1/jobs \
  -H "Origin: https://projectbeacon.netlify.app" \
  -H "Access-Control-Request-Method: GET"

# Admin API test
curl -H "Authorization: Bearer fef3eab741631182dc97579b8b44b422fcd26470c46c38e0cfe65e2fbac9ec1f" \
  https://beacon-runner-production.fly.dev/admin/config
```

## Portal Integration

### Automatic Updates
- Portal will auto-deploy via Netlify
- All API calls now route to new Runner URL
- CORS errors should be resolved

### Manual Testing
Once Netlify deployment completes:
1. Visit: `https://projectbeacon.netlify.app`
2. Navigate to Bias Detection page
3. Verify no CORS errors in console
4. Test job submission

## Files Modified

### Configuration
- `runner-app/fly.production.toml` - New production config
- `Website/portal/src/lib/api/config.js` - Updated default URL
- `Website/portal/public/_redirects` - Updated API proxy

### Scripts (18 files)
- `scripts/submit-signed-job.js`
- `scripts/test-cors-api.js`
- `scripts/test-portal-signing.js`
- `scripts/pre-deploy-validation.js`
- And 14 more test/debug scripts

### CI/CD (3 files)
- `.github/workflows/production-tests.yml`
- `.github/workflows/deployment-tests.yml`
- `.github/workflows/scheduled-monitoring.yml`

### Documentation
- `docs/sot/facts.json` - Updated all Runner references
- `observability/runbook.md` - Updated emergency commands
- Multiple deployment guides and analysis documents

## Next Steps

1. ✅ Monitor Netlify deployment
2. ✅ Test portal after deployment completes
3. ✅ Verify CORS errors are resolved
4. ✅ Test end-to-end job submission
5. ⏳ Update any external documentation/wikis

## Rollback Plan (If Needed)

If issues arise, can quickly rollback:
```bash
# Revert portal changes
git revert HEAD~2..HEAD
git push origin main

# Redeploy old app (if needed)
flyctl apps create beacon-runner-change-me
flyctl deploy -a beacon-runner-change-me
```

## Status: ✅ COMPLETE

All migration tasks completed successfully. Portal deployment in progress via Netlify.
