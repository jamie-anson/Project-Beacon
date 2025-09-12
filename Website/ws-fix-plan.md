# WebSocket Fix Plan (Project Beacon)

Purpose: Methodically diagnose and fix production WebSocket failures in the portal, ensure a reliable same-origin WS path via Netlify, and ship a repeatable validation checklist.

---

## Objectives
- Restore a stable WS connection in production with 101 Switching Protocols on page load.
- Make WS enabled by default in builds (no manual toggles needed).
- Provide clear verification steps and rollback options.

---

## Current Observations
- Client bundle contains latest `portal/src/state/useWs.js`:
  - Feature flag via `VITE_ENABLE_WS` or `localStorage('beacon:enable_ws')`.
  - Base URL override via `VITE_WS_BASE` or `localStorage('beacon:ws_base')`.
  - Logs `[Beacon] Using WebSocket base: ...` before connecting.
- Netlify site attempts `wss://projectbeacon.netlify.app/ws` and fails.
- Plain GET:
  - Netlify `https://projectbeacon.netlify.app/ws` → 404 currently (unexpected if rewrite is active).
  - Railway `https://project-beacon-production.up.railway.app/ws` → returns WS hint JSON (expected).
- `netlify.toml` contains rewrites for `/ws` and `/ws/*` to Railway with `status=200, force=true`.

## Source of Truth

Status legend:
- False
- Believe to be true
- Tested to be true

Shared facts:
- Netlify site origin: https://projectbeacon.netlify.app
- Railway origin: https://project-beacon-production.up.railway.app
- WebSocket path: `/ws`
- Client hook file: `portal/src/state/useWs.js`
- Redirect rules live in `netlify.toml`
- Netlify WS address: wss://projectbeacon.netlify.app/ws
- Railway WS address: wss://project-beacon-production.up.railway.app/ws
- ENVs set in UI:
  - `VITE_ENABLE_WS=1`
  - `VITE_WS_BASE=wss://project-beacon-production.up.railway.app`
  - `VITE_API_BASE=https://projectbeacon.netlify.app`

Assertions and status:
- Netlify `/ws` rewrite is active in the current deploy: Tested to be true (GET to https://projectbeacon.netlify.app/ws returns hint JSON)
- Same-origin WS to Netlify upgrades (101) from the portal: Believe to be true (pending explicit handshake test on alias)
- Railway `/ws` GET returns hint JSON: Tested to be true
- Railway WS upgrades successfully (101) from client: Tested to be true
- `VITE_ENABLE_WS` present in Netlify env for current build: False (not embedded in built env)
- Client respects localStorage overrides after hard reload: Believe to be true (hook reads `beacon:ws_base` at mount)
- CSP allows `wss` to external hosts: Believe to be true (see `netlify.toml` connect-src 'self' https: wss:)
- Production bundle includes latest `useWs` logic: Tested to be true
- Default fallback when `VITE_WS_BASE` is unset is same-origin: Tested to be true (client attempted Netlify)
- API requests use a single `/api/v1` prefix (no duplication): Tested to be true (API base normalization in `portal/src/lib/api.js`)
- Unique deploys require same-origin API base to avoid CORS: Tested to be true (set `localStorage['beacon:api_base']=location.origin` on unique deploy)

Note: A previous production deploy served only `/docs` because `dist/portal/` wasn’t included. We performed a CLI deploy of the assembled `dist/` to restore `/portal` and embedded redirects.

Additional note: Live alias is working (API + WS). Unique deploys may still encounter CORS unless API base is same-origin (runtime override available).

How to update:
- As each check is validated, change its status to "Tested to be true" and date-stamp the change in the commit message (e.g., "SOT: Netlify /ws rewrite → Tested true on 2025-09-11").

---

## Hypotheses
1) Netlify rewrite for `/ws` is not active on current deploy (cache, publish artifact, or config mismatch).
2) Client isn’t picking up the runtime override due to cache or reload timing (override set after hook mount).
3) Build-time env `VITE_ENABLE_WS` and `VITE_WS_BASE` are not set in Netlify, leaving WS disabled by default and relying on localStorage.
4) Less likely: CSP/connect-src blocking WS (current CSP allows `wss:` globally), upstream/port mismatch, or proxy upgrade issue.

---

## Step-by-Step Plan

### Step 1 — Isolate client vs. proxy (force Railway WS directly)
Goal: Ensure backend WS works independently of Netlify proxy.

Browser console on the production portal:
```js
localStorage.setItem('beacon:enable_ws', 'true');
localStorage.setItem('beacon:ws_base', 'wss://project-beacon-production.up.railway.app');
// Now do a hard reload with cache disabled (DevTools Network tab → Disable cache → Cmd+Shift+R)
```
Expected:
- Console shows: `[Beacon] Using WebSocket base: wss://project-beacon-production.up.railway.app`.
- Network → WS shows a connection to Railway with 101 Switching Protocols.

