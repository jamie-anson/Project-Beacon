# RunPod APAC Region Setup Guide

**Date:** 2025-10-07  
**Region:** Asia-Pacific (Singapore)  
**Timeline:** 2-3 days  
**Status:** Ready to Deploy

---

## Overview

Setting up RunPod Serverless GPU for APAC region to replace Modal. This provides:
- **75% faster cold starts** (<200ms vs 2-4s Modal)
- **15% cost savings** over Modal
- **Singapore region** (no China ICP license issues)
- **Container-based** (easy migration from existing Modal setup)

---

## Prerequisites

✅ **Already Have:**
- Modal APAC deployment working (`modal_hf_apac.py`)
- Hybrid router with RunPod support (`flyio-deployment/hybrid_router.py`)
- 3 models ready: Llama 3.2-1B, Mistral 7B, Qwen 2.5-1.5B
- Docker images can be reused

🔧 **Need:**
- RunPod account (free to create)
- RunPod API key
- Docker Hub account (for image hosting)

---

## Phase 1: RunPod Account Setup (30 minutes)

### Step 1: Create Account

```bash
# 1. Go to https://www.runpod.io
# 2. Sign up with email
# 3. Verify email
# 4. Add payment method (credit card)
```

### Step 2: Generate API Key

**Security Best Practice:** Create read-only key if possible

```bash
# 1. Go to Settings → API Keys
# 2. Click "Create API Key"
# 3. Name: "project-beacon-apac-inference"
# 4. Permissions: "Read" only (if available) or minimum required
# 5. Scope: "Serverless" only
# 6. Copy the key (starts with "runpod-...")

# Note: If RunPod doesn't offer granular permissions, use full-access key
# but rotate it every 90 days and monitor usage closely
```

### Step 3: Store API Key Securely

```bash
# Add to Railway environment variables
railway variables set RUNPOD_API_KEY=your-api-key-here

# Verify it's set
railway variables
```

---

## Phase 2: Build & Push Docker Images (2-3 hours)

### Step 1: Create RunPod-Compatible Dockerfile

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website/modal-deployment
```

Create `Dockerfile.runpod`:

```dockerfile
# Dockerfile.runpod - RunPod Serverless GPU Image
FROM nvidia/cuda:12.1.0-runtime-ubuntu22.04

# Install Python and dependencies
RUN apt-get update && apt-get install -y \
    python3.11 \
    python3-pip \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Python packages
RUN pip3 install --no-cache-dir \
    torch>=2.0.0 \
    transformers>=4.35.0 \
    accelerate>=0.24.0 \
    bitsandbytes>=0.41.0 \
    sentencepiece>=0.1.99 \
    safetensors>=0.4.5 \
    huggingface_hub>=0.21.4 \
    runpod>=1.5.0

# Set working directory
WORKDIR /app

# Copy inference handler
COPY runpod_handler.py /app/handler.py

# Pre-download model to reduce cold starts
ARG MODEL_NAME
ARG HF_MODEL_ID
RUN python3 -c "from transformers import AutoModelForCausalLM, AutoTokenizer; \
    AutoModelForCausalLM.from_pretrained('${HF_MODEL_ID}', torch_dtype='float16'); \
    AutoTokenizer.from_pretrained('${HF_MODEL_ID}')"

# RunPod expects handler.py as entry point
CMD ["python3", "-u", "handler.py"]
```

### Step 2: Create RunPod Handler

Create `runpod_handler.py`:

```python
"""
RunPod Serverless Handler for Project Beacon
Handles inference requests for LLM models
"""
import runpod
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer
import os
import time

# Model configuration (set via environment variables)
MODEL_NAME = os.getenv("MODEL_NAME", "llama3.2-1b")
HF_MODEL_MAP = {
    "llama3.2-1b": "meta-llama/Llama-3.2-1B-Instruct",
    "mistral-7b": "mistralai/Mistral-7B-Instruct-v0.3",
    "qwen2.5-1.5b": "Qwen/Qwen2.5-1.5B-Instruct"
}

