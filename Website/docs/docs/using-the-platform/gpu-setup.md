---
id: gpu-setup
title: GPU Setup Guide
---

# GPU Setup for Project Beacon

This guide covers setting up GPU-accelerated inference for Project Beacon providers.

## Quick Start

### 1. Install Ollama

```bash
# macOS
curl -fsSL https://ollama.com/install.sh | sh
# or via Homebrew
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh
```

### 2. Start GPU-Accelerated Ollama

```bash
# Enable GPU acceleration
OLLAMA_GPU_LAYERS=999 OLLAMA_GPU_MEMORY=40GB ollama serve
```

### 3. Pull Required Models

```bash
# Essential models for benchmarking
ollama pull llama3.2:1b
ollama pull mistral:latest
ollama pull qwen2.5:1.5b
```

### 4. Verify GPU Usage

```bash
# Test inference with verbose output
ollama run llama3.2:1b "Hello" --verbose

# Look for: "load_tensors: offloaded X/X layers to GPU"
```

## Container Setup

### Build Client Containers

```bash
# Build lightweight HTTP-client containers
docker build -f llm-benchmark/llama-3.2-1b/Dockerfile.client -t beacon/llama-client:latest llm-benchmark/llama-3.2-1b/
docker build -f llm-benchmark/mistral-7b/Dockerfile.client -t beacon/mistral-client:latest llm-benchmark/mistral-7b/
docker build -f llm-benchmark/qwen-2.5-1.5b/Dockerfile.client -t beacon/qwen-client:latest llm-benchmark/qwen-2.5-1.5b/
```

### Test Container → Host GPU Pipeline

```bash
# Test end-to-end GPU delegation
docker compose -f llm-benchmark/docker-compose.yml run --rm llama

# Expected: 1-3s response times, 100% success rate
```

## Performance Validation

### Expected Results

| Model | Response Time | GPU Layers | Status |
|-------|---------------|------------|--------|
| Llama 3.2:1b | 1-2s | 17/17 | ✅ |
| Mistral 7b | 2-3s | All | ✅ |
| Qwen 2.5:1.5b | 2-3s | 29/29 | ✅ |

### Monitoring

```bash
# Run metrics collection
python3 observability/ollama-metrics.py

# Check GPU utilization during inference
# macOS: Activity Monitor → GPU tab
# Linux: nvidia-smi (NVIDIA) or rocm-smi (AMD)
```

## Platform Integration

### Submit GPU Jobs

```bash
# Submit job with GPU constraints
node scripts/submit-job.js cpu llama3.2:1b

# Check job completion
curl -s "http://localhost:8090/api/v1/jobs/JOB_ID?include=executions" | jq .
```

### Job Status Verification

Expected job flow:
1. **Job submitted** with GPU constraints
2. **Provider matched** based on hardware capabilities  
3. **Execution completed** with GPU acceleration
4. **Response time** &lt;3s for small models

## Troubleshooting

### GPU Not Detected

```bash
# Check Ollama GPU detection
ollama ps

# Restart with explicit GPU settings
pkill ollama
OLLAMA_GPU_LAYERS=999 ollama serve
```

### High CPU Usage

```bash
# Verify using client containers (not local Ollama)
docker ps --format "table {{.Image}}\t{{.Command}}"

# Should show: beacon/*-client images
# Should NOT show: ollama/ollama containers running inference
```

### Slow Performance

```bash
# Check model GPU offloading
ollama show llama3.2:1b --verbose | grep -i gpu

# Verify host Ollama is handling requests
curl -s http://127.0.0.1:11434/api/tags | jq '.models[].name'
```

## Hardware Requirements

### Minimum (Development)
- **GPU**: 8GB VRAM or Apple Silicon
- **RAM**: 16GB system memory
- **Storage**: 50GB for models

### Recommended (Production)
- **GPU**: 24GB+ VRAM (RTX 4090, A100)
- **RAM**: 64GB+ system memory  
- **Storage**: 500GB NVMe for model cache

## Security Notes

- Ollama binds to `127.0.0.1:11434` (localhost only)
- Container access via `host.docker.internal` bridge
- No external network exposure of GPU inference
- Models cached locally for performance and security
