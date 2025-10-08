# RunPod APAC Quick Start

**Goal:** Replace Modal with RunPod for APAC region only (75% faster cold starts, 15% cost savings)

**Note:** US and EU regions will remain on Modal for now. This is APAC-only deployment.

---

## Prerequisites Checklist

- [ ] RunPod account created at https://www.runpod.io
- [ ] RunPod API key generated (Settings → API Keys)
- [ ] Docker Hub account (for hosting images)
- [ ] Docker Desktop running locally

---

## Step 1: Build & Push Images (2-3 hours)

```bash
# Set your Docker Hub username
export DOCKERHUB_USERNAME="your-dockerhub-username"

# Build and push all 3 models
cd /Users/Jammie/Desktop/Project\ Beacon/Website
./scripts/build-runpod-images.sh

# This will:
# - Build Llama 3.2-1B, Mistral 7B, Qwen 2.5-1.5B
# - Push to Docker Hub
# - Take 2-3 hours (downloading models)
```

---

## Step 2: Deploy to RunPod (30 minutes)

### Deploy Llama 3.2-1B

1. Go to https://www.runpod.io/console/serverless
2. Click **"New Endpoint"**
3. Configure:
   - **Name:** `beacon-llama-apac`
   - **Docker Image:** `YOUR_USERNAME/beacon-llama-apac:latest`
   - **GPU Type:** NVIDIA T4
   - **Region:** Singapore (ap-southeast-1)
   - **Active Workers:** 1
   - **Max Workers:** 5
   - **Idle Timeout:** 120 seconds
   - **Environment Variables:**
     - `MODEL_NAME=llama3.2-1b`
4. Click **"Deploy"**
5. Copy the **Endpoint ID** (e.g., `abc123xyz`)

### Deploy Mistral 7B

Repeat above with:
- **Name:** `beacon-mistral-apac`
- **Image:** `YOUR_USERNAME/beacon-mistral-apac:latest`
- **Environment:** `MODEL_NAME=mistral-7b`

### Deploy Qwen 2.5-1.5B

Repeat above with:
- **Name:** `beacon-qwen-apac`
- **Image:** `YOUR_USERNAME/beacon-qwen-apac:latest`
- **Environment:** `MODEL_NAME=qwen2.5-1.5b`

---

## Step 3: Test Endpoints (10 minutes)

```bash
# Set your credentials
export RUNPOD_API_KEY="your-api-key"
export RUNPOD_APAC_LLAMA_ENDPOINT="llama-endpoint-id"
export RUNPOD_APAC_MISTRAL_ENDPOINT="mistral-endpoint-id"
export RUNPOD_APAC_QWEN_ENDPOINT="qwen-endpoint-id"

# Run tests
./scripts/test-runpod-apac.sh

# Expected output:
# ✅ SUCCESS - Llama 3.2-1B
# ✅ SUCCESS - Mistral 7B
# ✅ SUCCESS - Qwen 2.5-1.5B
```

---

## Step 4: Update Railway (5 minutes)

```bash
# Add environment variables to Railway
railway variables set RUNPOD_API_KEY="your-api-key"
railway variables set RUNPOD_APAC_LLAMA_ENDPOINT="llama-endpoint-id"
railway variables set RUNPOD_APAC_MISTRAL_ENDPOINT="mistral-endpoint-id"
railway variables set RUNPOD_APAC_QWEN_ENDPOINT="qwen-endpoint-id"

# Start with 0% traffic (testing only)
railway variables set USE_RUNPOD=false
railway variables set RUNPOD_ROLLOUT_PCT=0
```

---

## Step 5: Gradual Rollout (2-3 days)

### Day 1: 10% Traffic

```bash
railway variables set RUNPOD_ROLLOUT_PCT=10
# Monitor for 24 hours
```

### Day 2: 50% Traffic

```bash
railway variables set RUNPOD_ROLLOUT_PCT=50
# Monitor for 24 hours
```

### Day 3: 100% Traffic

```bash
railway variables set USE_RUNPOD=true
railway variables delete RUNPOD_ROLLOUT_PCT
# Monitor for 48 hours
```

---

## Monitoring

### Check Provider Status

```bash
# Check all APAC providers
curl "https://project-beacon-production.up.railway.app/providers" | \
  jq '.providers[] | select(.region == "asia-pacific")'
```

### Test Inference via Hybrid Router

```bash
# Test Llama 3.2-1B
curl "https://project-beacon-production.up.railway.app/inference" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "What is AI?",
    "region_preference": "asia-pacific"
  }'
```

---

## Emergency Rollback

```bash
# Disable RunPod immediately (< 5 minutes)
railway variables set USE_RUNPOD=false

# Traffic automatically routes back to Modal
```

---

## Cost Savings

- **Current Modal APAC:** ~$250/month
- **RunPod APAC:** ~$212/month
- **Savings:** $38/month ($456/year)
- **Performance:** 75% faster cold starts (<500ms vs 2-4s)

---

## Support

- **Full Guide:** `RUNPOD_APAC_SETUP.md`
- **Migration Plan:** `RUNPOD_MIGRATION_PLAN.md`
- **RunPod Docs:** https://docs.runpod.io/serverless
- **RunPod Discord:** https://discord.gg/runpod

---

## Files Created

✅ `modal-deployment/Dockerfile.runpod` - Docker image definition  
✅ `modal-deployment/runpod_handler.py` - Inference handler  
✅ `scripts/build-runpod-images.sh` - Build automation  
✅ `scripts/test-runpod-apac.sh` - Testing script  
✅ `RUNPOD_APAC_SETUP.md` - Detailed setup guide  
✅ `RUNPOD_QUICKSTART.md` - This file

---

**Ready to start? Run:** `./scripts/build-runpod-images.sh`