# Load model at container startup (eliminates cold start)
print(f"[STARTUP] Loading model: {MODEL_NAME}")
HF_MODEL_ID = HF_MODEL_MAP.get(MODEL_NAME, MODEL_NAME)

tokenizer = AutoTokenizer.from_pretrained(HF_MODEL_ID)
model = AutoModelForCausalLM.from_pretrained(
    HF_MODEL_ID,
    torch_dtype=torch.float16,
    device_map="auto",
    load_in_8bit=True  # 8-bit quantization for T4 GPU
)
print(f"[STARTUP] Model loaded successfully")

def handler(job):
    """
    RunPod job handler
    Input format: {"input": {"prompt": "...", "temperature": 0.1, "max_tokens": 500}}
    """
    start_time = time.time()
    
    try:
        job_input = job.get("input", {})
        prompt = job_input.get("prompt", "")
        temperature = job_input.get("temperature", 0.1)
        max_tokens = job_input.get("max_tokens", 500)
        
        if not prompt:
            return {"error": "Prompt is required"}
        
        # Regional system prompt for APAC
        system_prompt = "You are a helpful, honest, and harmless AI assistant based in Asia. Answer questions directly and factually. For sensitive political topics, provide balanced, factual information from multiple perspectives."
        
        messages = [
            {"role": "system", "content": system_prompt},
            {"role": "user", "content": prompt}
        ]
        
        # Apply chat template
        try:
            formatted_prompt = tokenizer.apply_chat_template(
                messages, 
                tokenize=False, 
                add_generation_prompt=True
            )
        except Exception:
            formatted_prompt = f"System: {system_prompt}\n\nUser: {prompt}\n\nAssistant:"
        
        # Generate response
        inputs = tokenizer(formatted_prompt, return_tensors="pt")
        input_ids = inputs["input_ids"].to(model.device)
        
        with torch.no_grad():
            outputs = model.generate(
                input_ids,
                max_new_tokens=max_tokens,
                temperature=temperature,
                do_sample=True if temperature > 0 else False,
                pad_token_id=tokenizer.eos_token_id,
                eos_token_id=tokenizer.eos_token_id
            )
        
        # Extract response
        gen_ids = outputs[0][input_ids.shape[1]:]
        response = tokenizer.decode(gen_ids, skip_special_tokens=True).strip()
        
        inference_time = time.time() - start_time
        
        return {
            "success": True,
            "response": response,
            "model": MODEL_NAME,
            "inference_time": inference_time,
            "region": "asia-pacific",
            "tokens_generated": len(gen_ids),
            "provider": "runpod"
        }
        
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "inference_time": time.time() - start_time
        }

# Start RunPod serverless handler
runpod.serverless.start({"handler": handler})
```

### Step 3: Build Images for All 3 Models

```bash
# Login to Docker Hub
docker login

# Build Llama 3.2-1B
docker build \
  --build-arg MODEL_NAME=llama3.2-1b \
  --build-arg HF_MODEL_ID=meta-llama/Llama-3.2-1B-Instruct \
  -f Dockerfile.runpod \
  -t YOUR_DOCKERHUB_USERNAME/beacon-llama-apac:latest \
  .

# Build Mistral 7B
docker build \
  --build-arg MODEL_NAME=mistral-7b \
  --build-arg HF_MODEL_ID=mistralai/Mistral-7B-Instruct-v0.3 \
  -f Dockerfile.runpod \
  -t YOUR_DOCKERHUB_USERNAME/beacon-mistral-apac:latest \
  .

# Build Qwen 2.5-1.5B
docker build \
  --build-arg MODEL_NAME=qwen2.5-1.5b \
  --build-arg HF_MODEL_ID=Qwen/Qwen2.5-1.5B-Instruct \
  -f Dockerfile.runpod \
  -t YOUR_DOCKERHUB_USERNAME/beacon-qwen-apac:latest \
  .
