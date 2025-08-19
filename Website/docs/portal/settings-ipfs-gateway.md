---
id: settings-ipfs-gateway
title: Configuring the IPFS Gateway
slug: /portal/settings-ipfs
description: Configure the IPFS gateway at runtime in the Portal. Uses a runtime override, environment fallback, and local API proxy as last resort.
---

The Portal resolves bundle links using a simple priority order:

1. Runtime override (`localStorage: beacon:ipfs_gateway`)
2. Environment variable (`VITE_IPFS_GATEWAY`)
3. Local API proxy — `/api/v1/transparency/bundles/:cid` (when no external gateway)

## Set a Gateway at Runtime

- Open `/settings`
- Enter a gateway (e.g., `https://ipfs.io`, `https://cloudflare-ipfs.com`)
- Click Save

Saved to:
- `localStorage` → `beacon:ipfs_gateway`

Takes effect immediately for “Open” buttons and bundle links.

## Env Variable (build-time)

- `VITE_IPFS_GATEWAY` (e.g., `https://ipfs.io`)

Code refs:
- `portal/src/pages/Settings.jsx`
- `portal/src/lib/api.js`:
  - `getIpfsGateway()`
  - `bundleUrl(cid)`

## Reset to Defaults

- Use the Reset action on `/settings` to clear the runtime override (falls back to `VITE_IPFS_GATEWAY` or local API).
