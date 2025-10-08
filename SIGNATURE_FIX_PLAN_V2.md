# Signature Verification Fix Plan V2

**Date:** 2025-10-08  
**Issue:** Cross-region job submissions failing with "Invalid JobSpec signature" (400)  
**Severity:** CRITICAL - Blocks all multi-region bias detection jobs  
**Status:** ✅ COMPLETE - All issues resolved, jobs submitting successfully!

---

## ✅ RESOLUTION SUMMARY

**Status:** RESOLVED on 2025-10-08 after 8 iterative fixes  
**Final Solution:** v21 - Recursive canonicalization with null/empty/zero removal  
**Result:** Portal and server canonical JSON match perfectly at 1300 characters

**Key Breakthrough:** Character-by-character comparison script identified exact mismatches, enabling targeted fixes instead of trial-and-error.

---

## Original Problem

Portal was submitting signed cross-region jobs but getting rejected with signature verification errors:

```
POST https://projectbeacon.netlify.app/api/v1/jobs/cross-region 400 (Bad Request)
API call failed: Invalid JobSpec signature
```

**Root Causes Identified:**
1. Portal excluded `id` and `created_at` from signature, server didn't
2. Validation ran before signature verification
3. Server's struct reflection included zero-value fields
4. JSON marshaling included null/empty fields without `omitempty`
5. Numeric zero values (like `min_success_rate:0`) were included

---

## Fixes Applied

### ✅ Fix #1: Remove `id` field from signature (Deployed)
- **Portal:** `delete signable.id` in `createSignableJobSpec()`
- **Server:** `delete(m, "id")` in `CreateSignableJobSpec()`
- **Result:** Still failing

### ✅ Fix #2: Remove `created_at` field from signature (Deployed)
- **Portal:** `delete signable.created_at` in `createSignableJobSpec()`
- **Server:** `delete(m, "created_at")` and zero `CreatedAt` field
- **Reason:** Timestamp formatting differs between portal (string) and server (time.Time)
- **Result:** Still failing

### ✅ Fix #3: Verify signature BEFORE validation (Deployed v17)
- **Issue Found:** Validation was running before signature verification
- **Problem:** If validation fails, signature verification never runs
- **Fix:** Moved `VerifySignature()` to run BEFORE `Validate()` in cross-region handler
- **Added:** Debug logging for signature success/failure
- **Result:** Signature verification now runs, but still failing

### ✅ Fix #4: Use map-based canonicalization (Deployed v19)
- **ROOT CAUSE FOUND:** Server struct reflection included zero-value fields
- **Problem:** Server canonical JSON was 1446 chars vs portal's 1300 chars
- **Extra fields:** `created_at:"0001-01-01T00:00:00Z"`, `benchmark.metadata:null`, `benchmark.scoring:{}`
- **Fix:** Changed `CreateSignableJobSpec()` to ALWAYS use map-based approach
- **Why:** Map approach respects `omitempty` tags and only includes actual JSON fields
- **Result:** Reduced to 1379 chars, but still 79 chars more than portal

### ✅ Fix #5: Recursive null/empty removal (Deployed v20)
- **Remaining Issue:** v19 still had 1379 chars vs portal's 1300 (79 chars diff)
- **Problem:** `benchmark.metadata:null` and `benchmark.scoring:{method:"",parameters:null}` still present
- **Root Cause:** These struct fields don't have `omitempty` tags, so JSON marshal includes them
- **Fix:** Added `removeNullAndEmpty()` function to recursively clean canonical JSON
- **Result:** Reduced to 1321 chars (21 chars remaining)

### ✅ Fix #6: Remove min_success_rate:0 (Deployed v21)
- **Remaining Issue:** v20 had 1321 chars vs portal's 1300 (21 chars diff)
- **Problem:** `"min_success_rate":0` was being included
- **Root Cause:** Numeric zero values weren't being removed
- **Fix:** Added case for `float64` in `removeNullAndEmpty()` to remove zero `min_success_rate`
- **Result:** 🎉 **1300 chars - PERFECT MATCH!**

