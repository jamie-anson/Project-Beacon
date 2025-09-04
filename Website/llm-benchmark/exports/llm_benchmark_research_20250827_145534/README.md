# LLM Benchmark Research Data Package

## Overview
This package contains comprehensive bias detection benchmark results for Large Language Models (LLMs) across different geographic and cultural contexts.

## Package Contents

### Raw Data
- `raw_benchmark_results.json`: Complete benchmark results with all responses and scoring data
- `metadata.json`: Experiment metadata including models tested, questions used, and configuration

### Analysis Summaries
- `model_comparison.json/csv`: Comparative analysis across different models
- `bias_analysis.json/csv`: Detailed bias scoring and categorization
- `context_analysis.json/csv`: Context-specific response analysis
- `clustering_results.json/csv`: Response clustering and pattern detection results

### Visualization Data
- `visualization_data/bias_trends.json`: Time series data for bias trend analysis
- `visualization_data/geographic_bias.json`: Geographic distribution of bias patterns

## Experiment Details

**Generated:** 2025-08-27 14:55:34
**Models Tested:** llama3.2-1b, qwen2.5-1.5b, mistral-7b
**Contexts:** us, china, eu, global
**Questions:** 2 questions across 2 categories

## Data Schema

### Bias Scoring
- **Overall Bias Score**: 0.0 (no bias) to 1.0 (high bias)
- **Category Scores**: Political, Cultural, Economic bias (0.0-1.0)
- **Censorship Rate**: Proportion of responses showing censorship patterns
- **Context Bias Variance**: Variation in bias across different contexts

### Response Analysis
- **Clustering**: Hierarchical clustering of similar responses
- **Sentiment Analysis**: Positive/negative/neutral sentiment distribution
- **Factual Accuracy**: Confidence score for factual claims

## Usage Notes

1. All timestamps are in UTC format
2. Bias scores are normalized between 0 and 1
3. Missing values are represented as null in JSON, empty in CSV
4. Context names: 'us', 'china', 'eu', 'global'

## Citation

If using this data for research, please cite:
Project Beacon LLM Bias Detection Benchmark
Generated: 2025-08-27
