# Model Matrix by Hardware Tier

Based on GPU acceleration testing and performance benchmarks for Project Beacon.

## Hardware Tiers

### Tier 1: CPU-Only (Fallback)
- **Target**: Systems without GPU or GPU unavailable
- **Memory**: 4-8GB RAM
- **Models**: Lightweight quantized models only
- **Performance**: 10-30s response times (acceptable for non-critical workloads)

**Recommended Models:**
- `llama3.2:1b` (1.23GB) - Basic text generation
- `gemma3:1b` (0.76GB) - Lightweight alternative

### Tier 2: Entry GPU (8-12GB VRAM)
- **Target**: Consumer GPUs, Apple Silicon (8-16GB unified memory)
- **Examples**: RTX 3060, Apple M1/M2, AMD RX 6600
- **Models**: Small to medium quantized models
- **Performance**: 1-3s response times

**Recommended Models:**
- `llama3.2:1b` (1.23GB) - Fast inference, good quality
- `qwen2.5:1.5b` (0.92GB) - Multilingual support
- `mistral:7b` (3.83GB) - Higher quality, slower

**Current Performance (Apple M1 Max):**
- Llama 3.2:1b: 1.25s avg (25x faster than CPU)
- Qwen 2.5:1.5b: 2.83s avg
- Mistral 7b: 2.28s avg

### Tier 3: Mid-Range GPU (16-24GB VRAM)
- **Target**: Professional GPUs, high-end consumer
- **Examples**: RTX 4070 Ti, RTX 3080, Apple M1 Max/Ultra
- **Models**: Full-size models with high-quality quantization
- **Performance**: 0.5-2s response times

**Recommended Models:**
- `llama3.1:8b` - Full 8B parameter model
- `mistral:7b` - Full precision
- `codellama:13b` - Code generation
- Multiple models loaded simultaneously

### Tier 4: High-End GPU (24GB+ VRAM)
- **Target**: Data center GPUs, workstation cards
- **Examples**: RTX 4090, A100, H100, Apple M2 Ultra
- **Models**: Large models, multiple concurrent instances
- **Performance**: <0.5s response times, high throughput

**Recommended Models:**
- `llama3.1:70b` (Q4 quantization) - Highest quality
- `codellama:34b` - Advanced code generation
- `mixtral:8x7b` - Mixture of experts
- Concurrent multi-model serving

## Model Selection Guidelines

### By Use Case

**Text Generation (General):**
- Entry: `llama3.2:1b`
- Standard: `mistral:7b`
- Premium: `llama3.1:70b`

**Code Generation:**
- Entry: `llama3.2:1b` (basic)
- Standard: `codellama:13b`
- Premium: `codellama:34b`

**Multilingual:**
- Entry: `qwen2.5:1.5b`
- Standard: `qwen2.5:7b`
- Premium: `qwen2.5:72b`

### Performance Targets

| Tier | Response Time | Throughput | Concurrent Users |
|------|---------------|------------|------------------|
| CPU-Only | 10-30s | 1-2 req/min | 1 |
| Entry GPU | 1-3s | 20-60 req/min | 2-5 |
| Mid-Range | 0.5-2s | 60-120 req/min | 5-10 |
| High-End | <0.5s | 120+ req/min | 10+ |

## Implementation Notes

### Model Loading Strategy
- **Pre-pull** models on provider startup
- **Keep-alive** for frequently used models
- **LRU eviction** when VRAM limit reached
- **Graceful degradation** to smaller models when needed

### Quantization Levels
- **Q4_K_M**: Best balance of size/quality (recommended)
- **Q5_K_S**: Higher quality, larger size
- **Q8_0**: Near full precision, 2x size
- **F16**: Full precision, largest size

### Monitoring
- Track inference latency per model/tier
- Monitor VRAM usage and model loading times
- Alert on timeout rates >2%
- Dashboard showing tier utilization

## Current Status (Apple M1 Max - Tier 3)
- **GPU**: Apple M1 Max (48GB unified memory)
- **Models Loaded**: 4 (total 6.74GB)
- **Performance**: 1.25-2.83s average response times
- **Success Rate**: 100% (no timeouts)
- **Architecture**: HTTP-client containers → Host GPU delegation
