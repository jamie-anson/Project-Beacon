# Fly.io Hybrid Router Deployment

This directory contains the Fly.io deployment for Project Beacon's hybrid routing service that intelligently routes inference requests between Golem providers and serverless GPU providers.

## Architecture

The hybrid router acts as a smart load balancer that:
- Routes 70% of traffic to cost-effective Golem providers (baseline)
- Routes 30% to serverless GPU providers for burst capacity
- Provides automatic failover and health monitoring
- Optimizes for cost or performance based on request parameters

## Setup

1. **Install Fly.io CLI:**
```bash
curl -L https://fly.io/install.sh | sh
```

2. **Authenticate:**
```bash
flyctl auth login
```

3. **Deploy:**
```bash
./deploy.sh
```

4. **Set secrets:**
```bash
flyctl secrets set MODAL_API_TOKEN=your_modal_token -a beacon-hybrid-router
flyctl secrets set RUNPOD_API_KEY=your_runpod_key -a beacon-hybrid-router
flyctl secrets set GOLEM_PROVIDER_ENDPOINTS=endpoint1,endpoint2 -a beacon-hybrid-router
```

## Multi-Region Deployment

The router automatically deploys to 3 regions:
- **US-East** (iad): Primary region
- **EU-West** (ams): European traffic
- **Asia-Pacific** (sin): Asian traffic

## API Endpoints

### Health Check
```bash
curl https://beacon-hybrid-router.fly.dev/health
```

### Inference
```bash
curl -X POST https://beacon-hybrid-router.fly.dev/inference \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:1b",
    "prompt": "What is artificial intelligence?",
    "temperature": 0.1,
    "max_tokens": 100,
    "region_preference": "us-east",
    "cost_priority": true
  }'
```

### Provider Status
```bash
curl https://beacon-hybrid-router.fly.dev/providers
```

### Metrics
```bash
curl https://beacon-hybrid-router.fly.dev/metrics
```

## Provider Types

### Golem Providers
- **Cost**: ~$0.0001/second (lowest cost)
- **Use case**: Steady-state baseline capacity
- **Regions**: Configured via GOLEM_PROVIDER_ENDPOINTS

### Modal Serverless
- **Cost**: ~$0.0003/second (T4 GPU)
- **Use case**: Python-native inference, burst capacity
- **Features**: Sub-second cold starts, dynamic batching

### RunPod Serverless
- **Cost**: ~$0.00025/second (15% savings claimed)
- **Use case**: Cost-optimized burst capacity
- **Features**: GPU-optimized functions, warm pools

## Routing Logic

1. **Health Filtering**: Only route to healthy providers
2. **Region Preference**: Honor region_preference if specified
3. **Cost Priority**: 
   - `cost_priority=true`: Golem → RunPod → Modal
   - `cost_priority=false`: Lowest latency first
4. **Capacity Check**: Ensure provider has available capacity
5. **Failover**: Automatic retry on different provider if failure

## Monitoring

- Health checks every 30 seconds
- Provider metrics tracking (latency, success rate)
- Cost estimation per request
- Regional performance monitoring

## Testing

```bash
# Local testing
python test_hybrid_router.py

# Load testing
python -m pytest test_load.py
```

## Integration with Project Beacon Runner

The hybrid router integrates with the main runner app:

1. **Replace direct provider calls** with router API calls
2. **Unified interface** for all GPU providers
3. **Automatic cost optimization** and failover
4. **Centralized monitoring** and metrics

## Cost Optimization

Target cost breakdown:
- 70% Golem baseline: $0.0001/sec × 0.7 = $0.00007/sec
- 30% serverless burst: $0.0003/sec × 0.3 = $0.00009/sec
- **Total blended cost**: ~$0.00016/sec per inference

## Scaling

- Auto-scaling based on request volume
- Multi-region deployment for global reach
- Provider capacity monitoring and alerts
- Cost threshold enforcement
