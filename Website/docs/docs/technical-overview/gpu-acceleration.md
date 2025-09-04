---
id: gpu-acceleration
title: GPU Acceleration
---

# GPU-Accelerated Inference

Project Beacon implements GPU-accelerated inference for LLM workloads through a host-delegation architecture that achieves 25x performance improvements over CPU-only inference.

## Architecture Overview

### Container → Host GPU Delegation

```
┌─────────────────┐    HTTP API     ┌──────────────────┐
│  Client         │ ──────────────→ │  Host Ollama     │
│  Container      │                 │  (GPU-enabled)   │
│  (HTTP only)    │                 │                  │
└─────────────────┘                 └──────────────────┘
                                             │
                                             ▼
                                    ┌──────────────────┐
                                    │  GPU Hardware    │
                                    │  (Metal/CUDA)    │
                                    └──────────────────┘
```

### Key Components

- **Client Containers**: Lightweight HTTP-only containers that make API calls
- **Host Ollama**: GPU-accelerated inference server running on host
- **Model Loading**: Pre-pulled models with GPU layer offloading
- **Networking**: Secure localhost binding with container bridge access

## Performance Results

### Benchmark Comparison

| Architecture | Response Time | Success Rate | GPU Utilization |
|--------------|---------------|--------------|-----------------|
| **Before (CPU)** | 30-70s | 60% (timeouts) | 0% |
| **After (GPU)** | 1.25-2.83s | 100% | 90%+ |

### Model Performance (Apple M1 Max)

| Model | Size | Avg Response | GPU Layers |
|-------|------|--------------|------------|
| Llama 3.2:1b | 1.23GB | 1.25s | 17/17 |
| Qwen 2.5:1.5b | 0.92GB | 2.83s | 29/29 |
| Mistral 7b | 3.83GB | 2.28s | All |

## Hardware Tiers

### Tier 1: CPU-Only (Fallback)
- **Target**: 4-8GB RAM, no GPU
- **Models**: `llama3.2:1b`, `gemma3:1b`
- **Performance**: 10-30s response times

### Tier 2: Entry GPU (8-12GB VRAM)
- **Target**: RTX 3060, Apple M1/M2, RX 6600
- **Models**: Small to medium quantized models
- **Performance**: 1-3s response times

### Tier 3: Mid-Range GPU (16-24GB VRAM)
- **Target**: RTX 4070 Ti, Apple M1 Max/Ultra
- **Models**: Full-size models, multiple concurrent
- **Performance**: 0.5-2s response times

### Tier 4: High-End GPU (24GB+ VRAM)
- **Target**: RTX 4090, A100, H100
- **Models**: Large models, high throughput
- **Performance**: &lt;0.5s response times

## Implementation

### Container Configuration

Client containers use `Dockerfile.client` pattern:

```dockerfile
FROM python:3.11-slim
RUN pip install requests numpy pandas
# Copy benchmark files only
ENV OLLAMA_BASE_URL=http://host.docker.internal:11434
ENTRYPOINT ["python3", "benchmark.py"]
```

### Host Ollama Setup

```bash
# Start with GPU acceleration
OLLAMA_GPU_LAYERS=999 OLLAMA_GPU_MEMORY=40GB ollama serve

# Verify GPU usage
ollama run llama3.2:1b "test" --verbose
# Should show: "load_tensors: offloaded X/X layers to GPU"
```

### Docker Compose Integration

```yaml
services:
  llama:
    image: beacon/llama-client:latest
    environment:
      - OLLAMA_BASE_URL=http://host.docker.internal:11434
      - BENCHMARK_MODEL=llama3.2:1b
    volumes:
      - ./results:/tmp
```

## Security

- **Localhost Binding**: Ollama bound to `127.0.0.1:11434` only
- **Container Access**: Restricted via Docker's `host.docker.internal` bridge
- **No External Exposure**: GPU inference server not accessible from network

## Monitoring

### Metrics Collection

```python
# Basic metrics via ollama-metrics.py
{
  "gpu_stats": {"gpu_name": "Apple M1 Max"},
  "models_loaded": 4,
  "inference_test": {
    "status": "success",
    "duration_seconds": 0.152
  }
}
```

### Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| Timeout Rate | &lt;2% | 0% |
| Avg Response Time | &lt;3s | 1.25s |
| GPU Utilization | >80% | 90%+ |

## Deployment

### Production Checklist

- [ ] GPU drivers installed (NVIDIA/AMD)
- [ ] Ollama configured with GPU acceleration
- [ ] Client containers built and tested
- [ ] Monitoring and alerting configured
- [ ] Performance benchmarks validated

### Scaling Considerations

- **Model Pre-loading**: Keep frequently used models warm
- **Concurrent Requests**: Balance GPU memory vs throughput
- **Failover**: Graceful degradation to CPU when GPU unavailable
- **Multi-GPU**: Load balancing across multiple GPUs

## Troubleshooting

### Common Issues

**GPU Not Detected:**
```bash
# Check GPU availability
ollama ps
# Should show GPU info in model loading logs
```

**High CPU Usage:**
```bash
# Verify containers using client architecture
docker ps --format "table {{.Image}}\t{{.Command}}"
# Should show beacon/*-client images, not ollama/ollama
```

**Slow Inference:**
```bash
# Check GPU layer offloading
ollama show model:name --verbose | grep -i gpu
# Should show layers offloaded to GPU
```

## Next Steps

- Deploy on NVIDIA/AMD production hosts
- Implement multi-model concurrent serving
- Add advanced observability and alerting
- Scale testing across multiple regions
