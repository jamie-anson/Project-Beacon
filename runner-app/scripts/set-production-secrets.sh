#!/bin/bash
# Manual secret setting for beacon-runner-production
# Fill in the values from your password manager or old deployment

set -e

echo "🔐 Setting production secrets..."
echo ""
echo "⚠️  You need to fill in the actual values in this script before running!"
echo ""

# TODO: Fill in these values from your secure storage
DATABASE_URL="postgresql://..."
REDIS_URL="redis://..."
ADMIN_TOKEN="your-admin-token"
TRUSTED_KEYS_JSON='[{"id":"...","public_key":"..."}]'
STORACHA_TOKEN="your-storacha-token"
HYBRID_BASE="https://project-beacon-production.up.railway.app"
OPENAI_API_KEY="sk-..."

# Set all secrets in one command to minimize restarts
flyctl secrets set -a beacon-runner-production \
  DATABASE_URL="$DATABASE_URL" \
  REDIS_URL="$REDIS_URL" \
  ADMIN_TOKEN="$ADMIN_TOKEN" \
  TRUSTED_KEYS_JSON="$TRUSTED_KEYS_JSON" \
  STORACHA_TOKEN="$STORACHA_TOKEN" \
  HYBRID_BASE="$HYBRID_BASE" \
  OPENAI_API_KEY="$OPENAI_API_KEY"

echo ""
echo "✅ Secrets set successfully!"
echo ""
echo "🔍 Verifying deployment..."
sleep 10
curl -I https://beacon-runner-production.fly.dev/health