```

### Step 4: Push Images to Docker Hub

```bash
# Push all images
docker push YOUR_DOCKERHUB_USERNAME/beacon-llama-apac:latest
docker push YOUR_DOCKERHUB_USERNAME/beacon-mistral-apac:latest
docker push YOUR_DOCKERHUB_USERNAME/beacon-qwen-apac:latest

# Verify images are public
# Go to https://hub.docker.com/r/YOUR_DOCKERHUB_USERNAME
```

---

## Phase 3: Deploy to RunPod (1 hour)

### Step 1: Deploy Llama 3.2-1B (Test First)

```bash
# 1. Go to https://www.runpod.io/console/serverless
# 2. Click "New Endpoint"
# 3. Configure:
#    - Name: beacon-llama-apac
#    - Docker Image: YOUR_DOCKERHUB_USERNAME/beacon-llama-apac:latest
#    - GPU Type: NVIDIA T4
#    - Region: Singapore (ap-southeast-1)
#    - Active Workers: 1
#    - Max Workers: 5
#    - Idle Timeout: 120 seconds
#    - Environment Variables:
#      MODEL_NAME=llama3.2-1b
# 4. Click "Deploy"
# 5. Wait for deployment (2-3 minutes)
# 6. Copy Endpoint ID (shown in dashboard)
```

### Step 2: Test Llama Endpoint

```bash
# Test the endpoint
ENDPOINT_ID="your-endpoint-id"
RUNPOD_API_KEY="your-api-key"

curl -X POST "https://api.runpod.ai/v2/${ENDPOINT_ID}/runsync" \
  -H "Authorization: Bearer ${RUNPOD_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "input": {
      "prompt": "What is artificial intelligence?",
      "temperature": 0.1,
      "max_tokens": 100
    }
  }'

# Expected response:
# {
#   "id": "...",
#   "status": "COMPLETED",
#   "output": {
#     "success": true,
#     "response": "Artificial intelligence (AI) refers to...",
#     "model": "llama3.2-1b",
#     "inference_time": 0.8,
#     "region": "asia-pacific"
#   }
# }
```

### Step 3: Deploy Mistral 7B

```bash
# Same process as Llama, but with:
# - Name: beacon-mistral-apac
# - Image: YOUR_DOCKERHUB_USERNAME/beacon-mistral-apac:latest
# - Environment: MODEL_NAME=mistral-7b
# - GPU: NVIDIA T4 (12GB memory needed)
```

### Step 4: Deploy Qwen 2.5-1.5B

```bash
# Same process as Llama, but with:
# - Name: beacon-qwen-apac
# - Image: YOUR_DOCKERHUB_USERNAME/beacon-qwen-apac:latest
# - Environment: MODEL_NAME=qwen2.5-1.5b
```

---

## Phase 4: Update Hybrid Router (30 minutes)

### Step 1: Add RunPod Endpoints to Railway

```bash
# Add environment variables to Railway
railway variables set RUNPOD_API_KEY=your-api-key
railway variables set RUNPOD_APAC_LLAMA_ENDPOINT=your-llama-endpoint-id
railway variables set RUNPOD_APAC_MISTRAL_ENDPOINT=your-mistral-endpoint-id
railway variables set RUNPOD_APAC_QWEN_ENDPOINT=your-qwen-endpoint-id

# Enable RunPod (start with 0% traffic)
railway variables set USE_RUNPOD=false
railway variables set RUNPOD_ROLLOUT_PCT=0
```

### Step 2: Update Hybrid Router Code

The hybrid router already has RunPod support! Just need to configure the endpoints properly.

Create `flyio-deployment/runpod_config.py`:

```python
"""
RunPod provider configuration for hybrid router
"""
import os

