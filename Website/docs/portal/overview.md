---
id: portal-overview
title: Runner Portal Overview
slug: /portal/overview
description: Live activity, transparency root, proof viewer, settings, and persistence in the Project Beacon Portal UI.
---

Project Beacon’s Portal provides live transparency updates, verifiable proofs, and quick access to bundles and diagnostics.

## Key UI Elements

- Live updates badge (header)
- Transparency Root card
- Activity Feed (CIDs and events)
- Proof Viewer (modal)
- Settings (runtime IPFS gateway)

## Live Updates Badge

- Text: “Live updates: Online / Offline / Error”
- Tooltip shows:
  - error message (if any)
  - retry count
  - next reconnect delay (seconds)
- WebSocket with exponential backoff (1s → 30s cap).

Code refs:
- `portal/src/state/useWs.js`
- `portal/src/App.jsx`

## Loading Skeletons

Skeleton UIs ensure graceful loading for:
- Transparency Root
- Recent Jobs
- System status
- Recent Executions
- Recent Diffs

Code ref:
- `portal/src/pages/Dashboard.jsx`

## Activity Feed

Each event may include `execution_id`, `ipfs_cid`, `merkle_root`, `timestamp/time`.

Actions per row:
- “Open” — view CID via configured gateway
- “View proof” — opens modal with Merkle proof, sequence, raw JSON
- “Copy CID”, “Copy root”

Persistence:
- `sessionStorage` key: `beacon:activity` (restored on reload)

Code refs:
- `portal/src/pages/Dashboard.jsx`
- `portal/src/components/ActivityFeed.jsx`

## Proof Viewer

- Modal displays Merkle proof, sequence, and raw JSON.
- Copy actions available for CID and root.

Code refs:
- `portal/src/components/Modal.jsx`
- `portal/src/components/ProofViewer.jsx`

## Settings

- Route: `/settings`
- Set a runtime IPFS Gateway (saved in `localStorage`, takes effect immediately)

Code refs:
- `portal/src/pages/Settings.jsx`
- `portal/src/lib/api.js` (`getIpfsGateway()`, `bundleUrl()`)

## Storage Keys

- `localStorage`:
  - `beacon:ipfs_gateway` — runtime gateway override
- `sessionStorage`:
  - `beacon:activity` — persisted Activity Feed events

## Screenshots to Include

- Dashboard showing:
  - Live Activity row with “Open”, “View proof”, “Copy CID”
  - Transparency Root card with “Copy root”
  - Skeleton states (loading)
- Settings page (gateway input, Save/Reset)
- Proof modal (root + JSON)
