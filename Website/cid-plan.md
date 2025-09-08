# CID Publishing & Verification Plan

Status: Draft
Date: 2025-09-08
Owner: Website/Infra

## Goal
Establish a reliable, automated process to:
- Publish built site artifacts to IPFS (Storacha-backed)
- Embed the resulting real CID(s) into the UI (docs footer and optional portal UI)
- Provide verifiable links via multiple gateways
- Add tests that fail on placeholders or non-resolving CIDs

## Current State (observed)
- Docs footer “Build CID” is populated from an environment var `DOCS_BUILD_CID` in `docs/docusaurus.config.ts`.
  - Source: `docs/docusaurus.config.ts` (uses `process.env.DOCS_BUILD_CID || 'bafy...placeholder'`)
- Netlify build command (`netlify.toml`) computes a pseudo-CID via `scripts/postbuild-pin.js` and writes `dist/docs-cid.txt`.
  - This is a deterministic SHA-256 based placeholder, not a real IPFS CID.
  - Then Netlify rebuilds docs with `DOCS_BUILD_CID` set from `dist/docs-cid.txt`.
- Optional GitHub Action exists at `.github/workflows/ipfs.yml` using `ipfs/ipfs-deploy-action@v1` to publish `portal/dist` to Storacha/IPFS.
  - Outputs a real `cid` in the workflow summary when secrets are present
  - Does not publish `dist/docs` nor write the real CID back into the repo/Netlify environment

Conclusion: The footers show a CID-like placeholder, not the real IPFS CID, because the real publishing flow (GH Action) isn’t wired into the docs build.

---

## Hypotheses (check all that apply)
- [ ] H1: IPFS action is currently disabled (missing `STORACHA_KEY` / `STORACHA_PROOF`), so no real CID is produced.
- [ ] H2: Even when the action runs, it publishes only `portal/dist`, not `dist/docs`.
- [ ] H3: There is no data flow from the action output CID to the docs build (`DOCS_BUILD_CID`), so the UI keeps the pseudo-CID.
- [ ] H4: The displayed CID at the bottom of the docs is mislabeled (it’s a build-hash placeholder, not a published CID).

---

## Plan Overview (milestones)
- [ ] M1: Reproduce and capture current behavior (action logs, footers, gateway checks)
- [x] M2: Decide single source of truth and data flow
- [x] M3: Implement GitHub Action publishing for both docs and portal with outputs
- [ ] M4: Persist real CIDs to repo (`ipfs/cids.json`) on main (with [skip ci]) — in progress (awaiting secrets + main run)
- [x] M5: Netlify build consumes `ipfs/cids.json` to embed real CIDs into UI
- [ ] M6: Update UI labels/links and add tests
- [ ] M7: Validate across gateways and add monitoring

---

## Step-by-Step Tasks

### M1. Reproduce & Capture Evidence
- [ ] Confirm docs footer shows placeholder CID:
  - Open deployed `/docs/` → footer shows `Build CID: bafy...` and links to `ipfs.io/ipfs/<placeholder>`
- [ ] Check GitHub Actions run for `.github/workflows/ipfs.yml`:
  - Verify if step “Check IPFS secrets (Storacha)” reports enabled/disabled
  - If skipped, note that action cannot produce a real CID
- [ ] If enabled, confirm action summary shows a real CID (not placeholder) and test it:
  - [ ] `curl -I https://ipfs.io/ipfs/<CID>` → expect 200/302
  - [ ] `curl -I https://cloudflare-ipfs.com/ipfs/<CID>` → expect 200/302

### M2. Decide Data Flow (recommendation)
Choose Option A (preferred) for lowest maintenance:
- [x] A. GitHub Action publishes to IPFS and commits `ipfs/cids.json` with latest CIDs on `main` (commit message contains `[skip ci]`)
- [x] Netlify build reads `ipfs/cids.json` to set `DOCS_BUILD_CID` (fallback to placeholder when missing)

Alternative (parked):
- [ ] B. Teach Netlify build to publish directly to IPFS (requires bringing a CLI/lib + secrets to Netlify) — more moving parts in Netlify.

