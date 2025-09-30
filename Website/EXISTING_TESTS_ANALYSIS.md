# Existing Tests Analysis - LiveProgressTable

**Date**: 2025-09-30  
**Purpose**: Analyze existing test coverage and identify gaps for per-question execution

---

## ✅ Existing Test Coverage

### Test File: `LiveProgressTable.test.jsx` (354 lines)

#### 1. Region Filtering Links (2 tests)
- ✅ Answer links use correct database region names
- ✅ mapRegionToDatabase function works correctly
- **Coverage**: Region name mapping (US → us-east, EU → eu-west, ASIA → asia-pacific)

#### 2. Multi-Model Execution Display (2 tests)
- ✅ Shows correct model count for multi-model jobs
- ✅ Shows correct status for multi-model regions
- **Coverage**: Multi-model job handling, status aggregation

#### 3. Job Failure Detection (3 tests)
- ✅ Shows failure alert for failed jobs
- ✅ Shows timeout alert for stuck jobs (15+ minutes)
- ✅ Shows all regions as failed when job fails
- **Coverage**: Error states, timeout detection

#### 4. Completed Job Handling (2 tests)
- ✅ Shows all regions as completed without execution records
- ✅ Shows correct provider info for completed jobs
- **Coverage**: Legacy job completion, missing execution records

#### 5. Progress Calculation (1 test)
- ✅ Calculates correct progress for multi-model jobs
- **Coverage**: Basic progress percentage calculation

---

## ❌ Missing Test Coverage (Per-Question Execution)

### 1. Per-Question Progress Calculation
**Status**: ❌ NOT TESTED

**What's Missing**:
- Calculating total executions: regions × models × questions
- Expected total: 2 questions × 3 models × 3 regions = 18
- Detecting if job has questions (hasQuestions flag)
- Falling back to region-based for legacy jobs

**Tests Needed**:
```javascript
test('calculates total executions for per-question jobs', () => {
  // Should show "18 executions" not "3 regions"
});

test('falls back to region-based for legacy jobs', () => {
  // Jobs without question_id should show "3 regions"
});
```

---

### 2. Job Stage Detection
**Status**: ❌ NOT TESTED

**What's Missing**:
- Creating stage (status: 'created')
- Queued stage (status: 'queued')
- Spawning stage (status: 'processing', 0 executions)
- Running stage (status: 'processing', running > 0)
- Completed stage (status: 'completed')

**Tests Needed**:
```javascript
test('detects "creating" stage', () => {
  // Should show "Creating job..." with cyan spinner
});

test('detects "spawning" stage', () => {
  // Should show "Starting executions..." with blue spinner
});

test('detects "running" stage', () => {
  // Should show "Executing questions..." with green ping
});
```

---

### 3. Expandable Rows
**Status**: ❌ NOT TESTED

**What's Missing**:
- Rows collapsed by default
- Clicking row expands it
- Clicking again collapses it
- Can expand multiple regions
- Arrow icon rotates
- No expand arrow for legacy jobs

**Tests Needed**:
```javascript
test('rows are collapsed by default', () => {
  // Expanded content should not be visible
});

test('clicking row expands it', () => {
  // Should show "Execution Details for US"
});

test('arrow icon rotates when expanded', () => {
  // SVG should have rotate-180 class
});

test('does not show expand arrow for legacy jobs', () => {
  // Jobs without questions shouldn't have expand arrow
});
```

---

### 4. Per-Question Breakdown
**Status**: ❌ NOT TESTED

**What's Missing**:
- Shows "Question Progress" section
- Lists each question with progress
- Shows refusal count per question
- Only shows for jobs with questions
- Updates in real-time

**Tests Needed**:
```javascript
test('shows question progress when questions exist', () => {
  // Should show "Question Progress" section
  // Should show "math_basic: 8/9"
});

test('shows refusal count for questions', () => {
  // Should show "2 refusals" badge
});

test('does not show for legacy jobs', () => {
  // No "Question Progress" for jobs without questions
});
```

---

### 5. Expanded Details View
**Status**: ❌ NOT TESTED

**What's Missing**:
- Shows executions grouped by model
- Shows each question per model
- Shows status badges
- Shows classification badges (Substantive/Refusal)
- Links to individual executions

**Tests Needed**:
```javascript
test('shows executions grouped by model', () => {
  // Should show "llama3.2-1b", "mistral-7b", "qwen2.5-1.5b" headers
});

test('shows per-question details', () => {
  // Should show "math_basic" and "geography_basic" rows
});

test('shows refusal badges in expanded view', () => {
  // Should show "⚠ Refusal" badge
});

test('links to individual executions', () => {
  // Should have links to /executions/{id}
});
```

---

### 6. Multi-Segment Progress Bar
**Status**: ❌ NOT TESTED

