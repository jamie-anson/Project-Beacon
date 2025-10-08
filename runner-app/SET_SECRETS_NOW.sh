#!/bin/bash
# Set production secrets for beacon-runner-production
# Replace the placeholder values with your actual secrets

flyctl secrets set -a beacon-runner-production \
  DATABASE_URL="YOUR_NEON_POSTGRES_URL_HERE" \
  REDIS_URL="YOUR_UPSTASH_REDIS_URL_HERE" \
  ADMIN_TOKENS="YOUR_ADMIN_TOKEN_HERE" \
  TRUSTED_KEYS_JSON='[{"id":"portal","public_key":"xsIE/NE5XmHyA2EPQ/ZMJjeHCsOqDo/eoGaQ83mnnMo="}]' \
  STORACHA_TOKEN="YOUR_STORACHA_TOKEN_HERE" \
  HYBRID_BASE="https://project-beacon-production.up.railway.app" \
  OPENAI_API_KEY="YOUR_OPENAI_KEY_WITH_WRITE_PERMISSIONS_HERE"

echo ""
echo "✅ Secrets set! Waiting for deployment to restart..."
sleep 15

echo "🔍 Testing health endpoint..."
curl -I https://beacon-runner-production.fly.dev/health
