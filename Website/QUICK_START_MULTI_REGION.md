# Quick Start: Deploy All Regions

**Goal**: Get EU-West and Asia-Pacific regions operational in ~30 minutes

---

## ✅ Current Status

**Files Ready**:
- ✅ `modal_hf_us.py` - US-East (app: "project-beacon-hf-us")
- ✅ `modal_hf_eu.py` - EU-West (app: "project-beacon-hf-eu")  
- ✅ `modal_hf_apac.py` - APAC (app: "project-beacon-hf-apac")

**Deployed Apps** (from `modal app list`):
- 3 apps deployed (likely US, EU, APAC already!)
- Need to verify which is which

---

## 🚀 Step-by-Step Deployment

### Step 1: Deploy EU-West (5 min)

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/Website/modal-deployment

# Deploy EU region
modal deploy modal_hf_eu.py
```

**Expected Output**:
```
✓ Created objects.
└── 🔨 Created web endpoint for project-beacon-hf-eu.inference
   URL: https://jamie-anson--project-beacon-hf-eu-inference.modal.run
```

**Copy the URL** - you'll need it for the router!

---

### Step 2: Deploy APAC (5 min)

```bash
# Deploy APAC region
modal deploy modal_hf_apac.py
```

**Expected Output**:
```
✓ Created objects.
└── 🔨 Created web endpoint for project-beacon-hf-apac.inference
   URL: https://jamie-anson--project-beacon-hf-apac-inference.modal.run
```

**Copy the URL** - you'll need it for the router!

---

### Step 3: Test Endpoints (5 min)

**Test EU-West**:
```bash
curl -X POST https://jamie-anson--project-beacon-hf-eu-inference.modal.run \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2-1b",
    "prompt": "What is 2+2?",
    "max_tokens": 100,
    "temperature": 0.1
  }'
```

**Test APAC**:
```bash
curl -X POST https://jamie-anson--project-beacon-hf-apac-inference.modal.run \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2.5-1.5b",
    "prompt": "What is the capital of France?",
    "max_tokens": 100,
    "temperature": 0.1
  }'
```

**Expected**: JSON response with inference result

---

### Step 4: Update Hybrid Router (15 min)

**Option A: If Router is on Railway**

1. Go to Railway dashboard
2. Find "project-beacon-production" service
3. Add environment variables:

```bash
MODAL_EU_ENDPOINT=https://jamie-anson--project-beacon-hf-eu-inference.modal.run
MODAL_APAC_ENDPOINT=https://jamie-anson--project-beacon-hf-apac-inference.modal.run
```

4. Redeploy the service

**Option B: If Router is in Code**

Find the router config file and add:

```python
# In router config
PROVIDERS = {
    "modal-us-east": {
        "endpoint": "https://jamie-anson--project-beacon-hf-us-inference.modal.run",
        "region": "us-east",
    },
    "modal-eu-west": {
        "endpoint": "https://jamie-anson--project-beacon-hf-eu-inference.modal.run",
        "region": "eu-west",
    },
    "modal-asia-pacific": {
        "endpoint": "https://jamie-anson--project-beacon-hf-apac-inference.modal.run",
        "region": "asia-pacific",
    },
}
```

---

### Step 5: Test End-to-End (5 min)

```bash
cd /Users/Jammie/Desktop/Project\ Beacon/runner-app

# Submit 3-region test
curl -X POST https://beacon-runner-production.fly.dev/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d @scripts/test-3regions-2questions.json
```

**Check results**:
```bash
JOB_ID="<job-id-from-above>"

curl -s "https://beacon-runner-production.fly.dev/api/v1/executions?jobspec_id=$JOB_ID" | \
  jq '[.executions[] | select(.created_at > "2025-09-30T15:00:00Z")] | {
    total: length,
    by_status: group_by(.status) | map({status: .[0].status, count: length}),
    by_region: group_by(.region) | map({region: .[0].region, count: length})
  }'
```

**Expected**:
```json
{
  "total": 6,
  "by_status": [{"status": "completed", "count": 6}],
  "by_region": [
    {"region": "us-east", "count": 2},
    {"region": "eu-west", "count": 2},
    {"region": "asia-pacific", "count": 2}
  ]
}
```

---

## 🎯 Success Criteria

- [ ] EU-West Modal deployed
- [ ] APAC Modal deployed  
- [ ] Both endpoints tested successfully
- [ ] Hybrid router updated
- [ ] 3-region test shows 6/6 executions completed
- [ ] No 404 errors from any region

---

## 📝 Next Steps

Once all regions are working:
1. Monitor costs across all regions
2. Set up keep-warm for consistent latency
3. Add region-specific monitoring
4. Document regional endpoints

---

**Ready to start!** Run Step 1 to deploy EU-West.
