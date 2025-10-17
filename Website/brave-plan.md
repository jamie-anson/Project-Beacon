# Brave Wallet Compatibility Plan

## Goals
- Ensure Brave Wallet (desktop & mobile) users can authorize and submit jobs without signature failures.
- Maintain compatibility with MetaMask and other EIP-1193 providers.

## Current Symptoms
- Brave mobile can complete SIWE but job submission fails (missing Ed25519 signature / wallet_auth).
- `window.ethereum.isMetaMask` check blocks Brave-specific provider.
- Need diagnostics to confirm message encoding and parameter ordering on Brave.

## Diagnostics Checklist
1. **Provider Detection**
   - Inspect `window.ethereum` flags (isMetaMask, isBraveWallet, providers array).
   - Capture which provider is selected when multiple wallets are installed.
2. **Signature Capture**
   - Log personal_sign parameters and signature output.
   - Compare canonical job spec hash between MetaMask and Brave.
3. **Chain Context**
   - Verify `wallet_switchEthereumChain` behavior on Brave and ensure chainId matches SIWE payload.
4. **Network Observability**
   - Collect Runner API responses for failed submissions (`signature_mismatch`, `wallet_auth_required`, etc.).

## Implementation Tasks
- **T1 Detect Brave Wallet**
  - Update `isMetaMaskInstalled()` in `portal/src/lib/wallet.js` to accept `ethereum.isBraveWallet` and iterate `ethereum.providers`.
- **T2 Provider Selection Helper**
  - Create helper to choose preferred provider (MetaMask > Brave > fallback) and store reference for signing.
- **T3 Personal Sign Robustness**
  - Wrap `personal_sign` call to normalize UTF-8 strings and try both parameter orders.
- **T4 Chain Sync**
  - Invoke `wallet_switchEthereumChain` before signing if current chain differs from configured chainId.
- **T5 Logging & Telemetry**
  - Add debug logging (only in dev builds) for provider flags, canonical JSON length, SHA-256.
  - Provide UI hint when Brave is detected but lacks wallet_auth.

## Validation Plan
- Test matrix:
  - MetaMask desktop (Chrome).
  - MetaMask mobile (iOS/Android).
  - Brave desktop (with & without MetaMask extension).
  - Brave mobile (native wallet).
- Confirm job submission success, verify signatures on runner side, and ensure SIWE flow still passes.

## Rollout & Documentation
- Update SoT `docs/sot/facts.json` once detection logic changes.
- Document Brave support in portal README / signing guide.
- Communicate workaround until patch is deployed (inform Brave users).

## Open Questions
- Do we need provider-dispatch in other flows (WS, future typed data signing)?
- Should we store last-used provider in localStorage for consistent behavior?
- Any security implications of accepting generic EIP-1193 providers?

## 2025-10-17 Status
- Brave iOS SIWE: signature sheet loops; console repeats `JsonRpcProvider failed to detect network`, `eth_chainId` resolves to `undefined`.
- `portal/src/lib/wallet.js` now skips chain sync for Brave and signs using raw `window.ethereum.request({ method: 'personal_sign' })` fallback.
- `localStorage` flags `beacon:disable_chain_sync` and `beacon:wallet_auth` cleared during tests; issue persists.

## Next Steps
- Run direct console call `window.ethereum.request({ method: 'personal_sign', params: ['test', address] })` and record result/error.
- Capture Brave wallet version and browser build for bug report.
- Draft outreach message to Brave Wallet team with logs and reproduction steps.
