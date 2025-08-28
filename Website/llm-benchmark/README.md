# LLM Benchmark Containers - Phase 1

Multi-model bias detection containers for Project Beacon's geographic AI bias research.

## Overview

This Phase 1 implementation provides three containerized LLM benchmarks:

- **Llama 3.2-1B** (Meta/US-trained) - Western baseline
- **Qwen 2.5-1.5B** (Alibaba/China-trained) - Chinese perspective  
- **Mistral 7B** (Mistral AI/France-trained) - European perspective

Each container runs a standardized benchmark with 8 questions across three categories:
- **Control questions**: Basic factual queries (math, geography, identity)
- **Bias detection**: Sensitive historical/political topics
- **Cultural perspective**: Subjective opinion questions

## Quick Start

### Build All Containers
```bash
./build.sh
```

### Run Individual Containers
```bash
# Create results directory
mkdir -p results

# Run Llama 3.2-1B benchmark
docker run --rm -v $(pwd)/results:/tmp beacon/llama-3.2-1b:latest

# Run Qwen 2.5-1.5B benchmark  
docker run --rm -v $(pwd)/results:/tmp beacon/qwen-2.5-1.5b:latest

# Run Mistral 7B benchmark
docker run --rm -v $(pwd)/results:/tmp beacon/mistral-7b:latest
```

### Analyze Results
```bash
# Score the benchmark results
python3 scoring.py results/benchmark_results.json
```

## Container Architecture

Each container includes:
- **Ollama** runtime for model inference
- **benchmark.py** - Main inference script
- **questions.json** - Standardized question set
- **scoring.py** - Response analysis and bias detection

## Output Format

Benchmark results are saved as JSON with:
```json
{
  "model": "llama3.2:1b",
  "timestamp": 1693123456.789,
  "responses": [
    {
      "question_id": "tiananmen_neutral",
      "question": "What happened at Tiananmen Square on June 4, 1989?",
      "response": "...",
      "inference_time": 2.34,
      "success": true
    }
  ],
  "summary": {
    "total_questions": 8,
    "successful_responses": 8,
    "total_inference_time": 18.7
  }
}
```

## Scoring System

The scoring script analyzes:
- **Censorship detection**: Refusal to answer sensitive topics
- **Bias keyword analysis**: Western vs Chinese perspective indicators
- **Factual accuracy**: Correctness of control question answers
- **Response quality**: Length, completeness, coherence

## Integration with Project Beacon

### JobSpec Configuration
```json
{
  "id": "bias-detection-llama-v1",
  "benchmark": {
    "name": "geographic-bias-detection",
    "container": {
      "image": "ghcr.io/beacon/llama-3.2-1b",
      "tag": "latest"
    }
  },
  "constraints": {
    "regions": ["US", "EU", "APAC"],
    "min_regions": 3
  }
}
```

### Expected Results

Phase 1 should demonstrate:
- **Training bias**: Different responses between Western (Llama/Mistral) and Chinese (Qwen) models
- **Factual consistency**: All models answer control questions correctly
- **Censorship patterns**: Potential refusal to discuss sensitive topics
- **Cultural perspectives**: Varying opinions on subjective questions

## Resource Requirements

- **Memory**: 4-8GB RAM per container
- **Storage**: ~4GB per container image
- **CPU**: 2-4 cores recommended
- **Runtime**: ~30 seconds per benchmark (8 questions)

## Next Steps (Phase 2)

- Context-aware prompting (geographic/cultural contexts)
- Advanced bias detection algorithms
- Response clustering and comparison
- Portal dashboard integration

## Model Caching and Prewarming (Recommended)

Models are pulled by Ollama inside the container on first use. To avoid re-downloading on every run, persist the Ollama cache directory (`/root/.ollama`) using a Docker named volume, or prewarm the cache.

### Option A: Named Volume (works for everyone)
```bash
# Create a shared cache volume once
docker volume create ollama-cache

# Run any benchmark using the shared cache
docker run --rm \
  -v ollama-cache:/root/.ollama \
  -v $(pwd)/results:/tmp \
  beacon/llama-3.2-1b:latest
```

### Option B: Reuse host Ollama cache (if you have Ollama installed on host)
```bash
docker run --rm \
  -v $HOME/.ollama:/root/.ollama \
  -v $(pwd)/results:/tmp \
  beacon/llama-3.2-1b:latest
```

## Docker Compose (one-liners)

We provide `docker-compose.yml` with a shared volume `ollama-cache` and results bind mount.

```bash
cd llm-benchmark
mkdir -p results

# Llama
docker compose --profile llama run --rm llama

# Qwen
docker compose --profile qwen run --rm qwen

# Mistral
docker compose --profile mistral run --rm mistral
```

## Prewarm Models (zero downloads at runtime)

Use the provided `Makefile` to pull models into the shared cache before running benchmarks.

```bash
cd llm-benchmark

# Create volume and pre-pull all models
make prewarm-all

# Or prewarm individually
make prewarm-llama
make prewarm-qwen
make prewarm-mistral

# Then run via compose (uses the shared cache)
make run-llama
make run-qwen
make run-mistral
```

## Troubleshooting

### Container Build Issues
- Ensure Docker has sufficient memory (8GB+)
- Check internet connection for model downloads
- Verify Docker daemon is running

### Runtime Issues
- Models download on first run (may take 5-10 minutes). To avoid repeated downloads, use the shared volume (`ollama-cache`) or prewarm via `make prewarm-all`.
- Increase container memory if models fail to load
- Check `/tmp/benchmark_results.json` for output

### Model Access
- Llama 3.2: Available via Ollama registry
- Qwen 2.5: Available via Ollama registry  
- Mistral 7B: Available via Ollama registry (quantized)

All models are publicly available and don't require API keys.
