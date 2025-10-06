# LiveProgressTable Storybook Stories

**Updated**: 2025-10-06  
**Location**: `/portal/src/stories/LiveProgressTable.stories.jsx`

---

## 📚 Story Overview

The LiveProgressTable now has **10 comprehensive Storybook stories** covering all major use cases and edge cases.

---

## 🎭 Available Stories

### 1. **Default** (Basic Running Job)
- **Status**: Running
- **Regions**: US, EU, ASIA
- **Executions**: Mixed (running, completed, queued)
- **Use Case**: Standard job in progress

### 2. **FailedJob** ⚠️
- **Status**: Failed
- **Executions**: None (job failed before creating executions)
- **Features**: 
  - Shows failure alert with error message
  - All regions display "failed" status
  - Actionable guidance for users
- **Use Case**: Job-level failure detection

### 3. **StuckJob** ⏱️
- **Status**: Processing (20+ minutes)
- **Executions**: None
- **Features**:
  - Timeout detection (>15 minutes)
  - Shows "Job Timeout" alert
  - Displays stuck duration
- **Use Case**: Stuck job detection

### 4. **MixedExecutionFailures** 🔀
- **Status**: Completed
- **Executions**: Some succeeded, some failed
- **Features**:
  - Shows execution-level failures
  - Displays failure messages and codes
  - Mixed status badges per region
- **Use Case**: Partial failures

### 5. **CompletedJobMissingExecutions** 📊
- **Status**: Completed
- **Executions**: Only US (EU and ASIA missing)
- **Features**:
  - Shows pending status for missing regions
  - Job marked complete despite missing data
- **Use Case**: Edge case handling

### 6. **MultiQuestionJob** ❓
- **Status**: Processing
- **Questions**: q1, q2, q3
- **Features**:
  - Question progress breakdown
  - Per-question completion tracking
  - Refusal count badges
  - Expandable execution details
- **Use Case**: Multi-question job tracking

### 7. **MultiModelJob** 🤖
- **Status**: Completed
- **Models**: llama3.2-1b, mistral-7b, qwen2.5-1.5b
- **Features**:
  - Multi-model progress per region
  - Model count display (e.g., "3/3 models")
  - Mixed success/failure across models
- **Use Case**: Multi-model execution

### 8. **LoadingState** 🔄
- **Status**: Running
- **Loading**: Active
- **Features**:
  - Shows "refreshing" status
  - Animated loading indicators
  - Shimmer effects on progress bars
- **Use Case**: Active refresh state

### 9. **CompletedJob** ✅
- **Status**: Completed
- **Executions**: All successful
- **Features**:
  - Success indicators
  - Enabled "View Diffs" button
  - Completion message
  - All regions show completed status
- **Use Case**: Successful job completion

### 10. **JobWithRefusals** 🚫
- **Status**: Completed
- **Refusals**: Multiple content refusals
- **Features**:
  - Content refusal badges (⚠ Refusal)
  - Substantive response badges (✓ Substantive)
  - Refusal count in question progress
- **Use Case**: Content refusal tracking

### 11. **EmptyJob** 📭
- **Status**: Queued
- **Executions**: None yet
- **Features**:
  - Shows pending status for all regions
  - Displays expected execution count
  - No progress bars
- **Use Case**: Newly created job

---

## 🎨 Visual Features Demonstrated

### **Progress Indicators**
- ✅ Multi-segment progress bars (green/yellow/red)
- ✅ Shimmer animation for active jobs
- ✅ Percentage calculations
- ✅ Countdown timer (10-minute estimate)

### **Status Badges**
- ✅ Color-coded status (completed, running, failed, pending, etc.)
- ✅ Enhanced status detection (connecting, processing, completing, stalled)
- ✅ Job-level vs execution-level status

### **Failure Handling**
- ✅ Prominent failure alerts
- ✅ Contextual error messages
- ✅ Timeout detection and display
- ✅ Failure reason codes and stages

### **Interactive Elements**
- ✅ Expandable region rows
- ✅ Execution details grid (Model × Question)
- ✅ Retry buttons for failed executions
- ✅ Classification badges (substantive, refusal, error)
- ✅ Links to execution details

