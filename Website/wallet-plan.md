# MetaMask Wallet Authentication - MVP Implementation Plan

## Overview
Implement wallet-based authentication for Project Beacon MVP, targeting Golem users who already have MetaMask installed.

Status: Updated 2025-09-01T18:26:30+01:00

## Progress

- [x] Raw JSON signature verification covered at handler level (`CreateJob` accepts raw-body signed payloads; tamper rejection tested)
- [x] Negative path test for signature mismatch (post-sign tamper) added
- [x] Wallet authorization flow (MetaMask) wired to backend
- [x] Portal wallet connect and message signing
- [x] Backend wallet_auth enforcement tests added and passing: chainId allowlist, expiry enforcement, and nonce replay protection via Redis
- [x] Early trusted-keys enforcement (non-wallet_auth): revoke/deny before signature verification to return `trust_violation:<status>`
- [x] Trusted-keys hot-reload e2e tests passing (ticker-based reload + revocation takes effect)
- [x] Endpoint scan complete: only CreateJob verifies signatures at API boundary; worker and model layers remain crypto-only (no additional trust checks)
 - [x] Unit test added: early trust rejection for revoked non-wallet_auth keys (CreateJob returns `trust_violation:revoked` before signature verification)
 - [x] Observability: Prometheus metric `trust_violations_total{status}` exported and wired at API boundary (early + post-verify checks)

## Phase 1: Core Implementation (Days 1-2)

### Backend Changes
- [x] Add wallet signature validation to runner app
- [x] Create `ValidateWalletSignature`, `ValidateWalletAuth`, and `ValidateWalletAuthMessage` in Go
- [x] Update job submission endpoint to accept `wallet_auth`
- [x] Add Ethereum address recovery from signatures (EIP-191 personal_sign)
- [x] Remove dependency on trusted-keys.json for wallet-signed jobs (feature flag; transition)
  - Enable with `TRUST_ALLOW_WALLET_WITHOUT_TRUSTED_KEYS=true` on the runner. When set, any valid `wallet_auth` is treated as trusted (no wallet allowlist, no trusted-keys required) and the bound Ed25519 key is accepted.
- [x] Ensure Ed25519 job signature verification uses raw JSON (not struct re-marshal)
- [x] Add handler tests for raw JSON signature acceptance and tamper rejection

### Portal Integration  
- [x] Install ethers.js for wallet interaction
- [x] Create `connectWallet()` function for MetaMask connection
- [x] Implement message signing binding wallet -> Ed25519 public key
- [x] Store wallet authorization in localStorage (7-day cache)
- [x] Update job submission to include `wallet_auth { address, message, signature, chainId, nonce, expiresAt }`

### UI Components
- [ ] Create "Connect Wallet" button component
- [ ] Add wallet connection modal/flow
- [ ] Show connected wallet address in UI
- [ ] Display wallet connection status
- [ ] Handle MetaMask not installed case

## Phase 2: User Experience (Day 3)

### Error Handling
- [ ] Detect when MetaMask is not installed
- [ ] Handle user rejection of wallet connection
- [ ] Handle user rejection of message signing
- [ ] Provide clear error messages for each case
- [ ] Add retry mechanisms for failed connections

### Visual Feedback
- [ ] Loading states during wallet connection
- [ ] Success confirmation after wallet auth
- [ ] Visual indicator of wallet connection status
- [ ] Clear messaging about no funds required

### Integration with Existing Flow
- [ ] Update BiasDetection component to use wallet auth
- [ ] Modify job submission error handling
- [ ] Update KeypairInfo component to show wallet status
- [ ] Ensure backward compatibility during transition

## Phase 3: Testing & Polish (Day 4)

### Testing
- [ ] Test with MetaMask browser extension
- [ ] Test with WalletConnect mobile wallets
- [x] Verify signature validation on backend
- [x] Enforce and test wallet_auth chainId allowlist
- [x] Enforce and test wallet_auth expiry
- [x] Enforce and test nonce replay protection using Redis
- [ ] Test job submission end-to-end
- [ ] Test error scenarios (no wallet, rejected signing)
 - [ ] Dashboard/Alerts: Grafana panel for `sum by (status)(trust_violations_total)` and alert on elevated rates

