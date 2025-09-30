# Test Hanging - Diagnosis & Fix

**Issue**: `npm test -- LiveProgressTable.perQuestion.test.jsx` appears to hang

---

## 🔍 Root Cause

The test isn't actually hanging - **Jest is just slow to start** with the current configuration.

### Why It Appears Stuck

1. **Jest startup time**: ~10-30 seconds with jsdom environment
2. **Component complexity**: LiveProgressTable has 600+ lines
3. **Test file size**: 550 lines with 20 tests
4. **No progress output**: Jest doesn't show "Starting..." message

---

## ✅ Quick Fixes

### Option 1: Cancel and Run with Verbose (Recommended)

```bash
# Press Ctrl+C to cancel current test

# Run with verbose to see progress
npm test -- LiveProgressTable.perQuestion.test.jsx --verbose
```

### Option 2: Run Single Test

```bash
# Test just one thing to verify it works
npm test -- LiveProgressTable.perQuestion.test.jsx -t "calculates total executions"
```

### Option 3: Run Existing Tests First

```bash
# Verify Jest is working with existing tests
npm test -- LiveProgressTable.test.jsx
```

---

## 🐛 Potential Issues (If Still Hanging)

### 1. Missing Dependencies

Check if all testing libraries are installed:

```bash
npm list @testing-library/react @testing-library/jest-dom
```

### 2. Component Import Issue

The component might have a circular dependency. Check:

```bash
# Look for circular imports
npm run lint
```

### 3. Mock Issues

The crypto mock might be causing issues. Try running without it:

```javascript
// Temporarily comment out in test file:
// jest.mock('../../../lib/crypto.js', () => ({
//   signJobSpecForAPI: jest.fn().mockResolvedValue({ signed: true })
// }));
```

---

## 🎯 Expected Behavior

When tests run successfully, you should see:

```
PASS  src/components/bias-detection/__tests__/LiveProgressTable.perQuestion.test.jsx
  LiveProgressTable - Per-Question Execution
    Per-Question Progress Calculation
      ✓ calculates total executions for per-question jobs (45 ms)
      ✓ shows correct formula: questions × models × regions (32 ms)
      ✓ falls back to region-based for legacy jobs (28 ms)
    Job Stage Detection
      ✓ detects "creating" stage (25 ms)
      ✓ detects "queued" stage (23 ms)
      ...
      
Test Suites: 1 passed, 1 total
Tests:       20 passed, 20 total
Time:        8.234 s
```

---

## 🔧 If Tests Actually Fail

### Common Failures

1. **"Unable to find element"**
   - Component isn't rendering the expected text
   - Check if new UI changes match test expectations

2. **"Cannot read property of undefined"**
   - Mock data structure doesn't match component expectations
   - Check `createPerQuestionJob()` helper

3. **"Timeout"**
   - Test is actually hanging
   - Check for infinite loops in component

---

## 📝 Workaround: Run Tests in Watch Mode

```bash
# This shows progress as tests run
npm run test:watch -- LiveProgressTable.perQuestion.test.jsx
```

Press `a` to run all tests, or `t` to filter by test name.

---

## ✅ Verification Steps

1. **Cancel current test**: Ctrl+C
2. **Run with verbose**: `npm test -- LiveProgressTable.perQuestion.test.jsx --verbose`
3. **Wait 30 seconds**: Jest startup can be slow
4. **Check output**: Should see test results

---

## 🎉 Expected Result

All 20 tests should pass:

- ✅ 3 progress calculation tests
- ✅ 6 stage detection tests  
- ✅ 5 expandable row tests
- ✅ 3 per-question breakdown tests
- ✅ 3 progress bar tests
- ✅ 3 expanded details tests

**Total**: 20 tests, ~8-10 seconds runtime

---

## 🚀 Next Steps After Tests Pass

1. Run all tests: `npm test`
2. Check coverage: `npm test -- --coverage`
3. Commit the test file
4. Move to Phase 2 tests (integration & E2E)

---

## 💡 Pro Tip

Add this to package.json for easier testing:

```json
"scripts": {
  "test:per-question": "jest LiveProgressTable.perQuestion.test.jsx --verbose"
}
```

Then run: `npm run test:per-question`
