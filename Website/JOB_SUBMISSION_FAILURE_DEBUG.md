# Job Submission Failure - Debug Guide

**Job ID**: bias-detection-1759264647176  
**Issue**: Job shows as "failed" in portal but doesn't exist in runner database

---

## 🔍 Root Cause

The job **never reached the runner**. It failed during submission from the portal.

**Evidence**:
- Runner API returns "Not Found" for this job ID
- No logs in runner for job creation
- Portal shows "SYSTEM FAILURE" immediately

---

## 🐛 Likely Causes

### 1. Portal Submission Error
The portal failed to submit the job to the runner API.

**Check**:
- Open browser DevTools → Network tab
- Look for failed POST request to `/api/v1/jobs`
- Check the error response

### 2. Signature Verification Failed
The job signature was invalid.

**Check**:
- Look for 400 response with "signature_mismatch"
- Check if wallet is connected properly

### 3. JobSpec Validation Failed
The job spec had invalid data.

**Check**:
- Look for 400 response with validation errors
- Check if questions/models/regions are properly formatted

### 4. Network/CORS Error
Request didn't reach the runner.

**Check**:
- Look for CORS errors in console
- Check if Railway service is up

---

## 🔧 How to Debug

### Step 1: Check Browser Console

Open DevTools and look for:
```
Failed to submit job: [error message]
```

### Step 2: Check Network Tab

Look for the POST request:
```
POST https://project-beacon-production.up.railway.app/api/v1/jobs
```

**If 400**: Check response body for validation error  
**If 500**: Runner error, check runner logs  
**If failed**: Network/CORS issue

### Step 3: Check Portal State

In the portal, the job shows as "failed" which means:
- Portal created a local job record
- Submission to runner failed
- Portal marked it as failed locally

This is **expected behavior** when submission fails.

---

## ✅ How to Fix

### Option 1: Check What You Submitted

**Questions**: Did you provide valid question IDs?  
**Models**: Did you select models?  
**Regions**: Did you select regions?

### Option 2: Try a Simple Job

Submit with minimal data:
- 1 question: `test_question`
- 1 model: `llama3.2-1b`
- 1 region: `US`

### Option 3: Check Wallet Connection

Make sure your wallet is connected before submitting.

---

## 🎯 Next Steps

1. **Open the portal in browser**
2. **Open DevTools (F12)**
3. **Go to Network tab**
4. **Try submitting another job**
5. **Look for the POST request and check the error**

The error message will tell us exactly what went wrong!

---

## 💡 Common Issues

### "questions is required"
You didn't provide any questions. Add at least one question ID.

### "signature_mismatch"
Wallet signature is invalid. Reconnect your wallet.

### "CORS error"
Railway service might be down. Check https://project-beacon-production.up.railway.app/api/v1/health

### "Network error"
Check your internet connection or Railway service status.

---

## 📝 What to Look For

When you submit the next job, capture:
1. The POST request URL
2. The request payload
3. The response status code
4. The response body

This will tell us exactly what's failing!