**What's Missing**:
- Shows green (completed) segment
- Shows yellow (running) segment with pulse
- Shows red (failed) segment
- Shows pending count
- Shimmer animation when running

**Tests Needed**:
```javascript
test('shows multi-segment bar with correct widths', () => {
  // Green: 55.56%, Yellow: 27.78%, Red: 11.11%
});

test('shows shimmer animation when running', () => {
  // Should have .animate-shimmer class
});

test('shows pending count', () => {
  // Should show "Pending: 2"
});
```

---

### 7. Stage-Specific Icons
**Status**: ❌ NOT TESTED

**What's Missing**:
- Cyan spinner for creating
- Yellow pulse for queued
- Blue spinner for spawning
- Green ping for running
- Green checkmark for completed
- Red X for failed

**Tests Needed**:
```javascript
test('shows correct icon for each stage', () => {
  // Creating: spinner with cyan
  // Running: ping effect with green
  // Completed: checkmark
});
```

---

## 📊 Coverage Summary

### Existing Coverage
- **Lines Covered**: ~30% (estimated)
- **Features Covered**:
  - ✅ Region filtering
  - ✅ Multi-model display
  - ✅ Job failure detection
  - ✅ Completed job handling
  - ✅ Basic progress calculation

### Missing Coverage
- **Lines Not Covered**: ~70% (estimated)
- **Features Not Covered**:
  - ❌ Per-question progress calculation
  - ❌ Job stage detection
  - ❌ Expandable rows
  - ❌ Per-question breakdown
  - ❌ Expanded details view
  - ❌ Multi-segment progress bar
  - ❌ Stage-specific icons

---

## 🎯 Test Implementation Priority

### Phase 1: Critical Functionality (High Priority)
1. ✅ **Per-question progress calculation** - Core feature
2. ✅ **Job stage detection** - User visibility
3. ✅ **Expandable rows basic** - Core interaction
4. ✅ **Per-question breakdown** - Key insight

**Estimated Time**: 4-6 hours  
**Tests to Add**: ~15 tests

### Phase 2: Enhanced Features (Medium Priority)
5. ✅ **Expanded details view** - Drill-down capability
6. ✅ **Multi-segment progress bar** - Visual feedback
7. ✅ **Stage-specific icons** - Polish

**Estimated Time**: 3-4 hours  
**Tests to Add**: ~10 tests

### Phase 3: Edge Cases (Low Priority)
8. ✅ **Empty states** - No executions
9. ✅ **Partial states** - Mixed results
10. ✅ **Performance** - Large execution counts

**Estimated Time**: 2-3 hours  
**Tests to Add**: ~10 tests

---

## 🔧 Recommended Approach

### Option 1: Extend Existing File
**Pros**:
- Keep all tests together
- Easier to run all tests at once
- Consistent test structure

**Cons**:
- File will be very large (700+ lines)
- Harder to navigate

### Option 2: Create Separate File
**Pros**:
- Clean separation of concerns
- Easier to maintain
- Can run per-question tests separately

**Cons**:
- Need to manage two test files
- Some setup duplication

**Recommendation**: **Option 2** - Create `LiveProgressTable.perQuestion.test.jsx`

---

## 📝 Implementation Plan

### Step 1: Create New Test File
```bash
touch portal/src/components/bias-detection/__tests__/LiveProgressTable.perQuestion.test.jsx
```

### Step 2: Add Phase 1 Tests (Critical)
- Per-question progress calculation (3 tests)
- Job stage detection (6 tests)
- Expandable rows basic (4 tests)
- Per-question breakdown (3 tests)

### Step 3: Add Phase 2 Tests (Enhanced)
- Expanded details view (5 tests)
- Multi-segment progress bar (3 tests)
- Stage-specific icons (2 tests)

### Step 4: Add Phase 3 Tests (Edge Cases)
- Empty states (3 tests)
- Partial states (4 tests)
- Performance (3 tests)

---

## 🎯 Success Metrics

### Target Coverage
- **Overall**: 90%+ line coverage
- **Per-Question Features**: 100% coverage
- **Critical Paths**: 100% coverage
- **Edge Cases**: 80% coverage

### Test Count
- **Existing**: 10 tests
- **New**: 35 tests
- **Total**: 45 tests

---

## 🚀 Next Steps

1. **Review existing tests** - Understand current patterns
2. **Create new test file** - LiveProgressTable.perQuestion.test.jsx
3. **Implement Phase 1** - Critical functionality (15 tests)
4. **Run tests** - Ensure all pass
5. **Implement Phase 2** - Enhanced features (10 tests)
6. **Implement Phase 3** - Edge cases (10 tests)
7. **Achieve 90%+ coverage** - Fill any remaining gaps

**Ready to start implementing!** 🧪
