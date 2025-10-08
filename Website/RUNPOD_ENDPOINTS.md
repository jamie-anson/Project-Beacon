# RunPod APAC Endpoints

## Deployed Endpoints

### ✅ Qwen 2.5-1.5B
- **Endpoint ID:** `nd7eqzpfnbwvsy`
- **Image:** `freelancejamie/beacon-qwen-apac:latest`
- **Model:** `qwen2.5-1.5b`
- **GPU:** 24GB High Supply PRO
- **Status:** Deployed
- **URL:** `https://api.runpod.ai/v2/nd7eqzpfnbwvsy/runsync`

### ⏳ Mistral 7B
- **Status:** Not deployed yet
- **Image:** `freelancejamie/beacon-mistral-apac:latest`
- **Model:** `mistral-7b`

### ⏳ Llama 3.2-1B
- **Status:** Not deployed yet (needs HF token)
- **Image:** `freelancejamie/beacon-llama-apac:latest`
- **Model:** `llama3.2-1b`

---

## Railway Configuration

Add these to Railway environment variables:

```bash
# Qwen endpoint (deployed)
railway variables set RUNPOD_APAC_QWEN_ENDPOINT="nd7eqzpfnbwvsy"

# Mistral endpoint (deploy next)
railway variables set RUNPOD_APAC_MISTRAL_ENDPOINT="your-mistral-endpoint-id"

# Llama endpoint (deploy last)
railway variables set RUNPOD_APAC_LLAMA_ENDPOINT="your-llama-endpoint-id"
```

---

## Next Steps

1. ✅ Qwen deployed - Add to Railway
2. 🔄 Deploy Mistral endpoint
3. 🔄 Deploy Llama endpoint (with HF token)
4. 🧪 Test all endpoints
5. 🚀 Start gradual rollout (10% → 50% → 100%)