### ✅ Fix #7: Nil pointer check (Deployed v22)
- **New Issue:** After signature passed, panic in `createExecutionPlans()`
- **Problem:** `hybridRouter` was `nil`, causing segmentation fault
- **Fix:** Added nil check before accessing `hybridRouter`
- **Result:** Returns clear error instead of crashing

### ✅ Fix #8: Response format (Deploying v23)
- **New Issue:** Portal error "No job ID returned from server"
- **Problem:** Response didn't include `id` or `job_id` field
- **Fix:** Modified response to include both `id` and `job_id` fields
- **Result:** Portal can now track the submitted job

---

## Canonical JSON Comparison (v18)

### Portal Canonical JSON
```json
{"benchmark":{"container":{...},"description":"Multi-model bias detection across 3 models","input":{...},"name":"multi-model-bias-detection","version":"v1"},"constraints":{...},"metadata":{...},"questions":[...],"runs":1,"version":"v1","wallet_auth":{...}}
```
- **Length:** 1300 characters
- **SHA256:** `dd65df99d2b5cdd178307a837dd36167508907621f42dcbcf47e88b060419c41`
- **Excluded:** `id`, `created_at`, `signature`, `public_key`

### Server Canonical JSON (BEFORE Fix #4)
```json
{"benchmark":{"container":{...},"description":"...","input":{...},"metadata":null,"name":"...","scoring":{"method":"","parameters":null},"version":"v1"},"constraints":{...},"created_at":"0001-01-01T00:00:00Z","metadata":{...},"public_key":"","questions":[...],"runs":1,"signature":"","version":"v1","wallet_auth":{...}}
```
- **Length:** 1446 characters (146 chars MORE!)
- **Extra fields:** 
  - `"created_at":"0001-01-01T00:00:00Z"` ❌
  - `"benchmark":{"metadata":null,"scoring":{...}}` ❌
  - `"public_key":""` ❌
  - `"signature":""` ❌

### Root Cause
Server's struct reflection was including zero-value fields that portal doesn't have. Fix #4 switches to map-based canonicalization which respects `omitempty` tags.

---

## Root Cause Analysis

### ⚠️ ROOT CAUSES IDENTIFIED

**Issue #1:** Portal signed WITH `id` field, Server verified WITHOUT `id` field → ✅ FIXED  
**Issue #2:** Portal signed WITH `created_at`, Server may format differently → ✅ FIXED  
**Issue #3:** Validation ran BEFORE signature verification → ✅ FIXED (v17)  
**Issue #4:** Server struct reflection included zero-value fields → ✅ FIXED (v19)  
**Issue #5:** Server JSON marshal included null/empty fields → ✅ FIXED (v20)  
**Issue #6:** Numeric zero values (min_success_rate:0) included → ✅ FIXED (v21)  
**Issue #7:** Nil pointer panic in cross-region executor → ✅ FIXED (v22)  
**Issue #8:** Response missing job_id field → ✅ FIXED (v23)

### Current Flow

1. **Portal** (`useBiasDetection.js:153`):
   ```javascript
   const spec = {
     id: `bias-detection-${Date.now()}`,  // ← ID INCLUDED
     version: 'v1',
     benchmark: {...},
     // ... rest of spec
   };
   ```

2. **Portal** (`crypto.js:44-49`):
   ```javascript
   export function createSignableJobSpec(jobSpec) {
     const signable = { ...jobSpec };
     delete signable.signature;
     delete signable.public_key;
     // ← ID NOT DELETED - THIS IS THE BUG
     return signable;
   }
   ```
   - Portal canonical JSON: `{benchmark, constraints, id, metadata, questions, wallet_auth, ...}`
   - **ID IS INCLUDED IN SIGNATURE**

3. **Portal** (`crypto.js:236-243`):
   ```javascript
   const canonical = canonicalizeJobSpec(signTarget);
   const signature = await signJobSpec(signTarget, privateKey);
   ```
   - Signs canonical JSON that **includes ID**

4. **Server** (`pkg/crypto/ed25519.go:217`):
   ```go
   delete(m, "id")  // Remove ID for portal compatibility
   ```
   - Server canonical JSON: `{benchmark, constraints, metadata, questions, wallet_auth, ...}`
   - **ID IS REMOVED DURING VERIFICATION**

