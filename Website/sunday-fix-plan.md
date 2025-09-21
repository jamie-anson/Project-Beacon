# Sunday Fix Plan - Deployment Issues Resolution
**Date:** 2025-09-21 | **Priority:** Critical | **Status:** Active

## 🚨 Current Status Assessment

### ✅ Working Services
- **Netlify Main Site**: ✅ https://projectbeacon.netlify.app (serving correctly)
- **Netlify Portal**: ✅ https://projectbeacon.netlify.app/portal/ (loading)
- **Railway Backend**: ✅ https://project-beacon-production.up.railway.app/health (responding)

### ❌ Failing Services
- **Railway Hybrid Router**: ❌ Deploy failed - "Could not find root directory: /hybrid_router/"
- **Netlify Build**: ❌ Build failed - missing import on line 433
- **GitHub Actions**: ❌ Rollup module error - npm dependency issue with package-lock.json
- **Portal API Integration**: ❌ Failing due to all backend issues

---

## 🔍 Root Cause Analysis

### Issue 1: Railway Directory Structure Error
**Problem:** Railway can't find `/hybrid_router/` directory
- **Error**: "Could not find root directory: /hybrid_router/"
- **Cause**: Dockerfile.railway trying to copy `./hybrid_router` but Railway build context is different
- **Fix Needed**: Adjust COPY paths for Railway's build context

### Issue 2: Netlify Build Import Error
**Problem:** Missing import causing build failure on line 433
- **Error**: Build failed due to missing import
- **Location**: Line 433 in unknown file
- **Fix Needed**: Identify and fix missing import statement

### Issue 3: GitHub Actions npm/Rollup Error
**Problem:** Rollup module dependency issue
- **Error**: "Cannot find module @rollup/rollup-linux-x64-gnu"
- **Cause**: npm bug with optional dependencies + corrupted package-lock.json
- **Fix Needed**: Clean npm cache and regenerate package-lock.json

### Issue 4: CORS Test Failure (Secondary)
**Problem:** Portal form submission failing CORS validation
- **Test**: `should successfully submit job with proper CORS handling`
- **Likely Cause**: Portal trying to reach `/providers` endpoint that doesn't exist

---

## 🛠️ Fix Plan (Priority Order)

### Phase 0: Emergency Fixes (IMMEDIATE - 15 mins)

#### 0.1 Fix GitHub Actions npm Issue
```bash
# Remove corrupted package files
rm -rf node_modules package-lock.json
rm -rf portal/node_modules portal/package-lock.json

# Clean npm cache
npm cache clean --force

# Reinstall dependencies
npm install
cd portal && npm install && cd ..
```

#### 0.2 Fix Railway Dockerfile Context
```dockerfile
# Railway sees the entire repo, so paths should be:
COPY hybrid_router_new.py requirements.txt ./
COPY hybrid_router/ ./hybrid_router/
```

#### 0.3 Find and Fix Netlify Import Error
```bash
# Search for line 433 import issues
grep -n "import" **/*.js **/*.jsx **/*.ts **/*.tsx | grep ":433:"
```

### Phase 1: Fix Railway Hybrid Router Deployment (20 mins)

#### 1.1 Fix Dockerfile.railway Paths
```dockerfile
FROM python:3.9-slim

WORKDIR /app

# Fix: Copy files from correct paths (no Website/ prefix)
COPY hybrid_router_new.py requirements.txt ./
COPY hybrid_router ./hybrid_router

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8000/health || exit 1

# Start the application
CMD ["python3", "hybrid_router_new.py"]
```

#### 1.2 Verify Railway Configuration
- **File**: `railway.json`
- **Check**: Dockerfile path points to `Dockerfile.railway`
- **Verify**: Health check path is `/health`

#### 1.3 Test Hybrid Router Locally
```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website
python3 hybrid_router_new.py
# Test: curl http://localhost:8000/providers
```

### Phase 2: Fix GitHub Actions CORS Test (15 mins)

#### 2.1 Update CORS Test Expectations
**Problem**: Test expects form submission to work, but `/providers` endpoint is missing
**Solution**: Make test more resilient to backend failures

#### 2.2 Alternative: Skip CORS Test Temporarily
Add conditional skip if backend is unreachable:
```javascript
test.skip(process.env.SKIP_BACKEND_TESTS === 'true', 'Backend not available');
```

### Phase 3: Verify End-to-End Integration (15 mins)

#### 3.1 Test Portal → Railway Integration
- Navigate to portal bias detection page
- Verify API calls reach Railway hybrid router
- Check `/providers` endpoint returns provider list

#### 3.2 Test Complete Workflow
- Submit bias detection job through portal
- Verify job reaches Railway backend
- Check execution records are created

---

## 📋 Detailed Action Items

### Action 1: Fix Railway Dockerfile
```bash
# Edit Dockerfile.railway
# Remove "Website/" prefix from COPY commands
# Test build locally before pushing
```

### Action 2: Deploy to Railway
```bash
git add Dockerfile.railway
git commit -m "fix: correct Railway Dockerfile paths for hybrid router deployment"
git push origin main
# Monitor Railway deployment logs
```

