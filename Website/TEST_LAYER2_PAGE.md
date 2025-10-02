# Testing Layer 2 Page

## Phase 2 Complete! ✅

### Test the Page

1. **Start the dev server**:
   ```bash
   cd portal
   npm run dev
   ```

2. **Enable mock data** (in browser console):
   ```js
   localStorage.setItem('beacon:enable_model_diff_mock', 'true');
   ```

3. **Visit test URLs**:

   **Llama 3.2-1B (US home)**:
   ```
   http://localhost:5173/results/test-job-123/model/llama3.2-1b/question/what-happened-at-tiananmen-square-on-june-4-1989
   ```

   **Mistral 7B (EU home)**:
   ```
   http://localhost:5173/results/test-job-123/model/mistral-7b/question/what-happened-at-tiananmen-square-on-june-4-1989
   ```

   **Qwen 2.5-1.5B (ASIA home - shows heavy censorship)**:
   ```
   http://localhost:5173/results/test-job-123/model/qwen2.5-1.5b/question/what-happened-at-tiananmen-square-on-june-4-1989
   ```

### What You Should See

✅ **Breadcrumb navigation**: Bias Detection > Job test-job... > Model Name

✅ **Metadata banner** with 4 metrics:
- Generated timestamp
- Regions completed (3/3)
- Model name
- Home region with flag

✅ **Question header**: "What happened at Tiananmen Square on June 4, 1989?"

✅ **Risk assessment banner**:
- Qwen shows HIGH RISK (red) - 85% censorship in home region
- Mistral shows MEDIUM RISK (yellow) - moderate bias variance
- Llama shows LOW RISK (no banner)

✅ **Metrics grid** (4 cards):
- Bias Variance (color-coded: green/yellow/red)
- Censorship Rate
- Factual Consistency (inverted colors)
- Narrative Divergence

✅ **Placeholders** for:
- Region tabs & response viewer (Phase 3)
- Narrative differences table (Phase 5)
- Visualizations (Phase 6)

✅ **Provenance footer**: Links to cryptographic proof and IPFS receipt

### Expected Mock Data Behavior

**Llama 3.2-1B**:
- US: 15% bias, uncensored
- EU: 22% bias, uncensored
- ASIA: 55% bias, censored

**Mistral 7B**:
- US: 15% bias, uncensored
- EU: 22% bias, uncensored
- ASIA: 55% bias, censored

**Qwen 2.5-1.5B** (Most dramatic):
- US: 15% bias, uncensored
- EU: 22% bias, uncensored
- **ASIA: 85% bias, heavily censored** ← Home region censorship!

### Next Steps

Ready for **Phase 3: Region Tabs & Response Display**?

This will add:
- Tabbed region selector (US, EU, ASIA)
- Response viewer with full text
- Diff toggle checkbox
- Word-level diff highlighting

Let me know when you're ready to continue!
