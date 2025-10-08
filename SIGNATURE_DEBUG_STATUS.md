# Signature Debug Status

**Time:** 2025-10-08T19:13:58+01:00  
**Status:** Deployments in progress, waiting for server logs

---

## Progress So Far

### Fix #1: Remove `id` field ✅
- **Portal:** Added `delete signable.id`
- **Server:** Already had `delete(m, "id")`
- **Result:** Still failing

### Fix #2: Remove `created_at` field ⏳
- **Portal:** Added `delete signable.created_at`
- **Server:** Added `delete(m, "created_at")` and `CreatedAt` field zeroing
- **Deployed:** Portal (Netlify), Runner (Fly.io in progress)
- **Result:** Waiting for deployment

---

## Latest Portal Canonical JSON

```json
{
  "benchmark": {
    "container": {
      "image": "ghcr.io/project-beacon/bias-detection:latest",
      "resources": {"cpu": "1000m", "memory": "2Gi"},
      "tag": "latest"
    },
    "description": "Multi-model bias detection across 3 models",
    "input": {
      "data": {"prompt": "greatest_invention"},
      "hash": "sha256:placeholder",
      "type": "prompt"
    },
    "name": "multi-model-bias-detection",
    "version": "v1"
  },
  "constraints": {
    "min_regions": 1,
    "provider_timeout": 600000000000,
    "regions": ["US", "EU"],
    "timeout": 600000000000
  },
  "metadata": {
    "created_by": "portal",
    "estimated_cost": "0.0012",
    "execution_type": "cross-region",
    "model": "llama3.2-1b",
    "model_name": "Llama 3.2-1B",
    "models": ["llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"],
    "multi_model": true,
    "nonce": "PtuLTDHlXFsC55DbchurNA",
    "timestamp": "2025-10-08T18:13:36.324Z",
    "total_executions_expected": 12,
    "wallet_address": "0x67f3d16a91991cf169920f1e79f78e66708da328"
  },
  "questions": ["greatest_invention", "greatest_leader"],
  "runs": 1,
  "version": "v1",
  "wallet_auth": {
    "address": "0x67f3d16a91991cf169920f1e79f78e66708da328",
    "chainId": 1,
    "expiresAt": "2025-10-15T18:13:32.330Z",
    "message": "Authorize Project Beacon key: qBP01SQs+cJ57B1sBjypEotITySqNR9EF3qK412l6Tk=",
    "nonce": "MQHhHqMMcDfxtdW3/XEBOg",
    "signature": "0x9c27e8c03d06686ae70866cd2fadd16a4543d03db8e3a2c077a2f6d3423e03a053ba2bba280552401bf34e60fd6d8641d7ba8507511187b9e96395c7ea376f6d1b"
  }
}
```

**Stats:**
- Length: 1300 characters
- SHA256: `f1b1811b3fad2e3496d7b605ef5e70b72fa1996d495e3593f05d58302d54b0ca`
- Fields excluded: `id`, `created_at`, `signature`, `public_key`

---

## Next Steps

1. **Wait for Fly.io deployment** (~5 min)
2. **Submit test job** after deployment
3. **Check server logs** for canonical JSON comparison:
   ```bash
   fly logs -a beacon-runner-production | grep "SIGNATURE DEBUG"
   ```
4. **Compare** portal vs server canonical JSON character-by-character

---

## Possible Remaining Issues

If signature still fails after `created_at` fix:

### 1. Number vs String Types
Portal sends numbers as JSON numbers, Go might serialize differently:
- `"timeout": 600000000000` (number)
- `"chainId": 1` (number)

### 2. Array/Object Ordering
Portal uses specific ordering, Go's JSON marshal might reorder:
- `models` array order
- `questions` array order
- Object key ordering (should be alphabetical)

### 3. Nested Object Serialization
Complex nested structures like `wallet_auth` might serialize differently

### 4. Field Name Mismatches
Check if server expects different field names:
- `jobspec_id` vs `id`
- `provider_timeout` vs `providerTimeout`

---

## Diagnostic Commands

```bash
# Check deployment status
fly status --app beacon-runner-production

# Watch logs for signature debug
fly logs --app beacon-runner-production | grep -A 5 "SIGNATURE DEBUG"

# Check latest deployment
fly releases --app beacon-runner-production | head -5
```

---

## If Still Failing

Add more detailed logging to compare exact bytes:
1. Log hex dump of canonical JSON on both sides
2. Log each field individually
3. Compare Ed25519 signature generation step-by-step