### M3. Expand GitHub Action Publishing
- [x] In `.github/workflows/ipfs.yml`:
  - [x] Build docs in CI:
    - `npm ci`
    - `npm run build:static && npm run build:docs` (or directly `docusaurus build docs --out-dir dist/docs`)
  - [x] Publish `dist/docs` via `ipfs/ipfs-deploy-action@v1` (second step):
    - Output as `steps.docs_ipfs.outputs.cid`
  - [x] Keep portal publish step; capture as `steps.portal_ipfs.outputs.cid`
  - [x] Write both to Step Summary
  - [x] Grant `contents: write` and add step to commit `ipfs/cids.json` on `main`

### M4. Persist CIDs to Repo on main
- [ ] Add a step (guarded by `if: github.ref == 'refs/heads/main'`) to write/commit a JSON file:
  - Path: `ipfs/cids.json`
  - Shape:
    ```json
    {
      "commit": "<short_sha>",
      "docs": "<bafy...>",
      "portal": "<bafy...>",
      "timestamp": "<ISO8601>"
    }
    ```
  - [ ] Commit with message: `chore: update IPFS CIDs [skip ci]`
  - [ ] Ensure `actions/checkout@v4` uses default token with write perms (or `GITHUB_TOKEN`)

### M5. Netlify Build Consumes Real CIDs
- [ ] Modify `netlify.toml` build command so `DOCS_BUILD_CID` prefers the committed JSON value:
  - Example inline override in build command:
    - `DOCS_BUILD_CID=$(node -e "try{console.log(require('./ipfs/cids.json').docs||'bafy...placeholder')}catch(e){console.log('bafy...placeholder')}")`
  - [ ] Keep current `postbuild:cid` as fallback only
- [ ] Optionally surface `VITE_PORTAL_CID` to the portal build (for a portal About/Settings display)

### M6. UI Updates & Tests
- [ ] In `docs/docusaurus.config.ts`, clarify labels and include both values:
  - [ ] Rename current link to `Published CID` (if fed from `ipfs/cids.json`)
  - [ ] Optional: Also show `Build hash` (pseudo) for debug transparency
- [ ] Add a validation script (similar to `scripts/test-build-output.js`) that fails on placeholders:
  - [ ] Check that `DOCS_BUILD_CID` starts with `bafy` and is not `'bafy...placeholder'`
  - [ ] Optionally, probe gateway resolution via HEAD (skip in CI if rate limited)
- [ ] For PRs, keep IPFS Action `set-pr-comment: true` to show preview CIDs

### M7. Validation & Monitoring
- [ ] Manual verify both gateways resolve the CID
- [ ] Add a small endpoint-check in pre-deploy validation to assert non-placeholder CID is embedded
- [ ] Document runtime override for portal IPFS gateway (`/settings`) already exists; ensure it works with new flow

---

## Acceptance Criteria
- [ ] Action publishes both docs and portal artifacts to IPFS (Storacha) and prints CIDs in the job summary
- [ ] `ipfs/cids.json` exists on `main` with latest values after each merge
- [ ] Netlify-built docs footer shows a real CID (not placeholder), linking to a working gateway URL
- [ ] Build/test pipeline fails if a placeholder CID is embedded
- [ ] PRs get comment with preview CID(s)

## Rollback Plan
- [ ] Revert `netlify.toml` changes to use only pseudo-CID
- [ ] Disable the repo-commit step in the IPFS workflow (comment out or set `if: false`)
- [ ] Site remains functional; only verifiability link reverts to placeholder

## Secrets & Safety
- [ ] Set `STORACHA_KEY` and `STORACHA_PROOF` in GitHub repo secrets
- [ ] Never print secret values in logs
- [ ] Commit step uses `GITHUB_TOKEN` least privilege; commit message includes `[skip ci]`

## Appendix: File References
- `.github/workflows/ipfs.yml` — IPFS deploy (Storacha)
- `netlify.toml` — Build pipeline & env passing
- `scripts/postbuild-pin.js` — pseudo-CID generator (placeholder)
- `docs/docusaurus.config.ts` — footer CID display
- `portal/src/lib/api.js` — IPFS gateway resolution for bundles (runtime)

## Appendix: Quick Commands
- Verify gateway resolution (replace <CID>):
```bash
curl -I https://ipfs.io/ipfs/<CID>
curl -I https://cloudflare-ipfs.com/ipfs/<CID>
```
- Print current placeholder from build output:
```bash
cat dist/docs-cid.txt || echo "missing"
```