### Documentation
- [ ] Update README with wallet authentication flow
- [ ] Create user guide for wallet connection
- [ ] Document API changes for wallet signatures
- [ ] Add troubleshooting guide for common issues

### Deployment Preparation
- [ ] Update environment variables if needed
- [ ] Test on staging environment
- [ ] Verify production deployment process
- [ ] Plan rollout to Golem community

## Technical Implementation Details

### Message Format (signed)
Use EIP-191 personal_sign with a simple, deterministic message binding the Ethereum address to the Ed25519 public key:

```
Authorize Project Beacon key: <base64-ed25519-public-key>
```

Backend validation ensures the message prefix and exact key binding match. Chain ID, nonce, and expiry are provided as payload fields (not part of the signed message) and enforced independently.

### Payload Fields (not signed by wallet)
- `chainId` (number): current EVM chain ID from `eth_chainId`
- `nonce` (string): cryptographically secure random nonce (frontend-generated)
- `expiresAt` (number): milliseconds since epoch; default now + 7 days

### Storage Format
```javascript
localStorage['beacon:wallet_auth'] = {
  address: "0x742d35Cc6634C0532925a3b8D",
  message: "Authorize Project Beacon key: MCowBQYDK2VwAyEA...",
  signature: "0x8a7b2c3d4e5f...", // EIP-191 personal_sign
  chainId: 11155111,
  nonce: "8b2b8c1a-...",
  expiresAt: 1756740000000, // ms epoch
  ed25519Key: "MCowBQYDK2VwAyEA...",
  timestamp: 1756135200000 // ms epoch
}
```

### API Changes
```
Job submission payload adds wallet_auth and keeps Ed25519 signing of the jobspec:
{
  "public_key": "<base64-ed25519>",
  "signature": "<ed25519-signature-over-raw-jobspec>",
  "wallet_auth": {
    "address": "0x742d35Cc6634C0532925a3b8D",
    "message": "Authorize Project Beacon key: MCowBQYDK2VwAyEA...",
    "signature": "0x8a7b2c3d4e5f...",
    "chainId": 11155111,
    "nonce": "8b2b8c1a-...",
    "expiresAt": 1756740000000
  }
}
```

## Success Criteria

### Functional Requirements
- [ ] Users can connect MetaMask with one click
- [ ] Wallet signature authorizes Ed25519 key for job submission
- [ ] No funds required in wallet
- [ ] Works with empty/new wallets
- [ ] Job submission succeeds after wallet auth

### User Experience Requirements
- [ ] Connection flow takes < 30 seconds
- [ ] Clear instructions for users without MetaMask
- [ ] Graceful handling of all error cases
- [ ] Visual confirmation of successful connection
- [ ] Persistent auth across browser sessions

### Technical Requirements
- [ ] Secure signature validation on backend
- [ ] No private key exposure in frontend
- [ ] Proper error logging and monitoring
- [ ] Compatible with existing Ed25519 signing flow
- [ ] Scalable for multiple wallet types (future)

## Rollout Strategy

### Week 1: Internal Testing
- [ ] Deploy to staging environment
- [ ] Test with team members
- [ ] Verify all functionality works
- [ ] Fix any critical issues

### Week 2: Golem Community Beta
- [ ] Announce to Golem Discord/Telegram
- [ ] Invite beta testers from provider community
- [ ] Gather feedback and usage data
- [ ] Monitor for any issues or confusion

### Week 3: Public Launch
- [ ] Deploy to production
- [ ] Update documentation and guides
- [ ] Announce broader availability
- [ ] Monitor adoption and success metrics

## Future Enhancements (Post-MVP)

### Additional Wallet Support
- [ ] WalletConnect for mobile wallets
- [ ] Coinbase Wallet integration
- [ ] Other popular wallet providers

### Enhanced Features
- [ ] Wallet-based user profiles
- [ ] Usage tracking per wallet address
- [ ] Reputation system based on wallet history
- [ ] Optional ENS name resolution for display

### Traditional Auth Integration
- [ ] GitHub OAuth as alternative
- [ ] Google Sign-In option
- [ ] Email verification fallback
- [ ] Multi-auth method support
