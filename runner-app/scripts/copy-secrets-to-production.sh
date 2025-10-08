#!/bin/bash
# Copy essential secrets from beacon-runner-change-me to beacon-runner-production

echo "Copying secrets from old app to new production app..."

# Core secrets that must be copied
SECRETS=(
  "DATABASE_URL"
  "REDIS_URL"
  "ADMIN_TOKEN"
  "TRUSTED_KEYS_JSON"
  "STORACHA_TOKEN"
  "HYBRID_BASE"
  "OPENAI_API_KEY"
)

# Feature flags
flyctl secrets set -a beacon-runner-production \
  TRUST_ENFORCE=false \
  REPLAY_PROTECTION_ENABLED=true \
  IPFS_ENABLED=true \
  RUNNER_SIG_BYPASS=true \
  USE_MIGRATIONS=true \
  HYBRID_ROUTER_DISABLE=false \
  ENABLE_HYBRID_DEFAULT=true

echo "✓ Feature flags set"
echo ""
echo "Now you need to manually copy these secrets:"
echo "  - DATABASE_URL"
echo "  - REDIS_URL" 
echo "  - ADMIN_TOKEN"
echo "  - TRUSTED_KEYS_JSON"
echo "  - STORACHA_TOKEN"
echo "  - HYBRID_BASE"
echo "  - OPENAI_API_KEY"
echo ""
echo "Get values from old app with:"
echo "  flyctl secrets list -a beacon-runner-change-me"
echo ""
echo "Set them with:"
echo "  flyctl secrets set -a beacon-runner-production KEY=value"
