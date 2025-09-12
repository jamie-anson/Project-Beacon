# Docker Model Runner Research for Project Beacon

## Executive Summary

Docker Model Runner (DMR) is a compelling technology that could significantly enhance Project Beacon's LLM infrastructure. It offers native host-based inference, OpenAI API compatibility, and seamless Docker ecosystem integration that could simplify our current Ollama-based architecture.

**Recommendation**: High potential for adoption, with careful consideration of Linux support timeline for Golem provider deployment.

## What is Docker Model Runner?

Docker Model Runner is Docker's solution for running Large Language Models locally, introduced in Docker Desktop 4.40+. Unlike traditional containerized approaches, DMR runs inference engines natively on the host for optimal performance.

### Key Features

- **Host-based inference**: Uses native llama.cpp execution (no containerization overhead)
- **OpenAI-compatible API**: Drop-in replacement for OpenAI endpoints
- **OCI Artifact support**: Models distributed as standard Docker artifacts via Docker Hub
- **Multiple connection methods**: Docker socket, internal DNS (`model-runner.docker.internal`), or TCP
- **GPU optimization**: Optimized for Apple Silicon Metal API, NVIDIA GPUs
- **Docker CLI integration**: Native `docker model` commands for familiar workflow

### Architecture

```
Application → Docker Model Runner API → llama.cpp (host) → GPU
```

Models are:
- Pulled as OCI artifacts from Docker Hub
- Cached locally for fast access
- Loaded into memory on-demand
- Unloaded after 5 minutes of inactivity

## Comparison with Current Ollama Architecture

### Current Project Beacon Setup
```
Container → HTTP Client → host.docker.internal:11434 → Ollama → GPU
```

### Docker Model Runner Setup
```
Container → model-runner.docker.internal → Docker Model Runner → llama.cpp → GPU
```

### Advantages Over Ollama

1. **Better Docker Integration**
   - Native `docker model` CLI vs separate Ollama tooling
   - Consistent with Docker ecosystem patterns
   - No separate installation/configuration required

2. **Standardized Model Distribution**
   - OCI artifacts vs Ollama's proprietary format
   - Leverage existing Docker Hub infrastructure
   - Version control and registry management
   - Private registry support for enterprise

3. **Performance Benefits**
   - Host-based execution avoids VM overhead
   - Optimized GPU acceleration paths
   - Faster model loading and inference

4. **Reliability Improvements**
   - Addresses known Ollama performance degradation issues
   - More stable inference engine
   - Better resource management

5. **API Flexibility**
   - Multiple connection methods vs single REST API
   - OpenAI-compatible endpoints for easy migration
   - Better integration patterns for containerized apps

## Platform Support Analysis

### Current Support (as of 2024)
- ✅ **macOS Apple Silicon**: Full support with Metal GPU acceleration
- ✅ **Windows amd64**: NVIDIA GPU support (drivers 576.57+)
- ✅ **Windows arm64**: Qualcomm Adreno GPU support (6xx series+)
- ⚠️ **Linux**: Docker Engine support for CPU + NVIDIA (drivers 575.57.08+)

### Linux Support Status

**Current State**: Limited Linux support available in Docker Engine
- CPU inference: ✅ Available
- NVIDIA GPU: ✅ Available (requires drivers 575.57.08+)
- AMD GPU: ❓ Not explicitly documented

**Implications for Golem Providers**:
- Most Golem providers run on Linux VMs/containers
- Current Linux support may be sufficient for basic deployment
- GPU acceleration depends on provider hardware configuration
- Need to verify compatibility with Golem's virtualization layer

## Project Beacon Integration Potential

### High-Value Use Cases

1. **Local Development Environment**
   - Replace Ollama installation complexity
   - Unified Docker workflow for developers
   - Consistent model management across team

2. **Provider Standardization**
   - Standardized deployment across all provider types
   - Simplified provider onboarding process
   - Consistent API endpoints for hybrid router

3. **Model Distribution**
   - Centralized model registry on Docker Hub
   - Version-controlled model updates
   - Private models for enterprise customers

4. **Simplified Architecture**
   - Eliminate separate Ollama installation steps
   - Reduce provider setup complexity
   - Better integration with existing Docker workflows

### Integration Points

1. **LLM Benchmark Containers**
   - Replace `OLLAMA_BASE_URL=host.docker.internal:11434`
   - Use `OLLAMA_BASE_URL=http://model-runner.docker.internal/engines/llama.cpp/v1`
   - Maintain OpenAI-compatible API calls

2. **Hybrid Router**
   - Add Docker Model Runner provider type
   - Route requests to `model-runner.docker.internal` endpoints
   - Support multiple inference backends

3. **Golem Provider Deployment**
   - Include Docker Model Runner in provider images
   - Standardize model pulling via `docker model pull`
   - Simplify provider startup scripts

## Limitations and Considerations

### Current Limitations

