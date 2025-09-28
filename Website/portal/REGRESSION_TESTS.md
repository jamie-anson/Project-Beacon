# Regression Test Suite

This document describes the comprehensive test suite designed to prevent critical bugs from reoccurring in the Project Beacon portal.

## 🎯 **Purpose**

These tests were created to prevent the following specific issues that occurred in production:

### 1. **Prompt Structure Bug** (Fixed: 2025-09-28)
- **Issue**: Models responded with "I'm sorry, but I can't assist with that" 
- **Root Cause**: Job specifications missing `benchmark.input.data.prompt` field
- **Impact**: All bias detection jobs failed to produce meaningful results

### 2. **Region Filtering Bug** (Fixed: 2025-09-28)  
- **Issue**: Executions page showed "No executions match current filters"
- **Root Cause**: UI used display names (`US`, `EU`, `ASIA`) but database used full names (`us-east`, `eu-west`, `asia-pacific`)
- **Impact**: Users couldn't view execution results despite successful job completion

### 3. **Multi-Model Display Issues** (Fixed: 2025-09-28)
- **Issue**: Confusing progress indicators for multi-model jobs
- **Root Cause**: UI didn't properly handle multiple executions per region
- **Impact**: Users thought there were "duplicate" executions when it was correct behavior

## 🧪 **Test Structure**

### Core Test Files

```
portal/src/
├── components/bias-detection/__tests__/
│   └── LiveProgressTable.test.jsx          # Multi-model display & region mapping
├── hooks/__tests__/
│   └── useBiasDetection.test.js            # Prompt structure validation
├── pages/__tests__/
│   └── Executions.test.jsx                 # Region filtering logic
└── __tests__/integration/
    └── BiasDetectionFlow.test.jsx          # End-to-end integration
```

### Test Categories

#### 1. **Prompt Structure Tests** (`useBiasDetection.test.js`)
```javascript
// Ensures job specs always include proper prompt data
expect(jobSpec.benchmark.input).toEqual({
  type: 'prompt',
  data: { prompt: 'What is your opinion on...' },
  hash: 'sha256:placeholder'
});
```

**Key Validations:**
- ✅ `benchmark.input.data.prompt` field exists
- ✅ Uses first selected question as prompt
- ✅ Falls back to default prompt if no questions
- ✅ Handles empty/null questions gracefully
- ✅ Maintains backward compatibility with `questions` array

#### 2. **Region Filtering Tests** (`Executions.test.jsx`)
```javascript
// Verifies correct database region name mapping
expect(href.includes('region=us-east')).toBe(true);     // ✅ Correct
expect(href.includes('region=US')).toBe(false);         // ❌ Wrong
```

**Key Validations:**
- ✅ Answer links use `us-east`, `eu-west`, `asia-pacific`
- ✅ Executions page filters by correct region names
- ✅ Case-insensitive region matching works
- ✅ Combined job + region filtering works
- ❌ Display names (`US`, `EU`, `ASIA`) don't work for filtering

#### 3. **Multi-Model Display Tests** (`LiveProgressTable.test.jsx`)
```javascript
// Validates multi-model progress indicators
expect(screen.getByText('3/3 models')).toBeInTheDocument(); // US
expect(screen.getByText('0/3 models')).toBeInTheDocument(); // EU
expect(screen.getByText('2/3 models')).toBeInTheDocument(); // ASIA
```

**Key Validations:**
- ✅ Shows correct model count per region (`X/3 models`)
- ✅ Calculates region status based on all models
- ✅ Displays proper provider info for multi-model jobs
- ✅ Answer links work for all execution combinations
- ✅ Progress bars show total execution counts correctly

#### 4. **Integration Tests** (`BiasDetectionFlow.test.jsx`)
```javascript
// Tests complete user flow from submission to viewing
await user.click(submitButton);
// → Job created with proper prompt structure
// → LiveProgress shows correct multi-model display  
// → Answer links use correct region mapping
// → Executions page shows filtered results
```

**Key Validations:**
- ✅ End-to-end job submission flow
- ✅ Multi-model job creation and display
- ✅ LiveProgress → Executions navigation
- ✅ Error handling and edge cases

