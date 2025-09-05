# Phase 1 Discovery Report: Hybrid GPU Serverless Infrastructure

**Date**: 2025-09-05  
**Phase**: 1 - Discovery & Requirements  
**Status**: Complete  

## Executive Summary

Completed comprehensive evaluation of serverless GPU providers and container hosting solutions for Project Beacon's hybrid MVP infrastructure. Key findings support a cost-effective hybrid approach combining Golem providers for baseline capacity with serverless GPU fallbacks for burst and reliability.

## 1. Serverless GPU Provider Analysis

### 1.1 Modal - Python-Native Serverless
**Strengths:**
- Developer-friendly Python-first approach
- Sub-second container starts with warm pools
- Dynamic batching capabilities built-in
- Competitive GPU pricing structure

**Pricing (per second):**
- NVIDIA T4: $0.000164/sec ($0.59/hour)
- NVIDIA L4: $0.000222/sec ($0.80/hour) 
- NVIDIA A10: $0.000306/sec ($1.10/hour)
- NVIDIA L40S: $0.000542/sec ($1.95/hour)
- NVIDIA A100 40GB: $0.000583/sec ($2.10/hour)
- NVIDIA A100 80GB: $0.000694/sec ($2.50/hour)
- NVIDIA H100: $0.001097/sec ($3.95/hour)

**Geographic Coverage:** Global with multiple regions
**Cold Start:** <1s with warm instance pools
**Best For:** Sustained inference workloads, Python-heavy ML pipelines

### 1.2 RunPod Serverless - GPU-Optimized Functions
**Strengths:**
- Specialized for GPU workloads
- Per-second billing with no minimums
- Warm instance pool management
- 15% cost savings claimed over competitors

**Pricing:** Detailed pricing requires account access, but positioned as cost-competitive
**Geographic Coverage:** Multiple regions (US, EU, Asia)
**Cold Start:** Optimized for GPU workloads
**Best For:** Burst capacity, cost-sensitive workloads

### 1.3 Replicate - Model Hosting Platform
**Strengths:**
- Pre-built model ecosystem
- Simple API deployment via Cog
- Automatic scaling and instance management
- Pay-per-prediction pricing model

**Pricing:** Model-specific (e.g., $0.01/1000 tokens for text models)
**Geographic Coverage:** Global infrastructure
**Cold Start:** Managed automatically with scaling
**Best For:** Standard model deployments, rapid prototyping

### 1.4 Lambda Labs - On-Demand GPU Cloud
**Status:** Unable to access pricing due to connection issues
**Note:** Known for H100/A100 availability and competitive pricing
**Best For:** High-end GPU requirements, research workloads

## 2. Container Hosting Solutions Analysis

### 2.1 Railway - Git-Centric PaaS
**Strengths:**
- Simple git-based deployment
- Pay-per-use pricing model
- Global regions available
- Docker container support

**Pricing:**
- Hobby: $5/month minimum (includes $5 usage)
- Pro: $20/month minimum (includes $20 usage)
- Up to 32GB RAM / 32 vCPU per service (Pro)

**Geographic Coverage:** Global regions
**GPU Support:** Not explicitly mentioned
**Best For:** Simple deployments, development workflows

### 2.2 Fly.io - Global Edge Platform
**Strengths:**
- Extensive global presence (30+ regions)
- GPU support with A10, L40S, A100 options
- Machines API for dynamic scaling
- Strong multi-region capabilities

**GPU Pricing (per hour):**
- A10: $1.50/hour
- L40S: $1.25/hour  
- A100 40GB: $2.50/hour
- A100 80GB: $3.50/hour

**Geographic Coverage:** 30+ regions worldwide
**Best For:** Multi-region deployments, GPU workloads, edge computing

### 2.3 Render - Managed Container Hosting
**Status:** Limited information gathered
**Note:** Docker Compose-like deployment model
**Best For:** Multi-service applications

## 3. Hardware Requirements & Cost Analysis

### 3.1 Target Models Performance Requirements
Based on existing benchmarks from containers-plan.md:

**Llama 3.2-1B:**
- VRAM: 2-4GB minimum
- Target latency: <1s inference
- Recommended: T4, L4, or A10 class GPUs

**Mistral 7B:**
- VRAM: 8-12GB minimum  
- Target latency: <2s inference
- Recommended: A10, L40S, or A100 40GB

**Qwen 2.5-1.5B:**
- VRAM: 3-6GB minimum
- Target latency: <1s inference
- Recommended: T4, L4, or A10 class GPUs

### 3.2 Cost Per Inference Estimates

**Modal Pricing (assuming 2s average inference):**
- T4 (small models): $0.000328 per inference
- A10 (medium models): $0.000612 per inference
- A100 40GB (large models): $0.001166 per inference

**Fly.io GPU Pricing (assuming 2s average inference):**
- A10: $0.000833 per inference
- A100 40GB: $0.001389 per inference

**Cost Optimization Target:**
- 70% Golem baseline: ~$0.0001 per inference (estimated)
- 30% serverless burst: $0.0003-0.001 per inference
- **Blended cost target: <$0.0005 per inference**

## 4. Geographic Distribution Strategy

### 4.1 Recommended Regions
**US-East:** Virginia/Ohio (AWS equivalent)
- Modal: Available
- Fly.io: iad, ewr regions
- Railway: Available

**EU-West:** Ireland/Frankfurt 
- Modal: Available
- Fly.io: ams, fra, lhr regions  
- Railway: Available

**Asia-Pacific:** Singapore/Tokyo
- Modal: Available
- Fly.io: sin, nrt regions
- Railway: Available

### 4.2 Latency Considerations
- Target <100ms region-to-runner latency
- Golem network connectivity varies by region
- Serverless providers offer better geographic distribution

## 5. Recommendations

### 5.1 Primary Stack Recommendation
**Tier 1 (Baseline):** 2-3 Golem providers across regions
**Tier 2 (Burst):** Modal + RunPod for cost-effective serverless
**Tier 3 (Hosting):** Fly.io for multi-region container deployment

### 5.2 Cost Optimization Strategy
1. Route 70% of steady-state traffic to Golem providers
2. Use Modal for Python-native inference workloads (burst)
3. Use RunPod for cost-sensitive overflow capacity
4. Deploy coordination logic on Fly.io for global reach

### 5.3 Implementation Priority
1. **Week 1**: Modal integration (easiest Python deployment)
2. **Week 2**: Fly.io multi-region container hosting
3. **Week 3**: RunPod integration for cost optimization
4. **Week 4**: Cross-region routing and failover testing

## 6. Risk Assessment

### 6.1 Technical Risks
- **Cold start latency**: Mitigated by warm instance pools
- **Provider outages**: Addressed by multi-provider strategy
- **Cost overruns**: Controlled by routing algorithms and budgets

### 6.2 Operational Risks
- **Complexity**: Multiple provider APIs to manage
- **Monitoring**: Need unified observability across providers
- **Debugging**: Distributed system troubleshooting challenges

## 7. Next Steps (Phase 2)

### 7.1 Immediate Actions
- [ ] Set up Modal development account and test deployment
- [ ] Configure Fly.io multi-region container hosting
- [ ] Design hybrid routing logic in runner app
- [ ] Create unified monitoring dashboard

### 7.2 Success Metrics
- **Latency**: <2s p95 inference across all providers
- **Availability**: >99.5% uptime with failover
- **Cost**: <$0.0005 blended cost per inference
- **Scalability**: Handle 100+ concurrent jobs per region

---

**Report Prepared By**: AI Assistant  
**Next Review**: Phase 2 completion  
**Status**: Ready for Phase 2 implementation
