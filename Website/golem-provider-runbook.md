# Golem Provider Runbook (MVP: Single Provider Presence + Hybrid GPU)

This runbook describes how to maintain a real presence on the Golem network with a minimal footprint (recommended: single EU provider) while the MVP obtains 3 regional answers from GPU providers (Modal/RunPod) via the Hybrid Router on Railway. It includes checklists to track progress.

## Prerequisites

- Wallet & Network
  - Funded GLM wallet and gas token appropriate for the chosen network.
- Host
  - Ubuntu 22.04 LTS (VM) or Docker host.
  - Open ports: `7464-7465` (Yagna), `8080` (provider health/API), `443` (TLS proxy).
- Repo paths
  - Hybrid router: `Website/flyio-deployment/hybrid_router.py`
  - Railway config: `Website/railway.json`
  - Portal config: `Website/netlify.toml`

## MVP Scope and Milestones

- [ ] MVP uses GPU providers (Modal/RunPod) for 3 regional answers (`us-east`, `eu-west`, `asia-pacific`).
- [ ] Maintain real Golem presence with 1 provider (recommended EU) for credibility and future growth.
- [ ] Wire only the EU Golem endpoint into the Hybrid Router (others optional/unset).
- [ ] Keep portal and router stable (WS via Railway, API via Runner on Fly until migrated).

## Bring-up (MVP: Single EU Provider)

Preferred path for MVP speed: upgrade the existing Fly app `beacon-golem-simple` to expose HTTPS `/health` and `/inference` (already implemented in `Website/golem-provider/` as `app.py` + `startup.sh` updates). VM path remains as an alternative if you want to avoid Fly.

We assume a health server on `:8080` and Yagna on `:7464-7465`. Terminate TLS on `:443` (Fly will terminate HTTPS in front of internal `:8080`; on a VM use Caddy/Nginx).

1) Start services on provider host (Terminal A)
- Install Yagna and start daemons
  ```bash
  curl -sSf https://join.golem.network/as-provider | bash
  export PATH="$HOME/.local/bin:$PATH"
  yagna service run --api-allow-origin='*' &
  sleep 10
  yagna app-key create requestor || true
  yagna provider preset create --preset-name beacon-provider \
    --exe-unit wasmtime --pricing linear \
    --price-duration 0.1 --price-cpu 0.1 --price-initial 0.0 || true
  yagna provider preset activate beacon-provider
  yagna provider run &
  ```
- Start provider HTTP health/inference service on `:8080` (FastAPI/Node) and front it with Caddy/Nginx on `:443`.

2) Verify health (Terminal A or C)
```bash
# With public domain (Fly simple provider)
curl -sSfL https://beacon-golem-simple.fly.dev/health | jq .
# Or with your own domain (VM or custom DNS)
curl -sSfL https://golem-eu.<your-domain>/health | jq .
# Or direct host (if reachable)
curl -sSfL http://<provider-host>:8080/health | jq .
```
Expected: HTTP 200 JSON with a healthy status payload.

## Configure Hybrid Router (Railway)

The router supports both a comma-separated list and region-specific env vars. For MVP, set only the EU endpoint; leave US/APAC unset.

- Option (MVP recommended): region-specific (EU only)
```
GOLEM_EU_ENDPOINT = https://beacon-golem-simple.fly.dev
```

- Optional (later): add more regions
```
GOLEM_US_ENDPOINT   = https://golem-us.<your-domain>
GOLEM_APAC_ENDPOINT = https://golem-apac.<your-domain>
```

Notes:
- If `GOLEM_*` vars are unset, the router will route only among GPU providers (Modal/RunPod).
- Providers are initialized in `setup_providers()` in `Website/flyio-deployment/hybrid_router.py`.

Redeploy (Terminal E):
- Push to `main` or run Railway “Redeploy”.
- Startup logs will list configured providers (expect `golem-eu-west` for MVP).

