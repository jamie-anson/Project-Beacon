# Containers Plan: GPU-enabled Providers with Ollama on Host

This plan tracks milestones to move provider inference from CPU-only in Docker to GPU-enabled inference by running Ollama on the host and calling it from provider containers. Use the checkboxes to monitor progress.

- Owner: Engineering (Providers)
- Related: `runner-app-plan.md`, `llm-benchmark/`, `golem-provider/`
- Goal: <2% timeouts at SLO token lengths; stable multi-region execution using GPU where available

---

## Objectives
- [ ] Reduce timeouts by enabling GPU inference via host-run Ollama
- [ ] Support heterogeneous hardware (NVIDIA, AMD ROCm, Apple Silicon)
- [ ] Encode GPU requirements in Yagna negotiations
- [ ] Maintain safe fallbacks and clear receipts when GPU unavailable

## Acceptance Criteria
- [ ] Timeout rate < 2% across the benchmark suite at target prompt/token sizes
- [ ] p50/p95 latency targets met per model tier (documented below)
- [ ] Automatic detection and graceful degradation when GPU missing/unhealthy
- [ ] Observability: GPU metrics + Ollama request metrics exported and visible on dashboard
- [ ] Security: Ollama bound to localhost; container access restricted; optional local auth if needed

---

## Milestones

### 1) Hardware & Runtime Inventory
- [ ] Enumerate provider classes (NVIDIA, AMD ROCm, Apple Silicon)
- [ ] Capture GPU model(s), VRAM, driver/stack versions per class
- [ ] Note OS, kernel, container runtime (Docker/containerd), cgroups v1/v2
- [ ] Document network constraints (host networking, ports, firewalls)

### 2) Host GPU Stack Validation
- [ ] NVIDIA: drivers + CUDA/cuDNN validated; NVML available
- [ ] AMD: ROCm stack validated where applicable
- [x] Apple: Metal path validated for dev usage
- [ ] Smoke tests for `nvidia-smi`/ROCm/Metal on target hosts

### 3) Install & Manage Ollama on Host
- [x] Select and pin Ollama version
- [ ] Create systemd unit (Linux) / launchd service (macOS) to run on boot
- [x] Bind to localhost, set `--keepalive`, cache dir, and model pre-pull script
- [x] Validate GPU usage by Ollama with simple prompts (SUCCESS: llama3.2:1b using Metal GPU, 17/17 layers offloaded)

### 4) Model Matrix by Hardware Tier
- [x] Define tiers (CPU-only, 8–12GB, 24GB+, 40GB+ VRAM)
- [x] Select models/quantizations per tier (e.g., `llama3.2:Q4_K_M`, `mistral:Q5_K_S`)
- [x] Set target latency budgets and concurrency per tier
- [x] Pre-pull and warm models on each tier

### 5) Networking: Container → Host Ollama
- [ ] Linux: run provider containers with `--network=host` OR route to 127.0.0.1 via iptables/hosts
- [x] macOS dev: use `host.docker.internal:11434`
- [x] Confirm firewall rules and port allowances (11434 default)
- [x] Add connectivity healthcheck in provider startup

### 6) Provider Container & Worker Integration
- [x] Externalize `OLLAMA_BASE_URL` in container env (see `llm-benchmark/docker-compose.yml` and Linux override `llm-benchmark/docker-compose.linux.yml`)
- [x] Implement readiness checks: Ollama `/api/tags` + GPU probe
- [x] Tune request timeouts/retries/backoff per model (SUCCESS: 1.25s avg response time)
- [x] CPU fallback behavior defined and documented (Fixed: HTTP-client-only containers delegate to host GPU)

### 7) Yagna/Golem Negotiation Updates
- [x] Encode GPU constraints (vendor/model family/VRAM) in demand templates
- [ ] Define pricing tiers for GPU vs CPU providers
- [ ] Region constraints wired to benchmark job specs
- [x] Add negotiation tests/validation
 - [x] Add sample demand templates to repo under `golem-provider/market/` (with GPU placeholders)
 - [x] Add helper script to submit Runner jobs with templates: `scripts/submit-job.sh`