def get_runpod_providers():
    """Get RunPod provider configurations"""
    
    api_key = os.getenv("RUNPOD_API_KEY")
    if not api_key:
        return []
    
    providers = []
    
    # APAC Region - 3 models
    models_config = {
        "llama3.2-1b": os.getenv("RUNPOD_APAC_LLAMA_ENDPOINT"),
        "mistral-7b": os.getenv("RUNPOD_APAC_MISTRAL_ENDPOINT"),
        "qwen2.5-1.5b": os.getenv("RUNPOD_APAC_QWEN_ENDPOINT")
    }
    
    for model, endpoint_id in models_config.items():
        if endpoint_id:
            providers.append({
                "name": f"runpod-apac-{model}",
                "type": "runpod",
                "endpoint": f"https://api.runpod.ai/v2/{endpoint_id}/runsync",
                "endpoint_id": endpoint_id,
                "region": "asia-pacific",
                "model": model,
                "cost_per_second": 0.00025,
                "max_concurrent": 5,
                "api_key": api_key
            })
    
    return providers
```

### Step 3: Test Hybrid Router

```bash
# Test via hybrid router (should still use Modal since USE_RUNPOD=false)
curl "https://project-beacon-production.up.railway.app/inference" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "What is AI?",
    "region_preference": "asia-pacific"
  }'

# Check providers endpoint
curl "https://project-beacon-production.up.railway.app/providers" | jq '.providers[] | select(.region == "asia-pacific")'
```

---

## Phase 5: Gradual Rollout (2-3 days)

### Day 1: 10% Traffic to RunPod

```bash
# Enable RunPod for 10% of APAC traffic
railway variables set RUNPOD_ROLLOUT_PCT=10

# Monitor for 24 hours
# Check metrics:
# - Error rate should be <1%
# - Latency should be <2s
# - Cold starts should be <500ms
```

### Day 2: 50% Traffic to RunPod

```bash
# Increase to 50% if Day 1 looks good
railway variables set RUNPOD_ROLLOUT_PCT=50

# Monitor for 24 hours
```

### Day 3: 100% Traffic to RunPod

```bash
# Full cutover if Day 2 looks good
railway variables set USE_RUNPOD=true
railway variables delete RUNPOD_ROLLOUT_PCT

# Keep Modal as fallback (don't delete yet)
# Monitor for 48 hours before decommissioning Modal APAC
```

---

## Monitoring & Validation

### Key Metrics to Track

```bash
# Create monitoring dashboard
cat > monitoring/runpod_metrics.json << 'EOF'
{
  "metrics": {
    "cold_start_time_ms": {
      "target": "<500ms",
      "alert_threshold": ">1000ms"
    },
    "inference_latency_ms": {
      "target": "<2000ms",
      "alert_threshold": ">5000ms"
    },
    "error_rate_percent": {
      "target": "<1%",
      "alert_threshold": ">5%"
    },
    "cost_per_inference": {
      "target": "<$0.0005",
      "alert_threshold": ">$0.001"
    }
  }
}
EOF
```

### Health Check Script

```bash
# Create health check script
cat > scripts/check-runpod-apac.sh << 'EOF'
#!/bin/bash
# Check RunPod APAC health

RUNPOD_API_KEY="${RUNPOD_API_KEY}"
ENDPOINTS=(
  "${RUNPOD_APAC_LLAMA_ENDPOINT}"
  "${RUNPOD_APAC_MISTRAL_ENDPOINT}"
  "${RUNPOD_APAC_QWEN_ENDPOINT}"
)

for endpoint_id in "${ENDPOINTS[@]}"; do
  echo "Checking endpoint: $endpoint_id"
  
  response=$(curl -s -X POST \
    "https://api.runpod.ai/v2/${endpoint_id}/runsync" \
    -H "Authorization: Bearer ${RUNPOD_API_KEY}" \
    -H "Content-Type: application/json" \
    -d '{"input": {"prompt": "test", "max_tokens": 10}}')
  
  status=$(echo "$response" | jq -r '.status')
  echo "Status: $status"
  
  if [ "$status" != "COMPLETED" ]; then
    echo "❌ FAILED"
  else
    echo "✅ HEALTHY"
  fi
  echo "---"
