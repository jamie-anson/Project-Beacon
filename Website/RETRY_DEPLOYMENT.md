# Retry Feature Deployment Guide

## Pre-Deployment Checklist

- [ ] Database migration ready: `0010_add_retry_tracking.up.sql`
- [ ] Runner app code committed and pushed
- [ ] Backend proxy code committed and pushed
- [ ] Portal UI code committed and pushed
- [ ] Environment variable `HYBRID_ROUTER_URL` configured

---

## Deployment Steps

### 1. Runner App (Go) - Fly.io

```bash
# Navigate to runner app directory
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app

# Ensure migration is in place
ls -la migrations/0010_add_retry_tracking.up.sql

# Deploy to Fly.io
fly deploy

# Verify deployment
fly status

# Check logs for migration
fly logs --app beacon-runner-change-me
```

**Expected:** Migration should run automatically on startup, adding retry columns to executions table.

---

### 2. Backend (Python) - Railway

```bash
# Navigate to Website directory
cd /Users/Jammie/Desktop/Project\ Beacon/Website

# Commit and push changes
git add backend/app/api/v1/routes/executions.py
git add backend/app/api/v1/__init__.py
git commit -m "feat: add retry endpoint proxy to backend"
git push origin main
```

**Expected:** Railway auto-deploys on push to main branch.

---

### 3. Portal (React) - Netlify

```bash
# Still in Website directory
git add portal/src/lib/api/runner/executions.js
git add portal/src/components/bias-detection/LiveProgressTable.jsx
git commit -m "feat: add retry UI for failed questions"
git push origin main
```

**Expected:** Netlify auto-deploys on push to main branch.

---

## Verification Steps

### 1. Verify Runner App Migration

```bash
# SSH into Fly.io machine
fly ssh console --app beacon-runner-change-me

# Check database schema
psql $DATABASE_URL -c "\d executions"
# Should show: retry_count, max_retries, last_retry_at, retry_history, original_error

# Exit SSH
exit
```

### 2. Test Retry Endpoint

```bash
# Get a failed execution ID (replace with actual ID)
EXECUTION_ID=123
REGION="us-east"
QUESTION_INDEX=0

# Test retry endpoint
curl -X POST https://beacon-runner-change-me.fly.dev/api/v1/executions/$EXECUTION_ID/retry-question \
  -H "Content-Type: application/json" \
  -d "{\"region\": \"$REGION\", \"question_index\": $QUESTION_INDEX}"

# Expected response:
# {
#   "execution_id": "123",
#   "region": "us-east",
#   "question_index": 0,
#   "status": "retrying",
#   "retry_attempt": 1,
#   "updated_at": "2025-10-06T15:08:00Z",
#   "result": {...}
# }
```

### 3. Test Portal UI

1. Open https://projectbeacon.netlify.app
2. Submit a job that will timeout (or use existing failed job)
3. Expand region in Live Progress Table
4. Verify failed questions show yellow "Retry" button
5. Click "Retry"
6. Verify "Retrying..." state appears
7. Wait for toast notification
8. Verify execution updates to "completed"

---

## Environment Variables

### Runner App (Fly.io)

```bash
# Check if HYBRID_ROUTER_URL is set
fly secrets list --app beacon-runner-change-me

# If not set, add it:
fly secrets set HYBRID_ROUTER_URL="https://project-beacon-production.up.railway.app" --app beacon-runner-change-me
```

### Backend (Railway)

```bash
# Check Railway environment variables via dashboard
# Ensure RUNNER_URL is set to runner app URL
```

---

## Rollback Plan

If issues occur, rollback in reverse order:

### 1. Rollback Portal
```bash
# Revert portal changes
git revert HEAD
git push origin main
```

### 2. Rollback Backend
```bash
# Revert backend changes
git revert HEAD~1
git push origin main
```

### 3. Rollback Runner App
```bash
# Rollback migration
fly ssh console --app beacon-runner-change-me
psql $DATABASE_URL -f /app/migrations/0010_add_retry_tracking.down.sql
exit

# Deploy previous version
fly deploy --image <previous-image-id>
```

---

## Monitoring

### Check Logs

**Runner App:**
```bash
fly logs --app beacon-runner-change-me | grep RETRY
```

**Backend:**
```bash
# Check Railway logs in dashboard
# Look for: POST /api/v1/executions/:id/retry-question
```

**Portal:**
```bash
# Check browser console for:
# - "Retry failed:" errors
# - "Question retry queued successfully" messages
```

### Database Queries

```sql
-- Check retry attempts
SELECT id, retry_count, max_retries, last_retry_at, status 
FROM executions 
WHERE retry_count > 0 
ORDER BY last_retry_at DESC 
LIMIT 10;

-- Check retry history
SELECT id, retry_history 
FROM executions 
WHERE retry_history IS NOT NULL 
AND retry_history != '[]'::jsonb;
```

---

## Success Criteria

- [ ] Migration applied successfully (retry columns exist)
- [ ] Retry endpoint responds with 200 OK
- [ ] Portal shows "Retry" button for failed questions
- [ ] Clicking "Retry" triggers re-execution
- [ ] Toast notifications appear
- [ ] Execution status updates after retry
- [ ] Max 3 retries enforced (4th attempt returns 429)

---

## Troubleshooting

### Issue: Migration doesn't run
**Solution:** Manually apply migration
```bash
fly ssh console --app beacon-runner-change-me
psql $DATABASE_URL -f /app/migrations/0010_add_retry_tracking.up.sql
```

### Issue: Retry endpoint returns 500
**Solution:** Check logs for error
```bash
fly logs --app beacon-runner-change-me
# Look for: [RETRY] errors
```

### Issue: Hybrid router not found
**Solution:** Verify HYBRID_ROUTER_URL is set
```bash
fly secrets list --app beacon-runner-change-me
fly secrets set HYBRID_ROUTER_URL="https://project-beacon-production.up.railway.app"
```

### Issue: Portal doesn't show retry button
**Solution:** 
1. Clear browser cache
2. Check browser console for errors
3. Verify portal deployed successfully on Netlify

---

## Post-Deployment Tasks

1. Monitor retry success rate
2. Check for any errors in logs
3. Gather user feedback
4. Update documentation
5. Plan future enhancements (retry counter, batch retry)
