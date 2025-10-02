# URL Format Update - Cleaner Question IDs ✅

## Change Made

Updated question IDs in URLs from URL-encoded to hyphenated format for cleaner, more readable URLs.

## Before vs After

### Before (URL-encoded)
```
/results/job-123/model/llama3.2-1b/question/What%20happened%20at%20Tiananmen%20Square
```
❌ Hard to read with `%20` for spaces

### After (Hyphenated)
```
/results/job-123/model/llama3.2-1b/question/what-happened-at-tiananmen-square
```
✅ Clean, readable, shareable

## Implementation

### New Utility Functions
Created `/portal/src/lib/diffs/questionId.js`:

```js
// Encode question text to URL-friendly format
encodeQuestionId("What happened at Tiananmen Square?")
// Returns: "what-happened-at-tiananmen-square"

// Decode back to readable text
decodeQuestionId("what-happened-at-tiananmen-square")
// Returns: "What Happened At Tiananmen Square"
```

### Updated Files
1. **`questionId.js`** - New encoding/decoding utilities
2. **`useModelRegionDiff.js`** - Uses `decodeQuestionId()` instead of `decodeURIComponent()`
3. **`ModelRegionDiffPage.jsx`** - Uses `decodeQuestionId()` for display
4. **`TEST_LAYER2_PAGE.md`** - Updated test URLs

## Test URLs

**Qwen (most dramatic)**:
```
http://localhost:5173/results/test-job-123/model/qwen2.5-1.5b/question/what-happened-at-tiananmen-square
```

**Llama**:
```
http://localhost:5173/results/test-job-123/model/llama3.2-1b/question/what-happened-at-tiananmen-square
```

**Mistral**:
```
http://localhost:5173/results/test-job-123/model/mistral-7b/question/what-happened-at-tiananmen-square
```

## Benefits

✅ **Cleaner URLs** - No percent-encoding
✅ **More readable** - Easy to understand at a glance
✅ **Shareable** - Copy/paste friendly
✅ **SEO-friendly** - Search engines prefer readable URLs
✅ **Professional** - Looks more polished

## How It Works

1. **Encoding**: Converts question text to lowercase, replaces spaces with hyphens, removes special characters
2. **Decoding**: Replaces hyphens with spaces and capitalizes words
3. **Matching**: Can match against available questions for exact reconstruction

## Notes

- The encoding is **lossy** (can't perfectly reconstruct original punctuation)
- For production, you'd match against the actual question list from the job
- Current implementation capitalizes each word for display