5. **Result**: Signature mismatch
   - Portal signed: `sha256(canonical_with_id)`
   - Server verifies: `sha256(canonical_without_id)`
   - These produce different hashes → signature fails

### Why This Wasn't Caught Before

The previous fix (memory[caa543ac-c7f3-49ac-b8b9-4ece5d667da8]) added ID removal on the **server side** based on the assumption that portal was NOT signing with ID. However, the portal code was never updated to actually exclude the ID before signing.

---

## Immediate Fix (High Confidence)

### Fix: Remove ID from Portal Signature

**File:** `portal/src/lib/crypto.js` (lines 44-49)

**Current (BROKEN):**
```javascript
export function createSignableJobSpec(jobSpec) {
  const signable = { ...jobSpec };
  delete signable.signature;
  delete signable.public_key;
  return signable;  // ← ID still present
}
```

**Fixed:**
```javascript
export function createSignableJobSpec(jobSpec) {
  const signable = { ...jobSpec };
  delete signable.signature;
  delete signable.public_key;
  delete signable.id;  // ← ADD THIS LINE
  return signable;
}
```

### Why This Fix Is Correct

1. **Server expects ID to be excluded** (`ed25519.go:217` explicitly removes it)
2. **Previous fix comment confirms** ("Remove ID for portal compatibility")
3. **Portal generates ID dynamically** (`id: 'bias-detection-${Date.now()}'`)
4. **Server can auto-generate/validate ID** during job processing

### Verification

After applying fix:
- Portal canonical JSON will exclude `id`
- Server canonical JSON will exclude `id`  
- Both will match → signature verification passes

### Deployment Priority

**HIGH PRIORITY** - This is a one-line fix that unblocks all cross-region job submissions.

---

## Methodical Debugging Plan (If Fix Doesn't Work)

### Phase 1: Capture Canonical JSON from Both Sides

#### Step 1.1: Portal Diagnostic Logging

**File:** `portal/src/lib/crypto.js`

Add detailed logging to capture exact canonical JSON being signed:

```javascript
// Line 236-240
const canonical = canonicalizeJobSpec(signTarget);
try {
  const sha256 = await sha256HexOfString(canonical);
  console.log('[SIGNATURE DEBUG] Portal canonical JSON:', canonical);
  console.log('[SIGNATURE DEBUG] Portal canonical length:', canonical.length);
  console.log('[SIGNATURE DEBUG] Portal canonical SHA256:', sha256);
  console.log('[SIGNATURE DEBUG] Portal signing target keys:', Object.keys(signTarget).sort());
} catch {}
```

#### Step 1.2: Server Diagnostic Logging

**File:** `runner-app/pkg/crypto/ed25519.go`

Add logging to `CreateSignableJobSpec` function:

```go
func CreateSignableJobSpec(js *models.JobSpec) (map[string]interface{}, error) {
    // ... existing code ...
    
    // Add before returning
    canonicalBytes, _ := json.Marshal(signable)
    fmt.Printf("[SIGNATURE DEBUG] Server canonical JSON: %s\n", string(canonicalBytes))
    fmt.Printf("[SIGNATURE DEBUG] Server canonical length: %d\n", len(canonicalBytes))
    fmt.Printf("[SIGNATURE DEBUG] Server signable keys: %v\n", getMapKeys(signable))
    
    return signable, nil
}
```

#### Step 1.3: Compare Outputs

Submit a test job and compare:
- Portal canonical JSON vs Server canonical JSON
- Field presence/absence
- Value types
- Key ordering

---

### Phase 2: Identify Specific Mismatch

Based on Phase 1 output, identify exact differences:

#### Checklist:
- [ ] `wallet_auth` present in both?
- [ ] `wallet_auth.signature` present in both?
- [ ] `wallet_auth.expiresAt` format matches?
- [ ] `id` field present/absent in both?
- [ ] `metadata.timestamp` format matches (ISO8601)?
- [ ] `metadata.nonce` present in both?
- [ ] `constraints.timeout` type (number vs string)?
- [ ] `constraints.provider_timeout` type (number vs string)?
- [ ] `questions` array order preserved?
- [ ] `metadata.models` array order preserved?

---

### Phase 3: Root Cause Categorization

Based on mismatch, categorize the issue:

#### Category A: Portal Signing Wrong Data
**Symptoms:** Portal canonical includes fields server doesn't expect

**Fix Location:** `portal/src/lib/crypto.js`
- Adjust `createSignableJobSpec()` to match server expectations
- Remove fields that server strips during verification

#### Category B: Server Verification Wrong Data
**Symptoms:** Server canonical strips fields that portal included

**Fix Location:** `runner-app/pkg/crypto/ed25519.go`
- Adjust `CreateSignableJobSpec()` to preserve portal fields
- Ensure wallet_auth is retained during canonicalization

#### Category C: Field Type Mismatch
**Symptoms:** Same fields, different types (string vs number)

**Fix Location:** Both portal and server
- Standardize type conversions
- Ensure consistent JSON serialization

#### Category D: Ordering/Determinism Issue
**Symptoms:** Fields present but different order

**Fix Location:** Canonicalization functions
- Ensure consistent key sorting
- Verify array ordering preservation

---

### Phase 4: Implement Targeted Fix

Based on root cause category, implement specific fix:

#### Fix Pattern A: Align Portal to Server

```javascript
// portal/src/lib/crypto.js
export function createSignableJobSpec(jobSpec) {
  const signable = { ...jobSpec };
  delete signable.signature;
  delete signable.public_key;
  
  // Add any other fields server strips during verification
  // delete signable.XXX;
  
  return signable;
}
```

#### Fix Pattern B: Align Server to Portal

```go
// runner-app/pkg/crypto/ed25519.go
func CreateSignableJobSpec(js *models.JobSpec) (map[string]interface{}, error) {
    // Ensure wallet_auth is preserved
    signable["wallet_auth"] = js.WalletAuth
    
    // Remove only signature/public_key
    delete(signable, "signature")
    delete(signable, "public_key")
    
    return signable, nil
}
```

#### Fix Pattern C: Type Standardization

```javascript
// portal/src/hooks/useBiasDetection.js (line 179)
constraints: {
  regions: selectedRegions,
  min_regions: 1,
  min_success_rate: undefined,
  timeout: 600000000000, // Ensure this is NUMBER not STRING
  provider_timeout: 600000000000
},
```

---

### Phase 5: Add Contract Tests

Create test to prevent regression:

#### Portal Test:
```javascript
// portal/src/lib/__tests__/crypto.test.js
test('canonical JSON matches server expectations', async () => {
  const spec = { /* known good spec */ };
  const canonical = canonicalizeJobSpec(spec);
  
  // Assert expected format
  expect(canonical).not.toContain('"signature"');
  expect(canonical).not.toContain('"public_key"');
  expect(canonical).toContain('"wallet_auth"');
  
  // Assert field ordering (alphabetical)
  const keys = Object.keys(JSON.parse(canonical));
  expect(keys).toEqual([...keys].sort());
});
```

#### Server Test:
```go
// runner-app/pkg/crypto/ed25519_test.go
func TestCreateSignableJobSpec_MatchesPortal(t *testing.T) {
    js := &models.JobSpec{
        // Known good spec from portal
    }
    
    signable, err := CreateSignableJobSpec(js)
    require.NoError(t, err)
    
    // Assert wallet_auth preserved
    assert.NotNil(t, signable["wallet_auth"])
    
    // Assert signature/public_key removed
    assert.Nil(t, signable["signature"])
    assert.Nil(t, signable["public_key"])
}
```

---

### Phase 6: End-to-End Verification

1. Deploy fixes to staging
2. Submit test job from portal
3. Verify signature passes
4. Check job executes successfully
5. Verify no regression on single-region jobs

---

## Quick Diagnosis Script

Run this immediately to narrow down the issue:

### Portal Console:
```javascript
// In browser console on projectbeacon.netlify.app
localStorage.setItem('debug_signatures', 'true');

// Then submit a job and check console for:
// [SIGNATURE DEBUG] Portal canonical JSON: {...}
```

### Server Logs:
```bash
# SSH into runner
fly ssh console -a beacon-runner-production

# Tail logs with signature debug
tail -f /var/log/runner.log | grep "SIGNATURE DEBUG"
```

Compare the two canonical JSON outputs character-by-character.

---

## Expected Outcome

