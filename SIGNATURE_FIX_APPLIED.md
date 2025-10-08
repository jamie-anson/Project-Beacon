# Signature Fix Applied - Ready for Testing

**Date:** 2025-10-08  
**Issue:** Cross-region job submissions failing with "Invalid JobSpec signature"  
**Status:** ✅ **FIX APPLIED**

---

## What Was Fixed

### Root Cause
The portal was signing JobSpecs **with** the `id` field, but the server was verifying signatures **without** the `id` field. This mismatch caused all signature verifications to fail.

### The Fix
Modified `portal/src/lib/crypto.js` to exclude the `id` field during signature generation:

```javascript
export function createSignableJobSpec(jobSpec) {
  const signable = { ...jobSpec };
  delete signable.signature;
  delete signable.public_key;
  delete signable.id;  // ← ADDED THIS LINE
  return signable;
}
```

### Why This Works
- Server's `CreateSignableJobSpec()` already removes `id` field (line 217 in `ed25519.go`)
- Now both portal and server create identical canonical JSON for verification
- Signatures will match because both sides sign/verify the same data

---

## Testing Checklist

### Pre-Deployment Testing

- [ ] **Build portal locally**
  ```bash
  cd Website/portal
  npm run build
  ```

- [ ] **Verify fix in build output**
  ```bash
  # Check that CORS and crypto changes survive minification
  grep -i "delete.*id" dist/assets/*.js
  ```

- [ ] **Run portal tests**
  ```bash
  npm test
  ```

### Post-Deployment Testing

- [ ] **Deploy to Netlify**
  ```bash
  git add Website/portal/src/lib/crypto.js
  git commit -m "fix: exclude id field from JobSpec signature to match server verification"
  git push origin main
  ```

- [ ] **Wait for Netlify deploy** (~2 minutes)

- [ ] **Test cross-region job submission**
  1. Go to https://projectbeacon.netlify.app/bias-detection
  2. Connect wallet
  3. Select multiple regions (e.g., US + EU)
  4. Select at least one question
  5. Click "Submit Job"

- [ ] **Expected Result: SUCCESS**
  - Job submits without 400 error
  - Job ID appears in Live Progress section
  - Job status shows "created" or "running"
  - No "Invalid JobSpec signature" error

- [ ] **Verify in browser console**
  ```javascript
  // Should see signature debug logs (if enabled)
  // [SIGNATURE DEBUG] Portal canonical JSON: {...}
  // Verify 'id' is NOT present in the canonical JSON
  ```

- [ ] **Check server logs**
  ```bash
  fly logs -a beacon-runner-production | grep -i signature
  # Should see successful signature verification
  ```

---

## Success Criteria

✅ **Pass**: Job submission completes with 202 Accepted  
✅ **Pass**: Job appears in database with status "created"  
✅ **Pass**: No signature verification errors in logs  
✅ **Pass**: Cross-region analysis data gets created  

❌ **Fail**: Still getting 400 Bad Request with signature error  
❌ **Fail**: Job rejected despite fix  

---

## Rollback Plan

If the fix doesn't work:

1. **Immediate rollback** (if needed):
   ```bash
   git revert HEAD
   git push origin main
   ```

2. **Enable diagnostic logging**:
   - Add console.log statements per Phase 1 of SIGNATURE_FIX_PLAN_V2.md
   - Submit test job and compare canonical JSON outputs

3. **Alternative approach**:
   - Remove ID exclusion from server's `CreateSignableJobSpec()`
   - Let both sides sign WITH the id field

---

## Next Steps After Success

1. **Add regression test**:
   ```javascript
   // portal/src/lib/__tests__/crypto.test.js
   test('createSignableJobSpec excludes id, signature, and public_key', () => {
     const spec = {
       id: 'test-123',
       version: 'v1',
       signature: 'xyz',
       public_key: 'abc',
       benchmark: { name: 'test' }
     };
     
     const signable = createSignableJobSpec(spec);
     
     expect(signable.id).toBeUndefined();
     expect(signable.signature).toBeUndefined();
     expect(signable.public_key).toBeUndefined();
     expect(signable.version).toBe('v1');
     expect(signable.benchmark).toEqual({ name: 'test' });
   });
   ```

2. **Add integration test**:
   - Test full job submission flow with signature verification
   - Ensure both single-region and cross-region jobs work

3. **Update documentation**:
   - Document signature canonicalization process
   - Add note about ID field exclusion

4. **Monitor production**:
   - Watch for any signature-related errors in first 24 hours
   - Check success rate of job submissions

---

## Files Changed

### Modified:
- ✅ `Website/portal/src/lib/crypto.js` (line 48: added `delete signable.id`)

### Documentation:
- ✅ `SIGNATURE_FIX_PLAN_V2.md` (comprehensive debugging plan)
- ✅ `SIGNATURE_FIX_APPLIED.md` (this file)

---

## Technical Details

### Signature Flow (BEFORE fix):
```
Portal: sign({id, version, benchmark, ...}) → signature_A
Server: verify({version, benchmark, ...}) → expects signature_B
Result: signature_A ≠ signature_B → VERIFICATION FAILS ❌
```

### Signature Flow (AFTER fix):
```
Portal: sign({version, benchmark, ...}) → signature_A
Server: verify({version, benchmark, ...}) → expects signature_A
Result: signature_A === signature_A → VERIFICATION PASSES ✅
```

### Why ID Is Excluded:
1. ID is often auto-generated by server
2. Portal generates temporary ID for tracking
3. Server may reassign/validate ID during processing
4. Signature should verify job CONTENT, not identifier

---

## Confidence Level

**🟢 HIGH CONFIDENCE (95%)**

**Reasoning:**
1. ✅ Exact root cause identified by code inspection
2. ✅ Fix aligns with server's existing implementation
3. ✅ Simple one-line change with clear intent
4. ✅ Matches pattern from previous signature fixes
5. ✅ No breaking changes to other functionality

**Remaining 5% uncertainty:**
- Possible secondary issues with other fields (unlikely)
- Deployment/caching issues (can verify with cache clear)
- Unexpected field transformations (JSON serialization differences)

---

## Contact

If issues persist after this fix:
1. Check browser console for new error messages
2. Review server logs for verification failures
3. Compare canonical JSON outputs (see Phase 1 of SIGNATURE_FIX_PLAN_V2.md)
4. Open issue with diagnostic logs attached

---

## Related Documentation

- **Detailed Plan:** `SIGNATURE_FIX_PLAN_V2.md`
- **Previous Fix:** Memory[caa543ac-c7f3-49ac-b8b9-4ece5d667da8]
- **Server Code:** `runner-app/pkg/crypto/ed25519.go:217`
- **Portal Code:** `Website/portal/src/lib/crypto.js:44-49`