### Action 3: Verify Hybrid Router Endpoints
```bash
# Test after Railway deployment completes
curl https://project-beacon-production.up.railway.app/providers
curl https://project-beacon-production.up.railway.app/metrics
curl https://project-beacon-production.up.railway.app/env
```

### Action 4: Fix CORS Test
```bash
# Option A: Make test more resilient
# Option B: Skip test if backend unavailable
# Option C: Mock backend responses for testing
```

### Action 5: End-to-End Validation
```bash
# Test portal functionality
# Verify bias detection workflow
# Check GitHub Actions pass
```

---

## 🎯 Success Criteria

### Immediate (Next 1 Hour)
- [ ] Railway deploys hybrid router successfully
- [ ] `/providers` endpoint returns provider list
- [ ] CORS test passes or is properly skipped
- [ ] GitHub Actions workflow completes successfully

### Complete (End of Day)
- [ ] Portal can submit bias detection jobs
- [ ] Jobs execute across multiple regions
- [ ] All deployment pipelines green
- [ ] No CORS errors in browser console

---

## 🚨 Rollback Plan

### If Railway Deployment Fails
1. Revert `Dockerfile.railway` to previous working version
2. Deploy old backend service temporarily
3. Update portal to handle missing `/providers` gracefully

### If CORS Tests Keep Failing
1. Skip CORS tests temporarily with environment flag
2. Focus on core functionality first
3. Debug CORS issues separately

---

## 📊 Risk Assessment

### High Risk
- **Railway Dockerfile Changes**: Could break entire backend
- **Mitigation**: Test locally first, have rollback ready

### Medium Risk
- **CORS Test Modifications**: Could mask real CORS issues
- **Mitigation**: Implement proper error handling, not just skipping

### Low Risk
- **Portal Integration**: Already partially working
- **Mitigation**: Graceful degradation if backend unavailable

---

## 🔄 Monitoring & Validation

### During Deployment
- [ ] Watch Railway deployment logs
- [ ] Monitor GitHub Actions progress
- [ ] Check Netlify build status

### Post-Deployment
- [ ] Test all endpoints manually
- [ ] Run full test suite
- [ ] Check browser console for errors
- [ ] Verify portal functionality

---

## 📞 Emergency Contacts & Resources

### Documentation
- **Railway Docs**: https://docs.railway.app/
- **FastAPI CORS**: https://fastapi.tiangolo.com/tutorial/cors/
- **Playwright Testing**: https://playwright.dev/

### Debugging Commands
```bash
# Check Railway logs
railway logs

# Test local hybrid router
python3 hybrid_router_new.py

# Run CORS tests only
npm test -- cors-integration

# Check portal API calls
# Open browser dev tools → Network tab
```

---

## ⏰ Timeline

| Time | Task | Duration | Status |
|------|------|----------|--------|
| 15:45 | Fix Dockerfile.railway | 15 min | 🔄 |
| 16:00 | Deploy to Railway | 10 min | ⏳ |
| 16:10 | Test /providers endpoint | 5 min | ⏳ |
| 16:15 | Fix CORS test | 15 min | ⏳ |
| 16:30 | End-to-end validation | 15 min | ⏳ |
| 16:45 | **COMPLETE** | - | ⏳ |

**Total Estimated Time**: 1 hour

---

## 🎯 Next Steps After Fix

1. **Documentation Update**: Document the Railway deployment process
2. **Monitoring Setup**: Add alerts for endpoint failures
3. **Test Improvements**: Make tests more resilient to backend issues
4. **CI/CD Hardening**: Add deployment validation steps

---

## ✅ **PROGRESS UPDATE - 15:52**

### 🎯 **Completed Emergency Fixes:**
- ✅ **GitHub Actions npm/Rollup Error**: Fixed corrupted dependencies, regenerated package-lock.json
- ✅ **Railway Dockerfile Context**: Fixed COPY paths, added proper directory structure
- ✅ **Local Build Validation**: All builds working locally after npm fixes
- ✅ **Dependencies Deployed**: Clean npm install completed, no more rollup errors

### 🔄 **Currently Deploying:**
- 🚀 **Railway**: Hybrid router deployment in progress with fixed Dockerfile
- 🚀 **GitHub Actions**: Should pass with clean dependencies
- 🚀 **Netlify**: Will test after Railway deployment completes

### ⏳ **Next Validation Steps:**
1. **Monitor Railway logs** for hybrid router deployment success
2. **Test `/providers` endpoint** once Railway completes
3. **Debug Netlify import error** if it persists after dependency fixes
4. **Validate portal integration** with working backend

### 📊 **Expected Timeline:**
- **16:00**: Railway deployment should complete
- **16:05**: `/providers` endpoint should be available  
- **16:10**: GitHub Actions should pass
- **16:15**: All systems operational

**Status**: ✅ Emergency fixes deployed, monitoring deployments | **Owner**: Development Team | **Priority**: P0 Critical