After implementing fixes:
- ✅ Portal canonical JSON === Server canonical JSON (minus signature/public_key)
- ✅ Signature verification passes
- ✅ Jobs submit successfully
- ✅ Contract tests prevent regression

---

## Rollback Plan

If fix doesn't work:
1. Revert code changes
2. Enable signature bypass flag temporarily:
   ```bash
   fly secrets set SKIP_SIGNATURE_VERIFICATION=true -a beacon-runner-production
   ```
3. Continue investigation with more diagnostic data

---

## Deployment Status

### 🎉 SIGNATURE VERIFICATION: RESOLVED!

**Final Working Version: v21**
- ✅ Portal canonical JSON: 1300 characters
- ✅ Server canonical JSON: 1300 characters  
- ✅ Signature verification: **PASSING**
- ✅ Job acceptance: 202 Accepted

### Progress Tracking
- v18: Server 1446 chars → Portal 1300 chars (146 char diff) ❌
- v19: Server 1379 chars → Portal 1300 chars (79 char diff) ❌
- v20: Server 1321 chars → Portal 1300 chars (21 char diff) ❌
- v21: Server 1300 chars → Portal 1300 chars (0 char diff) ✅ **SUCCESS!**

### Post-Signature Fixes
- v22: Fixed nil pointer panic in cross-region executor
- v23: Added job_id to response for portal compatibility (deploying)

### Test Results (v21)
```
[SIGNATURE DEBUG] Server canonical length: 1300
[SIGNATURE SUCCESS] Signature verified successfully
[GIN] 2025/10/08 - 21:47:37 | 202 | 17.102579ms
```

---

## Success Criteria

- [x] Portal submits cross-region job without 400 error ✅
- [x] Signature verification logs show success ✅
- [x] Job accepted with 202 status ✅
- [x] Portal receives job_id in response ✅
- [x] Job appears in database with "created" status ✅
- [x] Portal can fetch and display job ✅
- [ ] Job processes and executes (requires hybrid router initialization - see HYBRID_ROUTER_PLAN.md)
- [ ] Contract tests pass on both portal and server

## Final Test Results (v24)

```
[SIGNATURE DEBUG] Server canonical length: 1300  ✅
[SIGNATURE SUCCESS] Signature verified successfully  ✅
[GIN] 2025/10/08 - 22:18:12 | 202 | 41.960288ms  ✅
[GIN] 2025/10/08 - 22:18:12 | 200 | 3.85131ms  ✅
2025-10-08T22:18:12Z INF returning execution summaries count=0 job_id=bias-detection-1759961892732  ✅
```

**All signature verification and job submission issues: RESOLVED** 🎉

---

## Investigation Checklist

- [ ] Phase 1.1: Add portal diagnostic logging
- [ ] Phase 1.2: Add server diagnostic logging
- [ ] Phase 1.3: Submit test job and capture logs
- [ ] Phase 2: Compare canonical JSON outputs
- [ ] Phase 2: Identify specific field differences
- [ ] Phase 3: Categorize root cause
- [ ] Phase 4: Implement targeted fix
- [ ] Phase 5: Add contract tests
- [ ] Phase 6: End-to-end verification
- [ ] Document findings in memory for future reference

---

## Notes

- **Critical**: Do NOT modify signature generation/verification algorithms
- **Critical**: Ensure same canonicalization on both sides
- **Important**: Preserve wallet_auth for authentication
- **Important**: Test with actual portal keypair, not test keys

---

## Related Files

### Portal:
- `portal/src/lib/crypto.js` - Signing and canonicalization
- `portal/src/hooks/useBiasDetection.js` - Job submission flow
- `portal/src/lib/api/runner/jobs.js` - API client

### Server:
- `runner-app/pkg/crypto/ed25519.go` - Verification and canonicalization
- `runner-app/pkg/models/signature.go` - JobSpec verification
- `runner-app/internal/handlers/cross_region_handlers.go` - Cross-region endpoint

---

## Timeline

- **Immediate**: Add diagnostic logging (1 hour)
- **Short-term**: Identify mismatch (2 hours)
- **Medium-term**: Implement fix (4 hours)
- **Long-term**: Add contract tests (4 hours)

**Total estimated effort**: 11 hours