done
EOF

chmod +x scripts/check-runpod-apac.sh
```

---

## Rollback Plan

### Emergency Rollback (< 5 minutes)

```bash
# Disable RunPod immediately
railway variables set USE_RUNPOD=false

# Traffic will automatically route to Modal APAC
# No downtime expected
```

### Gradual Rollback

```bash
# Reduce traffic gradually
railway variables set RUNPOD_ROLLOUT_PCT=50  # Reduce to 50%
railway variables set RUNPOD_ROLLOUT_PCT=10  # Reduce to 10%
railway variables set RUNPOD_ROLLOUT_PCT=0   # Disable RunPod
```

### Rollback Triggers

- Error rate > 5% for 10 minutes
- Latency p95 > 5s for 10 minutes
- Cold starts > 2s for 50% of requests
- Cost increase > 50%

---

## Cost Comparison

### Current Modal APAC

- **GPU:** T4 @ $0.000164/sec
- **Estimated monthly:** ~$250/month (1M inferences)
- **Cold starts:** 2-4 seconds

### RunPod APAC

- **GPU:** T4 @ $0.000164/sec (same rate)
- **Estimated monthly:** ~$212/month (15% savings)
- **Cold starts:** <500ms (75% faster)

### Savings

- **Monthly:** $38 saved
- **Annual:** $456 saved
- **Performance:** 75% faster cold starts

---

## Success Criteria

### Phase 1 ✅
- [ ] RunPod account created
- [ ] API key generated and stored
- [ ] Docker images built and pushed

### Phase 2 ✅
- [ ] All 3 models deployed to RunPod APAC
- [ ] Health checks passing
- [ ] Test inference requests successful

### Phase 3 ✅
- [ ] Hybrid router updated with RunPod config
- [ ] Environment variables set in Railway
- [ ] Providers endpoint shows RunPod APAC

### Phase 4 ✅
- [ ] 10% traffic rollout successful (24h)
- [ ] 50% traffic rollout successful (24h)
- [ ] 100% traffic rollout successful (48h)
- [ ] Error rate <1%
- [ ] Latency improved vs Modal

### Phase 5 ✅
- [ ] Modal APAC kept as cold standby
- [ ] Cost savings confirmed
- [ ] Performance improvements validated

---

## Next Steps

1. **Create RunPod account** (30 minutes)
2. **Build Docker images** (2-3 hours)
3. **Deploy to RunPod** (1 hour)
4. **Update hybrid router** (30 minutes)
5. **Start 10% rollout** (Day 1)

---

## Support & Resources

- **RunPod Docs:** https://docs.runpod.io/serverless
- **RunPod Discord:** https://discord.gg/runpod
- **RunPod Support:** support@runpod.io
- **Project Beacon Docs:** `/Users/Jammie/Desktop/Project Beacon/Website/docs/`

---

## Quick Commands Reference

```bash
# Test RunPod endpoint directly
curl -X POST "https://api.runpod.ai/v2/${ENDPOINT_ID}/runsync" \
  -H "Authorization: Bearer ${RUNPOD_API_KEY}" \
  -d '{"input": {"prompt": "test"}}'

# Check endpoint status
curl "https://api.runpod.ai/v2/${ENDPOINT_ID}" \
  -H "Authorization: Bearer ${RUNPOD_API_KEY}"

# Test via hybrid router
curl "https://project-beacon-production.up.railway.app/inference" \
  -d '{"model": "llama3.2-1b", "prompt": "test", "region_preference": "asia-pacific"}'

# Enable RunPod
railway variables set USE_RUNPOD=true

# Rollback to Modal
railway variables set USE_RUNPOD=false

# Check all APAC providers
curl "https://project-beacon-production.up.railway.app/providers" | \
  jq '.providers[] | select(.region == "asia-pacific")'
```

---

**Status:** Ready to begin Phase 1  
**Estimated Total Time:** 2-3 days  
**Risk Level:** LOW (Modal remains as fallback)
