# Railway Migration Plan

## Overview
Migrating Project Beacon's hybrid router service from Fly.io to Railway due to persistent deployment issues. Modal GPU functions and Netlify portal will remain unchanged as they're working reliably.

## Pre-Migration Status
- ✅ Modal GPU functions: Working ($3.20/month usage)
- ✅ Netlify portal: Working 
- ❌ Fly.io hybrid router: Suspended/broken for 3+ days
- ❌ Fly.io golem apps: Multiple suspended apps

## Phase 1: Railway Setup

### 1.1 Repository Configuration
- [x] Create `railway.json` configuration file
- [x] Commit and push railway.json to main branch
- [x] Connect GitHub repository to Railway dashboard
- [x] Verify Railway detects the configuration (Project ID: 26123d99-d3c5-4c96-a251-00bf2bc39348)

### 1.2 Environment Variables Setup
Set these in Railway project settings:
- [x] `MODAL_API_TOKEN` - Token for Modal GPU functions
- [ ] `RUNPOD_API_KEY` - RunPod API key (if used)
- [ ] `GOLEM_PROVIDER_ENDPOINTS` - Leave empty for now (no active Golem providers)
- [x] `PORT` - Set to 8000 for Railway

### 1.3 Initial Deployment
- [ ] Deploy hybrid router to Railway
- [ ] Verify deployment succeeds
- [ ] Check Railway logs for any errors
- [ ] Note the Railway app URL (e.g., `https://project-beacon-production.up.railway.app`)

## Phase 2: Endpoint Updates

### 2.1 Portal API Configuration
Update portal to use Railway URLs instead of Fly.io:
- [x] Add Railway and Fly.io service status monitoring to dashboard
- [ ] Update `portal/src/lib/api.js` - change base URLs
- [ ] Update any hardcoded Fly.io URLs in portal code
- [ ] Update environment variables in Netlify if needed
- [ ] Test portal API connections locally

### 2.2 Test Configuration Updates
Update test files to use new Railway endpoints:
- [ ] Update `flyio-deployment/test_hybrid_router.py`
- [ ] Update `tests/e2e/cors-integration.test.js`
- [ ] Update `tests/e2e/deployment-verification.test.js`
- [ ] Update any other test files with hardcoded endpoints

### 2.3 Documentation Updates
- [ ] Update `flyio-deployment/README.md` with Railway instructions
- [ ] Update `integration-guide.md` with new endpoints
- [ ] Update any API documentation with Railway URLs

## Phase 3: Testing & Validation

### 3.1 Health Check Testing
- [ ] Test Railway health endpoint: `GET /health`
- [ ] Verify response format matches expectations
- [ ] Check response time and reliability

### 3.2 API Functionality Testing
- [ ] Test inference endpoint: `POST /inference`
- [ ] Test provider status endpoint: `GET /providers`
- [ ] Test metrics endpoint: `GET /metrics`
- [ ] Verify all responses match Fly.io behavior

### 3.3 Portal Integration Testing
- [ ] Test portal dashboard loads correctly
- [ ] Test job submission works
- [ ] Test real-time updates work
- [ ] Test all portal pages function correctly

### 3.4 End-to-End Testing
- [ ] Run full Playwright test suite
- [ ] Test cross-region functionality
- [ ] Test failover scenarios
- [ ] Verify monitoring and logging work

## Phase 4: Production Cutover

### 4.1 DNS/URL Updates
- [ ] Update any custom domain configurations
- [ ] Update external service integrations
- [ ] Notify any external users of endpoint changes

### 4.2 Monitoring Setup
- [ ] Verify Railway monitoring works
- [ ] Set up alerts for Railway deployment
- [ ] Monitor Railway costs and usage
- [ ] Confirm all metrics are being collected

### 4.3 Performance Validation
- [ ] Compare Railway vs Fly.io response times
- [ ] Monitor Railway uptime for 24 hours
- [ ] Verify auto-scaling works correctly
- [ ] Check memory and CPU usage

## Phase 5: Cleanup

### 5.1 Fly.io Cleanup
- [ ] Stop all Fly.io apps
- [ ] Delete Fly.io apps to avoid charges
- [ ] Remove Fly.io secrets and configurations
- [ ] Archive Fly.io deployment files (don't delete yet)

### 5.2 Documentation Cleanup
- [ ] Update main README with Railway instructions
- [ ] Update deployment documentation
- [ ] Create troubleshooting guide for Railway
- [ ] Document rollback procedure if needed

## Rollback Plan (If Needed)

### Emergency Rollback Steps
- [ ] Redeploy to Fly.io using existing configuration
- [ ] Revert portal API endpoints to Fly.io URLs
- [ ] Revert test configurations
- [ ] Update documentation back to Fly.io

## Success Criteria
- ✅ Railway deployment is stable for 48+ hours
- ✅ All portal functionality works correctly
- ✅ API response times are comparable or better than Fly.io
- ✅ No increase in error rates
- ✅ Cost is equal or lower than Fly.io

## Key URLs to Update

### Current Fly.io URLs (to be replaced):
- `https://beacon-hybrid-router.fly.dev`
- `https://beacon-golem-us.fly.dev`
- `https://beacon-golem-eu.fly.dev`
- `https://beacon-golem-apac.fly.dev`

### New Railway URLs (TBD after deployment):
- `https://project-beacon-production.up.railway.app`

## Files to Update
- `portal/src/lib/api.js`
- `flyio-deployment/test_hybrid_router.py`
- `tests/e2e/cors-integration.test.js`
- `tests/e2e/deployment-verification.test.js`
- `integration-guide.md`
- `README.md`

## Notes
- Modal GPU functions remain unchanged (working perfectly)
- Netlify portal deployment remains unchanged
- Railway offers $5/month predictable pricing vs Fly.io's complex billing
- Railway has better Go/Python support and fewer deployment issues