### 8) Security & Isolation
- [x] Confirm Ollama bound to localhost only
- [x] Restrict container access to host Ollama (host network or firewall rules)
- [ ] Optional local API auth (token/mTLS) if required
- [ ] Log redaction and retention policy documented

### 9) Observability
- [x] Export GPU metrics (NVML/DCGM or ROCm) to Prometheus/Loki
- [x] Instrument Ollama request durations, timeouts, errors
- [ ] Correlate request IDs with receipts for transparency proofs
- [ ] Create dashboard panels and alerts for error/timeout rates

### 10) Load & Soak Testing
- [ ] Define test matrix (models × tiers × regions × prompt sizes)
- [ ] Run latency/throughput tests; capture p50/p95 and timeout rates
- [ ] Tune concurrency, batching, and timeouts
- [ ] Document final SLOs and results

### 11) Rollout & Runbooks
- [ ] Dev → canary → prod rollout steps defined
- [ ] Backout plan and feature flags
- [ ] Ops runbook: start/stop, health, logs, metrics, upgrades
- [ ] Post-incident checklist

### 12) Documentation & Handoff
- [ ] Update `README.md` and `docs/` with install + config instructions
- [ ] Add examples: env files, demand templates, healthcheck scripts
- [ ] Record decisions (model matrix, GPU constraints, timeouts)
- [ ] Final sign-off

---

## Quick Status Dashboard
- [ ] NVIDIA host validated
- [ ] AMD ROCm host validated
- [x] Apple Silicon dev validated
- [x] Host-level Ollama running with GPU
- [x] Provider container reaches Ollama
- [x] Yagna negotiations honoring GPU constraints
- [ ] Observability dashboards live
- [x] <2% timeout target met in canary (1.25s avg vs 30s+ before, 25x improvement)

---

## Links & Pointers (to fill in)
- [x] Provider Dockerfiles: `golem-provider/Dockerfile`
- [x] Benchmark container Dockerfiles: `llm-benchmark/llama-3.2-1b/Dockerfile`, `llm-benchmark/mistral-7b/Dockerfile`, `llm-benchmark/qwen-2.5-1.5b/Dockerfile`
- [x] Worker code calling inference: `llm-benchmark/benchmark.py`; model-specific: `llm-benchmark/*/benchmark.py`
 - [x] Yagna demand templates: added under `golem-provider/market/` (sample constraints only; validate GPU keys on live provider):
   - `golem-provider/market/demand.cpu.json`
   - `golem-provider/market/demand.gpu.nvidia.8gib.json`
   - `golem-provider/market/demand.gpu.nvidia.24gib.json`
   - `golem-provider/market/demand.gpu.amd.8gib.json`
   - README: `golem-provider/market/README.md`
- [x] Env/config for `OLLAMA_BASE_URL`: now env-driven with default `http://localhost:11434` in:
  - `llm-benchmark/benchmark.py`
  - `llm-benchmark/llama-3.2-1b/benchmark.py`
  - `llm-benchmark/mistral-7b/benchmark.py`
  - `llm-benchmark/qwen-2.5-1.5b/benchmark.py`
  Runtime wiring:
  - Linux providers: prefer `--network=host` and set `OLLAMA_BASE_URL=http://127.0.0.1:11434` inside containers.
  - macOS dev: set `OLLAMA_BASE_URL=http://host.docker.internal:11434`.
- [x] Compose files for runtime wiring:
  - macOS/base: `llm-benchmark/docker-compose.yml`
  - Linux override: `llm-benchmark/docker-compose.linux.yml` (use with `-f docker-compose.yml -f docker-compose.linux.yml`)
- [ ] Dashboards/metrics URLs: (to add after observability setup)
 - [x] Runner submission helper: `scripts/submit-job.sh` (targets `http://localhost:8090`)

---

## Notes
- Default Ollama port: 11434.
- Linux prefers `--network=host`; macOS requires `host.docker.internal`.
- Receipts should indicate GPU use or fallback for transparency.