### **Multi-Question Support**
- ✅ Question progress breakdown
- ✅ Per-question completion tracking
- ✅ Refusal count badges
- ✅ Question-level retry functionality

### **Multi-Model Support**
- ✅ Model count per region
- ✅ Model × Question execution grid
- ✅ Individual model status tracking

---

## 🚀 Running Storybook

### **Start Storybook**
```bash
npm run storybook
```

### **View Stories**
Navigate to: `Bias Workflow > LiveProgressTable`

### **Interactive Controls**
- Modify `selectedRegions` array
- Toggle `loadingActive` state
- Change `isCompleted` flag
- Update `activeJob` data

---

## 📝 Story Structure

Each story follows this pattern:

```javascript
export const StoryName = {
  args: {
    activeJob: {
      id: 'unique-job-id',
      status: 'processing|completed|failed',
      job: {
        questions: ['q1', 'q2'],
        models: [{ id: 'model-id', regions: ['US', 'EU'] }]
      },
      executions: [
        {
          id: 'exec-id',
          region: 'us-east',
          model_id: 'model-id',
          question_id: 'q1',
          status: 'completed',
          response_classification: 'substantive'
        }
      ]
    },
    selectedRegions: ['US', 'EU'],
    loadingActive: false,
    refetchActive: () => {},
    activeJobId: 'unique-job-id',
    isCompleted: false,
    diffReady: false
  }
};
```

---

## 🎯 Coverage

### **Job States**
- ✅ Created/Queued
- ✅ Processing/Running
- ✅ Completed
- ✅ Failed
- ✅ Stuck/Timeout

### **Execution States**
- ✅ Pending
- ✅ Queued
- ✅ Connecting
- ✅ Processing
- ✅ Completing
- ✅ Completed
- ✅ Failed
- ✅ Stalled
- ✅ Refreshing

### **Edge Cases**
- ✅ No executions
- ✅ Missing executions for some regions
- ✅ All executions failed
- ✅ Mixed success/failure
- ✅ Job failed before executions
- ✅ Job stuck/timeout

### **Features**
- ✅ Single-model jobs
- ✅ Multi-model jobs
- ✅ Single-question jobs
- ✅ Multi-question jobs
- ✅ Content refusals
- ✅ Classification badges
- ✅ Retry functionality
- ✅ Progress tracking
- ✅ Time estimates

---

## 🔍 Testing with Storybook

### **Visual Regression Testing**
1. Open each story
2. Verify UI matches design
3. Check responsive behavior
4. Test interactions (expand, retry, etc.)

### **Accessibility Testing**
1. Test keyboard navigation
2. Verify ARIA labels
3. Check color contrast
4. Test screen reader compatibility

### **Interaction Testing**
1. Click region rows to expand/collapse
2. Click retry buttons (check console for API calls)
3. Click refresh button
4. Click View Diffs button (when enabled)
5. Click View full results link

---

## 📊 Story Metrics

- **Total Stories**: 11
- **Job States Covered**: 5
- **Execution States Covered**: 9
- **Edge Cases**: 6
- **Feature Scenarios**: 10+

---

## 🎓 Best Practices Demonstrated

1. **Comprehensive Coverage**: All major use cases and edge cases
2. **Realistic Data**: Mock data mirrors production scenarios
3. **Interactive Controls**: Storybook args for customization
4. **Clear Naming**: Descriptive story names
5. **Documentation**: Comments explain each scenario
6. **Consistent Structure**: All stories follow same pattern
7. **Visual Variety**: Different states and configurations

---

## 🚀 Next Steps

### **Optional Enhancements**
- [ ] Add Storybook interactions (automated clicks/tests)
- [ ] Add visual regression snapshots
- [ ] Create component-specific stories (ProgressHeader, RegionRow, etc.)
- [ ] Add accessibility addon tests
- [ ] Create story documentation pages

### **Maintenance**
- ✅ Update stories when adding new features
- ✅ Keep mock data realistic
- ✅ Document new edge cases
- ✅ Maintain story organization

---

**Status**: ✅ Complete - 11 comprehensive stories covering all scenarios
