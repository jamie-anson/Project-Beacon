---
id: architecture
title: Architecture
---

Pipeline overview:

1. JobSpec →
2. Runner (Golem) →
3. Receipt →
4. IPFS →
5. Transparency Log (e.g., Rekor) →
6. Diff Engine →
7. Attestation Report

Notes:
- Cryptographic primitives secure integrity and provenance.
- IPFS provides durability and content-addressing.
- Rekor offers append-only proofs.

Placeholders:
- Latest IPFS CID: `bafy...TBD`
- Rekor log entry: `sha256:...TBD`
