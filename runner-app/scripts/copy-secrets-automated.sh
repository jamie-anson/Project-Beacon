#!/bin/bash
# Automated secret copying from beacon-runner-change-me to beacon-runner-production

set -e

echo "🔐 Copying secrets from old app to new production app..."
echo ""

# Get DATABASE_URL
echo "📦 Copying DATABASE_URL..."
DB_URL=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="DATABASE_URL") | .Value // empty')
if [ -n "$DB_URL" ]; then
  flyctl secrets set -a beacon-runner-production DATABASE_URL="$DB_URL"
else
  echo "⚠️  DATABASE_URL not found in old app"
fi

# Get REDIS_URL
echo "📦 Copying REDIS_URL..."
REDIS=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="REDIS_URL") | .Value // empty')
if [ -n "$REDIS" ]; then
  flyctl secrets set -a beacon-runner-production REDIS_URL="$REDIS"
else
  echo "⚠️  REDIS_URL not found in old app"
fi

# Get ADMIN_TOKEN
echo "📦 Copying ADMIN_TOKEN..."
ADMIN=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="ADMIN_TOKEN") | .Value // empty')
if [ -n "$ADMIN" ]; then
  flyctl secrets set -a beacon-runner-production ADMIN_TOKEN="$ADMIN"
else
  echo "⚠️  ADMIN_TOKEN not found in old app"
fi

# Get TRUSTED_KEYS_JSON
echo "📦 Copying TRUSTED_KEYS_JSON..."
KEYS=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="TRUSTED_KEYS_JSON") | .Value // empty')
if [ -n "$KEYS" ]; then
  flyctl secrets set -a beacon-runner-production TRUSTED_KEYS_JSON="$KEYS"
else
  echo "⚠️  TRUSTED_KEYS_JSON not found in old app"
fi

# Get STORACHA_TOKEN
echo "📦 Copying STORACHA_TOKEN..."
STORACHA=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="STORACHA_TOKEN") | .Value // empty')
if [ -n "$STORACHA" ]; then
  flyctl secrets set -a beacon-runner-production STORACHA_TOKEN="$STORACHA"
else
  echo "⚠️  STORACHA_TOKEN not found in old app"
fi

# Get HYBRID_BASE
echo "📦 Copying HYBRID_BASE..."
HYBRID=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="HYBRID_BASE") | .Value // empty')
if [ -n "$HYBRID" ]; then
  flyctl secrets set -a beacon-runner-production HYBRID_BASE="$HYBRID"
else
  echo "⚠️  HYBRID_BASE not found in old app"
fi

# Get OPENAI_API_KEY
echo "📦 Copying OPENAI_API_KEY..."
OPENAI=$(flyctl secrets list -a beacon-runner-change-me -j | jq -r '.[] | select(.Name=="OPENAI_API_KEY") | .Value // empty')
if [ -n "$OPENAI" ]; then
  flyctl secrets set -a beacon-runner-production OPENAI_API_KEY="$OPENAI"
else
  echo "⚠️  OPENAI_API_KEY not found in old app"
fi

echo ""
echo "✅ Secret copying complete!"
echo ""
echo "🔍 Verifying secrets were set..."
flyctl secrets list -a beacon-runner-production