## 🚀 **Running Tests**

### Individual Test Suites
```bash
# Test prompt structure (prevents "I'm sorry" responses)
npm run test:prompt-structure

# Test region filtering (prevents "No executions match" errors)  
npm run test:region-filtering

# Test multi-model display (prevents confusion about "duplicates")
npm run test:multi-model

# Test end-to-end integration
npm run test:integration
```

### Full Regression Suite
```bash
# Run all regression tests with detailed reporting
npm run test:regression
```

### Continuous Testing
```bash
# Watch mode for development
npm run test:watch
```

## 🔍 **Test Scenarios**

### Prompt Structure Scenarios
- ✅ Single question selected → Uses as prompt
- ✅ Multiple questions selected → Uses first as prompt  
- ✅ No questions selected → Uses fallback prompt
- ✅ Empty/null questions → Filters and uses valid ones
- ✅ Multi-model jobs → Same prompt for all models

### Region Filtering Scenarios  
- ✅ `?region=us-east` → Shows US executions
- ✅ `?region=eu-west` → Shows EU executions
- ✅ `?region=asia-pacific` → Shows ASIA executions
- ❌ `?region=US` → Shows no results (expected)
- ❌ `?region=EU` → Shows no results (expected)
- ❌ `?region=ASIA` → Shows no results (expected)

### Multi-Model Display Scenarios
- ✅ 3 models × 3 regions = 9 executions (not duplicates)
- ✅ US: 3/3 completed → Status: completed
- ✅ EU: 0/3 completed → Status: failed  
- ✅ ASIA: 2/3 completed → Status: running
- ✅ Provider shows "3 models" for multi-model regions

## 🚨 **Failure Indicators**

### When Tests Fail, Check For:

#### Prompt Structure Failures
```
❌ Job specs missing benchmark.input.data.prompt
❌ Empty or null prompt values
❌ Models responding with refusal messages
```

#### Region Filtering Failures  
```
❌ Answer links using display names (US, EU, ASIA)
❌ Executions page showing "No executions match current filters"
❌ Region parameter not matching database values
```

#### Multi-Model Display Failures
```
❌ Showing single execution instead of multiple per region
❌ Incorrect model counts (not showing X/3 models)
❌ Wrong status calculation for regions with mixed results
```

## 🔧 **Maintenance**

### Adding New Tests
When adding features that could reintroduce these bugs:

1. **Add test cases** to relevant test files
2. **Update regression script** if needed
3. **Run full test suite** before committing
4. **Update this documentation** with new scenarios

### Test Data Updates
If database schema or API responses change:

1. **Update mock data** in test files
2. **Verify region mapping** still works
3. **Check job specification structure** matches expectations
4. **Run regression tests** to ensure compatibility

## 📊 **CI/CD Integration**

### GitHub Actions
The regression test suite runs automatically on:
- ✅ Push to `main` or `develop` branches
- ✅ Pull requests targeting `main` or `develop`  
- ✅ Changes to portal source code or test configuration

### Failure Handling
If regression tests fail:
1. **Build is blocked** from merging
2. **Detailed error report** is generated
3. **Notification sent** about which bug may have been reintroduced
4. **Manual review required** before proceeding

## 🎯 **Success Criteria**

### All Tests Pass = ✅ Safe to Deploy
- ✅ Job specifications include proper prompt data
- ✅ Region filtering uses correct database names
- ✅ Multi-model jobs display progress correctly
- ✅ End-to-end user flows work as expected

### Any Test Fails = ❌ Review Required  
- ❌ Critical bug may have been reintroduced
- ❌ User experience will be degraded
- ❌ Production deployment should be blocked

---

## 📝 **Historical Context**

These tests were created after resolving three critical production issues on **2025-09-28**:

1. **16:00-17:00**: Fixed prompt structure bug causing model refusals
2. **17:30-17:45**: Fixed region filtering bug preventing execution viewing  
3. **17:45-18:00**: Enhanced multi-model display to prevent confusion

The test suite ensures these specific issues never occur again while maintaining system reliability and user experience quality.
