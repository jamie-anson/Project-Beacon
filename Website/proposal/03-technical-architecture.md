---
title: Technical Architecture
---

Flow:
1. Job spec created (model, prompts, seeds, regions, budgets).
2. Tasks packaged (Docker) and dispatched to Golem providers by region.
3. Results collected; content-addressed to IPFS; Merkle root/sequence updated.
4. Proofs and bundles exposed via API + Portal; public verification scripts.

Key details:
- Golem: task definition, budget caps, timeout/retry, payment driver/network.
- Storage: IPFS (CIDs), integrity via Merkle trees; receipts and sequence numbers.
- Security/privacy: key management, deterministic hashing, prompt safety, abuse mitigations.
- Observability: Prometheus metrics, structured logs, WS events for live activity.

Artifacts in repo:
- Diagrams (Mermaid/PlantUML), API/JSON schemas, example job specs and proofs.


---
First Draft
---

## Technical Architecture

The technical architecture of Project Beacon is designed to leverage the decentralized capabilities of the Golem Network, ensuring scalability, transparency, and security in AI processing.

### Flow
1. **Job Specification**: Define model parameters, prompts, seeds, regions, budgets, and target services (Golem + centralized APIs).
2. **Hybrid Task Dispatch**: 
   - Package decentralized tasks in Docker and dispatch to Golem providers by region
   - Route centralized API calls through regional Golem providers to capture geographic variations
3. **Result Collection**: Collect results from both decentralized and centralized sources, content-address them to IPFS, and update the Merkle root/sequence.
4. **Comparative Analysis**: Generate cross-service comparisons highlighting differences between decentralized and centralized AI responses.
5. **Proof Exposure**: Expose proofs and bundles via API and Portal, allowing for public verification of both execution paths.

### Key Details
- **Golem Integration**: Task definition, budget caps, timeout/retry mechanisms, and payment driver/network settings are integral to the process.
- **Centralized API Integration**: Secure API key management, rate limiting, and regional routing through Golem providers to capture geographic response variations from ChatGPT, Gemini, Claude, and other major AI services.
- **Storage Solutions**: Utilize IPFS for content addressing with CIDs, ensuring integrity through Merkle trees and maintaining receipts and sequence numbers for both decentralized and centralized results.
- **Security and Privacy**: Implement key management, deterministic hashing, prompt safety measures, and abuse mitigations to ensure secure operations across all AI service types.
- **Observability**: Employ Prometheus metrics, structured logs, and WebSocket events for real-time activity monitoring across hybrid infrastructure.
- **Attester Monitoring**: Integrate tools that allow attesters to monitor project progress in real-time, enhancing transparency and accountability.

### Artifacts in Repository
- Diagrams (Mermaid/PlantUML), API/JSON schemas, example job specifications, and proofs are maintained for reference and development.