Set via Railway CLI (Terminal E):
```bash
railway variables set GOLEM_EU_ENDPOINT=https://beacon-golem-simple.fly.dev
railway redeploy
```

## Validation

1) List providers via router (Terminal C)
```bash
curl -sSfL https://project-beacon-production.up.railway.app/providers | jq .
```
Expect to see GPU providers (Modal/RunPod) for all regions and a single Golem entry for `eu-west` (if configured). `healthy: true` once `/health` returns 200.

2) Region-pinned smoke test (GPU-backed answers)
```bash
curl -sSfL -X POST https://project-beacon-production.up.railway.app/inference \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "llama-3.2-1b",
    "prompt": "Say hello from US-East",
    "temperature": 0.1,
    "max_tokens": 64,
    "region_preference": "us-east",
    "cost_priority": true
  }' | jq .
```
Verify `provider_used` and that the response is successful.

3) Portal sanity
- Open the portal and submit a job; confirm routing matches expectations and the WebSocket shows updates.

## Failover & Cross-Region

- With ≥2 regions online (GPU providers already cover 3 regions), temporarily disable one route.
- Submit an inference without `region_preference`.
- Router should choose a healthy region based on cost/latency.

## Troubleshooting

- Router shows no Golem providers
  - For MVP this is acceptable if EU endpoint isn’t set. To enable, set `GOLEM_EU_ENDPOINT` and redeploy.
  - Check router startup logs for provider list.

- Providers unhealthy in router
  - Confirm `/health` returns 200 at the configured domain.
  - Check TLS validity and firewall ingress (`443`, `7464`, `7465`).

- Inference errors
  - Inspect provider logs.
  - Confirm `/inference` payload fields: `model`, `prompt`, `temperature`, `max_tokens`.

- Portal CORS/WS
  - `netlify.toml` uses same-origin proxy for API and WS.
  - CSP already allows `wss:`; keep overrides minimal.

## Ops Notes

- Autosleep
  - We only run post-deploy smoke checks; no scheduled polling.

- Costs
  - Router default costs: Golem (0.0001) < RunPod (0.00025) < Modal (0.0003). Adjust in `hybrid_router.py` if needed.

- Observability
  - `GET /metrics` for high-level router metrics.
  - `GET /providers` for detailed provider states.
  - `GET /ws` returns a JSON hint (HTTP GET). Full WS upgrade via portal.

## Milestone Checklist (MVP)

- [ ] EU provider available (prefer upgrading `beacon-golem-simple` on Fly) OR VM provisioned
- [ ] Firewall open: 443, 7464, 7465; SSH restricted
- [ ] Yagna running (`yagna service run`), provider preset active (`yagna provider run`)
- [ ] Health/inference service on `:8080`, TLS proxy on `:443`
- [ ] `https://beacon-golem-simple.fly.dev/health` (or your VM domain) returns 200 JSON
- [ ] Railway env set: `GOLEM_EU_ENDPOINT=https://beacon-golem-simple.fly.dev` (or your VM domain; US/APAC unset)
- [ ] Redeploy Railway; `/providers` shows `golem-eu-west` and GPU providers
- [ ] Region-pinned smoke tests succeed for `us-east`, `eu-west`, `asia-pacific`
- [ ] Failover without `region_preference` selects a healthy region

## Alternative: VM Path (optional)

If you prefer not to use Fly for the MVP provider, use a budget VM (e.g., Hetzner CPX41), attach DNS (`golem-eu.<your-domain>`), terminate TLS with Caddy, and set:
```
GOLEM_EU_ENDPOINT = https://golem-eu.<your-domain>
```
Then redeploy and validate as above.

## Next Steps (Beyond MVP)

- [ ] Add `GOLEM_US_ENDPOINT` and/or `GOLEM_APAC_ENDPOINT` when additional providers are online
- [ ] Migrate Runner off Fly (portal `/api/v1` proxy) once ready
- [ ] Publish community JobSpec/container to attract requestors on Golem and earn GLM
