# Modal Deployment for Project Beacon

This directory contains Modal serverless GPU deployment for Project Beacon's LLM inference models.

## Setup

1. **Install Modal CLI:**
```bash
pip install modal
```

2. **Authenticate:**
```bash
modal setup
```

3. **Deploy the app:**
```bash
modal deploy modal_inference.py
```

## Models Supported

- **Llama 3.2-1B**: Fast inference, 2-4GB VRAM
- **Mistral 7B**: Medium models, 8-12GB VRAM  
- **Qwen 2.5-1.5B**: Efficient inference, 3-6GB VRAM

## GPU Configurations

- **T4**: Cost-effective for small models ($0.59/hour)
- **A10**: Balanced performance ($1.10/hour)
- **A100**: High-performance for large models ($2.10-2.50/hour)

## Usage

### Local Testing
```bash
python test_modal.py
```

### API Endpoints

**Inference API:**
```bash
curl -X POST https://your-app-id--inference-api.modal.run \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:1b",
    "prompt": "What is artificial intelligence?",
    "temperature": 0.1,
    "max_tokens": 100
  }'
```

**Health Check:**
```python
import modal
health = modal.Function.lookup("project-beacon-inference", "health_check")
result = health.remote()
```

## Functions

- `setup_models()`: Pre-load models into container
- `run_inference()`: Single inference request
- `run_batch_inference()`: Batch processing for multiple requests
- `health_check()`: Monitor service health
- `inference_api()`: HTTP endpoint for external access

## Performance Targets

- **Latency**: <2s p95 inference time
- **Throughput**: 5-10 concurrent requests per container
- **Cost**: $0.0003-0.001 per inference (2s average)

## Integration with Project Beacon Runner

The Modal deployment integrates with the Project Beacon runner app through:

1. **HTTP API**: Direct REST calls to inference endpoints
2. **Python SDK**: Direct function calls from runner
3. **Batch Processing**: Efficient handling of multiple jobs

## Monitoring

- Container idle timeout: 5-10 minutes for warm instances
- Health checks every 30 seconds
- Cost tracking per inference request
- Performance metrics (latency, throughput, error rates)

## Next Steps

1. Test deployment with Project Beacon models
2. Integrate with runner app routing logic
3. Set up monitoring and alerting
4. Configure multi-region deployment
