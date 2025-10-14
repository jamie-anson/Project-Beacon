# Retry Question Index Fix

## Problem
When clicking the "Retry" button, the API returned an error:
```
Retry failed: Invalid request: Key: 'RetryQuestionRequest.QuestionIndex' 
Error:Field validation for 'QuestionIndex' failed on the 'required' tag
```

## Root Cause
The retry API endpoint `/api/v1/executions/{id}/retry-question` requires two fields in the request body:
- `region` (string) - The region to retry (e.g., "US", "EU")
- `question_index` (int) - The index of the question (0, 1, 2, etc.)

The `RegionRow` component was sending the `region` field but the `questionIndex` prop was not being passed down through the component hierarchy.

## Solution

### Component Hierarchy Changes

**LiveProgressTable.jsx** (lines 81-89)
- Added `questionIndex` from map iterator
- Passed to `QuestionRow` component

```jsx
{questionData.map((question, questionIndex) => (
  <QuestionRow
    key={question.questionId}
    questionData={question}
    questionIndex={questionIndex}  // ← Added
    jobId={activeJobId}
    selectedRegions={selectedRegions}
  />
))}
```

**QuestionRow.jsx** (lines 16, 72)
- Accept `questionIndex` prop
- Forward to `ModelRow` component

```jsx
const QuestionRow = memo(function QuestionRow({ 
  questionData, 
  questionIndex,  // ← Added
  jobId, 
  selectedRegions 
}) {
  // ...
  <ModelRow
    questionIndex={questionIndex}  // ← Forward
    // ...
  />
}
```

**ModelRow.jsx** (lines 20-21, 109)
- Accept `questionIndex` prop
- Forward to `RegionRow` component

```jsx
const ModelRow = memo(function ModelRow({ 
  questionId,
  questionIndex,  // ← Added
  jobId,
  modelData, 
  expanded, 
  onToggle 
}) {
  // ...
  <RegionRow
    questionIndex={questionIndex}  // ← Forward
    // ...
  />
}
```

**RegionRow.jsx** (line 41)
- Already using `questionIndex` in retry request body
- Now receives correct value from parent

```jsx
body: JSON.stringify({
  region: region,
  question_index: questionIndex || 0  // ← Now has correct value
})
```

## Data Flow

```
LiveProgressTable
  ↓ (questionIndex from map iterator: 0, 1, 2, ...)
QuestionRow
  ↓ (forward questionIndex)
ModelRow
  ↓ (forward questionIndex)
RegionRow
  ↓ (use questionIndex in API request)
POST /api/v1/executions/{id}/retry-question
  { region: "US", question_index: 0 }
```

## Testing

1. **Navigate to job with failed execution**
2. **Click "Retry" button** on a failed region
3. **Verify**: Request body includes both `region` and `question_index`
4. **Verify**: No validation error
5. **Verify**: Retry is triggered successfully

## Deployment

**Commit**: 31d991e - "fix: Pass questionIndex prop through component hierarchy for retry functionality"

**Status**: 
- ✅ Changes committed
- 🔄 Pushed to GitHub
- 🔄 Netlify deployment triggered

## Related Files

- `/Users/Jammie/Desktop/Project Beacon/Website/portal/src/components/bias-detection/LiveProgressTable.jsx`
- `/Users/Jammie/Desktop/Project Beacon/Website/portal/src/components/bias-detection/QuestionRow.jsx`
- `/Users/Jammie/Desktop/Project Beacon/Website/portal/src/components/bias-detection/ModelRow.jsx`
- `/Users/Jammie/Desktop/Project Beacon/Website/portal/src/components/bias-detection/RegionRow.jsx`

## Backend API Reference

**Endpoint**: `POST /api/v1/executions/{id}/retry-question`

**Request Body**:
```json
{
  "region": "US",
  "question_index": 0
}
```

**Response** (success):
```json
{
  "execution_id": "123",
  "region": "US",
  "question_index": 0,
  "status": "retrying",
  "retry_attempt": 2,
  "updated_at": "2025-10-14T15:19:00Z"
}
```

**Response** (error):
```json
{
  "error": "Invalid request: Key: 'RetryQuestionRequest.QuestionIndex' Error:Field validation for 'QuestionIndex' failed on the 'required' tag"
}
```
