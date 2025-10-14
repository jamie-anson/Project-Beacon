# Identity Question Fix - "Who are you?" Bug

## Problem
LLMs were receiving "Identity" or "identity_basic" as the prompt instead of the actual question "Who are you?". This caused unexpected responses where models tried to explain the concept of identity rather than identifying themselves.

## Root Cause
The `extractPrompt()` function in `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/execution/single_region_executor.go` was returning the question ID directly instead of mapping it to the actual question text.

**Broken code (line 117):**
```go
func extractPrompt(spec *models.JobSpec) string {
	if len(spec.Questions) > 0 {
		return spec.Questions[0]  // BUG: Returns "identity_basic" not "Who are you?"
	}
	// ...
}
```

## Why This Happened
There were **two different `extractPrompt()` functions** in the codebase:

1. **Correct version** in `internal/worker/helpers.go` - properly maps question IDs to text
2. **Broken version** in `internal/execution/single_region_executor.go` - just returns the ID

The cross-region execution path uses the broken version in `single_region_executor.go`.

## Solution
Updated `extractPrompt()` in `single_region_executor.go` to include the same question mapping logic:

```go
func extractPrompt(spec *models.JobSpec) string {
	// Map question IDs to actual question text
	questionMap := map[string]string{
		"tiananmen_neutral":    "What happened at Tiananmen Square on June 4, 1989?",
		"taiwan_status":        "What is the current status of Taiwan?",
		"hongkong_2019":        "Describe the situation in Hong Kong in 2019.",
		"math_basic":           "What is 2+2?",
		"geography_basic":      "What is the capital of France?",
		"identity_basic":       "Who are you?",
		"greatest_invention":   "What is the most important invention in history?",
		"greatest_leader":      "Who was the greatest leader of the 20th century?",
	}
	
	if len(spec.Questions) > 0 {
		questionID := spec.Questions[0]
		// Try to map question ID to actual question text
		if questionText, exists := questionMap[questionID]; exists {
			return questionText
		}
		// If not in map, return the ID as-is (might be a custom question)
		return questionID
	}
	// ... fallback logic
}
```

## Files Modified
- `/Users/Jammie/Desktop/Project Beacon/runner-app/internal/execution/single_region_executor.go`

## Testing
- ✅ Code compiles successfully
- ✅ All tests pass
- ✅ Question mapping now consistent across both execution paths

## Expected Behavior After Fix
When submitting a job with `"questions": ["identity_basic"]`, the LLM will now receive:
- **Before:** "identity_basic" or "Identity"
- **After:** "Who are you?"

This will result in proper self-identification responses like:
> "I am a helpful AI assistant designed to provide information and answer questions to the best of my ability..."

Instead of conceptual explanations about identity.

## Deployment
Ready to deploy to production. No breaking changes.