1. **Beta Status**: Experimental feature, subject to changes
2. **Platform Coverage**: Limited Linux support may affect some Golem providers
3. **Docker Version Requirements**: Requires Docker Desktop 4.40+ or newer Engine
4. **Model Format**: GGUF format only (matches our current setup)
5. **Documentation**: Limited compared to mature Ollama ecosystem

### Risk Assessment

**Low Risk**:
- GGUF format compatibility (already using)
- OpenAI API compatibility (existing containers work)
- Docker ecosystem integration (familiar tooling)

**Medium Risk**:
- Beta software stability
- Limited troubleshooting resources
- Platform support gaps

**High Risk**:
- Linux support limitations for Golem providers
- Potential breaking changes during beta period

## Implementation Recommendations

### Phase 1: Local Development (Immediate)
- Test Docker Model Runner with existing LLM benchmark containers
- Validate performance vs current Ollama setup
- Document migration process for development team

### Phase 2: Proof of Concept (Short-term)
- Deploy test Golem provider with Docker Model Runner
- Validate Linux compatibility in Golem environment
- Performance benchmarking vs Ollama-based providers

### Phase 3: Gradual Migration (Medium-term)
- Migrate local development environments
- Update provider deployment scripts
- Maintain Ollama fallback for unsupported platforms

### Phase 4: Full Adoption (Long-term)
- Standardize on Docker Model Runner across all providers
- Leverage OCI artifact distribution for model updates
- Integrate with hybrid router for unified inference

## Linux Support Investigation Results

### Docker Model Runner Linux Support Status

**Current Support (Confirmed 2024)**:
- ✅ **Docker CE on Linux**: Full support available including WSL2
- ✅ **Installation**: Via `docker-model-plugin` package (apt/dnf)
- ✅ **GPU Support**: NVIDIA GPUs with drivers 575.57.08+
- ✅ **Architecture**: x86-64 Linux systems supported
- ✅ **API Access**: Available on `localhost:12434` (not Docker socket in CE)

**Key Differences from Docker Desktop**:
- Runs as containerized service (`docker/model-runner` image)
- Uses `localhost:12434` instead of `model-runner.docker.internal`
- Requires manual installation of plugin package
- Container access via `172.17.0.1:12434` or `--add-host` configuration

### Golem Provider Compatibility Analysis

**Golem Provider Requirements**:
- Linux x86-64 architecture ✅ (matches Docker Model Runner)
- Nested virtualization enabled ✅ (compatible with Docker containers)
- KVM support for VM runtime ✅ (Docker Model Runner runs in containers)
- Network access for model downloads ✅ (can access Docker Hub)

**Compatibility Assessment**: **HIGH COMPATIBILITY**

1. **Architecture Match**: Both require Linux x86-64
2. **Virtualization**: Golem's nested virtualization supports Docker containers
3. **Resource Access**: Docker Model Runner can run within Golem's VM constraints
4. **Network Access**: Golem providers can access external registries for model pulling
5. **GPU Passthrough**: NVIDIA GPU support available in both systems

**Potential Integration Path**:
```bash
# In Golem provider environment
apt-get install docker-model-plugin
docker model pull ai/llama3.2:1b
# Container access: http://172.17.0.1:12434/engines/llama.cpp/v1
```

### Implementation Considerations

**Advantages for Golem Providers**:
- Standardized model distribution via Docker registries
- Consistent deployment across all provider types
- Better resource management than separate Ollama installation
- Native Docker integration matches Golem's container-based approach

**Challenges**:
- Different API endpoint (`172.17.0.1:12434` vs `model-runner.docker.internal`)
- Requires plugin installation in provider images
- Container-based deployment adds slight overhead vs native Ollama

**Recommendation**: **PROCEED WITH PILOT TESTING**
Linux support is mature enough for Golem provider deployment. The containerized approach actually aligns well with Golem's architecture.

## Conclusion

Docker Model Runner represents a significant opportunity to simplify and standardize Project Beacon's LLM infrastructure. The technology aligns well with our containerized architecture while potentially solving reliability and performance issues with the current Ollama approach.

**Key Success Factors**:
- Linux support validation for Golem providers
- Performance benchmarking vs current setup
- Gradual migration strategy with fallback options
- Team training on new Docker Model Runner workflows

**Timeline**: Ready for proof-of-concept testing immediately, with potential production adoption in Q1 2025 pending Linux compatibility validation.

## Advanced Alternative: vLLM for High-Performance Inference

### Video Insights: Performance Breakthrough

Recent performance testing reveals **vLLM** as a superior alternative to both Ollama and Docker Model Runner for high-throughput scenarios:

**Performance Comparison**:
- **LM Studio/Ollama**: Single concurrent request limitation
- **Docker Model Runner**: Parallel processing capability 
- **vLLM**: **5,800 tokens/second** with 256 concurrent requests

### Key Performance Factors

**1. Parallelism**
- **Problem**: Ollama/LM Studio process requests sequentially
- **Solution**: vLLM handles multiple concurrent requests to saturate GPU
- **Benefit**: Dramatically reduced latency under load

