# Demand Templates (Sample)

This folder contains sample demand templates to help encode resource requirements during Yagna negotiations. These are examples to copy into your requestor code (e.g., golem-js) or to adapt in your own tooling.

Important:
- Use the constraint expressions as a starting point. Validate GPU-specific property keys against your runtime/provider environment.
- These JSON files intentionally focus on `constraints` only, to avoid guessing additional property names.
- Combine with your pricing strategy on the requestor side.

Templates:
- `demand.cpu.json` — CPU-only baseline.
- `demand.gpu.nvidia.8gib.json` — NVIDIA GPU tier (8+ GiB VRAM).
- `demand.gpu.nvidia.24gib.json` — NVIDIA GPU tier (24+ GiB VRAM).
- `demand.gpu.amd.8gib.json` — AMD ROCm-capable GPU tier (8+ GiB VRAM).

Usage (conceptual example with golem-js):
```ts
// Pseudocode: set constraints string on your demand
const constraints = readFileSync('golem-provider/market/demand.gpu.nvidia.8gib.json').constraints;
// demand.setConstraints(constraints);
```

Notes on placeholders:
- `${GPU_VENDOR_EXPR}` and `${GPU_VRAM_EXPR}` are placeholders for GPU-specific property keys/expressions.
- Consult your provider/runtime documentation to replace them with correct expressions.
- Example ideas (to be validated): vendor, model family, VRAM in GiB.

## Runner API Quickstart (port 8090)

Example submitting a job with constraints (requires jq):

```bash
CONSTRAINTS=$(jq -r .constraints golem-provider/market/demand.cpu.json)
curl -X POST http://localhost:8090/api/v1/jobs \
  -H 'Content-Type: application/json' \
  -d "{\n  \"id\": \"cpu-baseline-$(date +%s)\",\n  \"constraints\": \"$CONSTRAINTS\",\n  \"benchmark\": { \"name\": \"text-generation\", \"model\": \"llama3.2\" },\n  \"regions\": [\"testnet\"]\n}"
```

## Helper Script

Use `scripts/submit-job.sh` to submit with a tier and model:

```bash
# Tiers: cpu | nvidia-8gib | nvidia-24gib | amd-8gib
scripts/submit-job.sh nvidia-8gib llama3.2 my-job-001
```

Related plans:
- See `containers-plan.md` → "Yagna/Golem Negotiation Updates" for next steps (GPU constraints & pricing tiers).
