# MetaMask Wallet Authentication Integration

## Overview

Project Beacon now supports MetaMask wallet authentication as an alternative to the previous Ed25519-only signing system. Users can connect their MetaMask wallet to authorize their Ed25519 keys for job submissions, providing a familiar Web3 authentication experience.

## How It Works

1. **Ed25519 Key Generation**: The portal still generates Ed25519 keypairs for cryptographic signing of job specifications
2. **Wallet Authorization**: Users connect their MetaMask wallet and sign an authorization message
3. **Dual Authentication**: Job submissions include both the Ed25519 signature and wallet authentication proof
4. **Backend Validation**: The runner app validates both signatures before accepting jobs

## User Flow

### 1. Connect Wallet
- User clicks "Connect Wallet" button
- MetaMask prompts for connection approval
- Portal displays connected wallet address

### 2. Authorize Ed25519 Key
- Portal generates authorization message: `"Authorize Project Beacon key: [ed25519-public-key]"`
- User signs the message in MetaMask
- Authorization is stored locally for future use

### 3. Submit Jobs
- Jobs are signed with Ed25519 key (existing flow)
- Wallet authentication is automatically included
- Backend validates both signatures

## Technical Implementation

### Frontend (Portal)

#### New Dependencies
- `ethers@^6.13.0` - Ethereum wallet interaction

#### Key Files
- `src/lib/wallet.js` - Wallet connection and signing utilities
- `src/components/WalletConnection.jsx` - UI component for wallet management
- `src/lib/crypto.js` - Updated to support wallet authentication
- `src/pages/BiasDetection.jsx` - Updated to require wallet connection

#### API Changes
Job submission payloads now include:
```json
{
  "id": "job-123",
  "signature": "ed25519-signature",
  "public_key": "ed25519-public-key",
  "wallet_auth": {
    "address": "0x742d35Cc6634C0532925a3b8D",
    "signature": "0x8a7b2c3d4e5f...",
    "message": "Authorize Project Beacon key: MCowBQYDK2VwAyEA..."
  }
}
```

### Backend (Runner App)

#### New Dependencies
- `github.com/ethereum/go-ethereum@v1.13.5` - Ethereum signature validation

#### Key Files
- `pkg/crypto/wallet.go` - Wallet signature validation functions
- `pkg/models/jobspec.go` - Updated JobSpec with WalletAuth field
- `pkg/crypto/wallet_test.go` - Comprehensive test suite

#### Validation Flow
1. Parse wallet authentication from job payload
2. Validate Ethereum address format
3. Verify wallet signature against authorization message
4. Confirm Ed25519 key matches message content
5. Continue with existing Ed25519 signature validation

## Security Features

### No Private Key Exposure
- Ed25519 private keys remain in browser localStorage
- Wallet private keys never leave MetaMask
- Only signatures are transmitted

### Message Format Validation
- Authorization messages follow strict format
- Ed25519 public key must match between message and job
- Prevents key substitution attacks

### Signature Verification
- Ethereum signature recovery validates wallet ownership
- Ed25519 signature ensures job integrity
- Both signatures must be valid for job acceptance

### Replay Protection
- Ed25519 signatures include timestamp and nonce
- Wallet authorization has 7-day expiration
- Account changes clear stored authorization

## Error Handling

### MetaMask Not Installed
- Clear instructions to install from https://metamask.io
- Graceful fallback UI state

### Connection Errors
- User rejection: Clear explanation of required approval
- Pending requests: Guidance to check MetaMask
- Internal errors: Retry instructions

### Signing Errors
- User rejection: Explanation of authorization purpose
- Locked wallet: Instructions to unlock MetaMask
- Network errors: Retry mechanisms

### Backend Validation Errors
- Invalid wallet signature: Clear error message
- Message format errors: Detailed validation feedback
- Ed25519 key mismatch: Security-focused error response

## Backward Compatibility

### Existing Ed25519 Flow
- Jobs without wallet authentication still accepted
- Existing signed jobs remain valid
- Gradual migration path for users

### API Compatibility
- `wallet_auth` field is optional in job payloads
- Existing API endpoints unchanged
- No breaking changes to job structure

## Testing

### Frontend Tests
- Wallet connection simulation
- Error scenario handling
- UI state management
- Local storage persistence

### Backend Tests
- Ethereum signature validation
- Message format verification
- Integration with Ed25519 flow
- Error condition coverage

### End-to-End Tests
- Complete wallet connection flow
- Job submission with wallet auth
- Error recovery scenarios
- Cross-browser compatibility

## Deployment Considerations

### Environment Variables
No new environment variables required - wallet validation uses standard Ethereum cryptography.

### Database Changes
No database schema changes required - wallet authentication is validated at request time.

### Monitoring
- Track wallet authentication success/failure rates
- Monitor for invalid signature attempts
- Log wallet address usage patterns (privacy-compliant)

## User Benefits

### Familiar Experience
- Standard Web3 wallet connection flow
- No need to manage Ed25519 keys manually
- Consistent with other dApps

### Enhanced Security
- Hardware wallet support through MetaMask
- Multi-signature wallet compatibility
- Reduced key management burden

### Better UX
- One-click connection
- Persistent authorization
- Clear connection status

## Future Enhancements

### Additional Wallets
- WalletConnect for mobile wallets
- Coinbase Wallet integration
- Hardware wallet direct support

### Advanced Features
- ENS name resolution for display
- Wallet-based user profiles
- Usage analytics per address

### Traditional Auth Integration
- GitHub OAuth fallback
- Email verification option
- Multi-auth method support

## Migration Guide

### For Existing Users
1. Existing Ed25519 keys continue to work
2. Connect wallet when ready for enhanced experience
3. No data loss or re-signing required

### For Developers
1. Update portal to latest version
2. Test wallet connection flow
3. Verify job submission with wallet auth
4. Update any custom integrations

## Support

### Common Issues
- **MetaMask not detected**: Ensure extension is installed and enabled
- **Connection rejected**: User must approve connection in MetaMask
- **Signing failed**: Check wallet is unlocked and has sufficient gas
- **Authorization expired**: Reconnect wallet to refresh authorization

### Troubleshooting
1. Refresh page and try again
2. Check MetaMask is unlocked
3. Verify correct network (any network works)
4. Clear browser cache if issues persist

### Contact
For technical issues or questions about wallet integration, please refer to the project documentation or submit an issue in the repository.