**2. FP8 Quantization**
- **Technology**: 16-bit → 8-bit floating-point weight conversion
- **Hardware**: Optimized for NVIDIA GPUs (RTX Pro 6000 demonstrated)
- **Impact**: Significant speed improvements while maintaining quality

**3. GPU Saturation**
- **Current Issue**: Single requests underutilize GPU capacity
- **vLLM Advantage**: Concurrent processing maximizes GPU throughput
- **Result**: Better cost efficiency per inference

### vLLM vs Docker Model Runner

| Feature | Docker Model Runner | vLLM |
|---------|-------------------|------|
| **Concurrent Requests** | Limited | 256+ demonstrated |
| **Throughput** | ~tokens/second | 5,800+ tokens/second |
| **GPU Utilization** | Moderate | High saturation |
| **Quantization** | GGUF (4-bit) | FP8 (8-bit optimized) |
| **Deployment** | Docker native | Docker + GPU optimization |
| **Use Case** | Development/single-user | Production/multi-user |

### Project Beacon Integration Potential

**High-Value Scenarios**:
1. **Multi-Region Load**: Handle concurrent requests across regions
2. **Provider Efficiency**: Maximize GPU utilization on expensive hardware
3. **Cost Optimization**: Better throughput per dollar spent on GPU time
4. **Benchmark Performance**: Faster execution of LLM benchmark suites

**Implementation Considerations**:
- **Complexity**: More setup than Docker Model Runner
- **Hardware Requirements**: NVIDIA GPU with FP8 support preferred
- **Memory**: Higher VRAM requirements for concurrent processing
- **Network**: Better suited for centralized inference vs edge deployment

### Hybrid Strategy Recommendation

**Development Environment**: Docker Model Runner
- Simpler setup and debugging
- Good for single-developer workflows
- Consistent with Docker ecosystem

**Production Providers**: vLLM
- Maximum throughput for paid GPU time
- Better concurrent user handling
- Optimized for high-performance scenarios

**Edge Cases**: Ollama fallback
- Platforms without Docker support
- Resource-constrained environments
- Rapid prototyping needs

### Implementation Priority

1. **Phase 1**: Continue Docker Model Runner evaluation for development
2. **Phase 2**: Pilot vLLM on high-performance Golem providers
3. **Phase 3**: Performance benchmark comparison across all three options
4. **Phase 4**: Hybrid deployment based on provider capabilities and cost structure

This multi-tier approach maximizes both developer experience and production performance while maintaining flexibility across different deployment scenarios.

## Provider Requirements Simplification

### Current Ollama-Based Provider Complexity

**Installation Requirements**:
- Install and configure Yagna daemon (ports 7464-7465)
- Set up separate Ollama installation and configuration
- Manage model downloads and storage manually
- Configure health/inference HTTP service (port 8080)
- Set up TLS termination (port 443)
- Handle firewall configuration for multiple ports
- Coordinate between multiple services (Yagna + Ollama + health server)

**Provider Onboarding Challenges**:
- Custom LLM infrastructure expertise required
- Provider-specific configurations and troubleshooting
- Manual model updates and version management
- Complex service coordination and monitoring

### Simplified Requirements with New Approaches

**Docker Model Runner**:
```bash
# Simplified installation
apt-get install docker-model-plugin
docker model pull ai/llama3.2:1b
# Automatic model caching and management
```

**vLLM for High-Performance Providers**:
```bash
# GPU-optimized deployment
docker run --gpus all vllm/vllm-openai:latest
# Concurrent request handling, FP8 quantization
```

### Provider Ecosystem Benefits

**1. Lower Barrier to Entry**
- **Before**: Custom Ollama setup, model management, service coordination
- **After**: Standard Docker commands, automatic model distribution
- **Impact**: More providers can participate with familiar tooling

**2. Better Hardware Utilization**
- **Current**: Single-request processing wastes GPU cycles
- **vLLM**: Concurrent processing maximizes expensive GPU investment
- **Result**: Better ROI attracts high-performance providers

**3. Standardized Deployment**
- **Before**: Provider-specific configurations and troubleshooting
- **After**: Consistent Docker-based deployment across all providers
- **Benefit**: Reduced technical expertise requirements

**4. Automatic Updates**
- **Before**: Manual model updates and version management
- **After**: `docker model pull` for automatic updates via registries
- **Advantage**: Simplified maintenance and version control

### Provider Matching Improvements

**Development-Friendly Providers**: Docker Model Runner
- Familiar Docker tooling vs custom LLM infrastructure
- Consistent API endpoints across all providers
- Built-in health monitoring through Docker

**High-Performance GPU Providers**: vLLM
- Maximum GPU utilization = better ROI for expensive hardware
- Concurrent request handling = serve multiple jobs simultaneously
- FP8 quantization = faster inference on NVIDIA GPUs

**Clear Upgrade Path**: Ollama → Docker Model Runner → vLLM
- Providers can start simple and upgrade based on capabilities
- Standardized migration path reduces complexity
