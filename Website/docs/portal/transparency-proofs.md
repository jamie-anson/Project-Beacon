---
id: transparency-proofs-ui
title: Transparency Proofs UI
slug: /portal/proofs
description: View and verify transparency proofs for executions and CIDs in the Portal UI.
---

The Transparency UI helps you inspect proofs, roots, and bundles.

## Transparency Root

- GET `http://localhost:8090/api/v1/transparency/root`
- Expected fields: `merkle_root | root`, `sequence`, `updated_at`
- “Copy root” button available on the card

Example (Terminal C):
```bash
curl -s http://localhost:8090/api/v1/transparency/root | jq
```

## Proof Lookup

You can fetch proofs by either `execution_id` or `ipfs_cid`.

- By execution id:
```bash
curl -s "http://localhost:8090/api/v1/transparency/proof?execution_id=<EXEC_ID>" | jq
```

- By CID:
```bash
curl -s "http://localhost:8090/api/v1/transparency/proof?ipfs_cid=<CID>" | jq
```

The Proof Viewer modal shows:
- Merkle root (`merkle_root | root`)
- Sequence
- Proof nodes (JSON)
- Copy actions for CID and root

Code refs:
- `portal/src/components/ActivityFeed.jsx`
- `portal/src/components/Modal.jsx`
- `portal/src/components/ProofViewer.jsx`

## Opening CIDs

- “Open” button resolves via the configured gateway
- Resolution order: runtime override → `VITE_IPFS_GATEWAY` → local API proxy
- Local API proxy: `GET /api/v1/transparency/bundles/:cid`