If it still connects to Netlify, re-check keys:
```js
localStorage.getItem('beacon:enable_ws'); // "true"
localStorage.getItem('beacon:ws_base');   // "wss://project-beacon-production.up.railway.app"
location.origin; // should be https://projectbeacon.netlify.app
```
Also try an Incognito window to avoid extensions interfering.

Manual sanity test (optional):
```js
// Direct WS from console to verify the endpoint upgrades
new WebSocket('wss://project-beacon-production.up.railway.app/ws');
```

Outcome:
- If Railway connects: backend is healthy → proceed to Step 2.
- If Railway fails: inspect Railway logs and `flyio-deployment/hybrid_router.py` WS settings; fix upstream before proxying.

---

### Step 2 — Fix Netlify `/ws` rewrite
Goal: Make same-origin `wss://<site>/ws` succeed so overrides are unnecessary.

Checklist:
- Confirm `netlify.toml` contains:
```
[[redirects]]
  from = "/ws"
  to = "https://project-beacon-production.up.railway.app/ws"
  status = 200
  force = true

[[redirects]]
  from = "/ws/*"
  to = "https://project-beacon-production.up.railway.app/ws/:splat"
  status = 200
  force = true
```
- Trigger “Clear cache and deploy site” in Netlify.
- After deploy: in the Netlify UI, open the deploy and check the “Redirects” section — ensure both `/ws` rules appear.

Verification:
- GET check (non-upgrade):
  - `https://projectbeacon.netlify.app/ws` should return the WS hint JSON (proxied from Railway).
- WS upgrade check:
  - From a terminal (optional):
    - Using `wscat`: `npx wscat -c wss://projectbeacon.netlify.app/ws`
    - Or `websocat`: `websocat -v wss://projectbeacon.netlify.app/ws`
  - Expected 101 upgrade and open socket.

If still failing after redeploy:
- Ensure the site’s publish directory is `dist/` (as in `netlify.toml`).
- Make a no-op commit and redeploy to ensure new `netlify.toml` is picked up.
- Temporarily set `VITE_WS_BASE` to Railway in Netlify env to bypass proxy (see Step 3) and plan a follow-up to re-enable proxy later.

---

### Step 3 — Make WS enabled by default (build-time env)
Goal: Avoid relying on manual localStorage toggles.

Netlify environment variables (UI → Site settings → Build & deploy → Environment):
- `VITE_ENABLE_WS=1`
- Optional: `VITE_WS_BASE` (leave empty to use same-origin proxy; set to Railway only if proxy remains unreliable).

Redeploy the site. Post-deploy validation:
- Reload without any localStorage overrides (clear keys):
```js
localStorage.removeItem('beacon:enable_ws');
localStorage.removeItem('beacon:ws_base');
location.reload();
```
- Expect the client to log `[Beacon] Using WebSocket base: wss://projectbeacon.netlify.app` and connect via Netlify proxy with 101.
- The message “WebSocket disabled by config” should NOT appear.

---

### Step 4 — Observability and guardrails
Goal: Make issues obvious and easier to debug next time.

- Client logging in `portal/src/state/useWs.js`:
  - Keep the info log for ws base and warn on errors.
  - Optionally gate extra logs behind `localStorage.setItem('beacon:ws_debug','1')`.
- UI indicator (optional): surface `connected`, `retries`, `nextDelayMs` somewhere visible in the portal.
- Document toggles:
  - `localStorage.setItem('beacon:enable_ws','true')` (enable)
  - `localStorage.setItem('beacon:ws_base','wss://...')` (override)
  - `localStorage.removeItem(...)` (reset)

---

### Step 5 — Rollback/Contingency
- If Netlify proxy continues to fail, ship with `VITE_WS_BASE=wss://project-beacon-production.up.railway.app` as a temporary workaround.
- Keep CSP `connect-src 'self' https: wss:` to allow external `wss://` until proxy is fixed.
- Track an issue to revisit and restore same-origin proxy once Netlify confirms the rewrites.

---

## Acceptance Criteria
- With no localStorage overrides, production portal connects to `wss://projectbeacon.netlify.app/ws` and shows 101 Switching Protocols in Network → WS.
- Console logs show `[Beacon] Using WebSocket base: wss://projectbeacon.netlify.app` (or the configured base if we choose Railway).
- No recurring `WebSocket connection failed - backend may be offline` warnings.
- POST-deploy smoke test documented and repeatable.

---

## Execution Order (Quick Checklist)
- [ ] Step 1: Force Railway via localStorage; hard reload; confirm successful WS.
- [ ] Step 2: Netlify — clear cache + redeploy; verify `/ws` rewrite in Deploy → Redirects; test GET and WS upgrade.
- [ ] Step 3: Set Netlify env `VITE_ENABLE_WS=1` (and optionally `VITE_WS_BASE`); redeploy; validate without overrides.
- [ ] Step 4: Add optional observability if needed.
- [ ] Step 5: If proxy still flaky, set `VITE_WS_BASE` to Railway as a temporary production workaround.
