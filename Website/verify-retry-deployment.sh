#!/bin/bash

echo "=== Retry Feature Deployment Verification ==="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "1. Checking Runner App Health..."
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" https://beacon-runner-change-me.fly.dev/api/v1/health)
if [ "$HEALTH" = "200" ]; then
    echo -e "${GREEN}✓${NC} Runner app is healthy (HTTP $HEALTH)"
else
    echo -e "${RED}✗${NC} Runner app health check failed (HTTP $HEALTH)"
fi

echo ""
echo "2. Testing Retry Endpoint (expect 404 for non-existent execution)..."
RETRY_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST https://beacon-runner-change-me.fly.dev/api/v1/executions/999999/retry-question \
  -H "Content-Type: application/json" \
  -d '{"region": "us-east", "question_index": 0}')

HTTP_CODE=$(echo "$RETRY_RESPONSE" | tail -n1)
BODY=$(echo "$RETRY_RESPONSE" | head -n-1)

if [ "$HTTP_CODE" = "404" ]; then
    echo -e "${GREEN}✓${NC} Retry endpoint exists and returns 404 for non-existent execution"
    echo "   Response: $BODY"
elif [ "$HTTP_CODE" = "000" ]; then
    echo -e "${RED}✗${NC} Retry endpoint not responding (timeout or connection error)"
else
    echo -e "${YELLOW}?${NC} Retry endpoint returned HTTP $HTTP_CODE"
    echo "   Response: $BODY"
fi

echo ""
echo "3. Checking Portal Deployment..."
PORTAL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" https://projectbeacon.netlify.app)
if [ "$PORTAL_STATUS" = "200" ]; then
    echo -e "${GREEN}✓${NC} Portal is accessible (HTTP $PORTAL_STATUS)"
else
    echo -e "${RED}✗${NC} Portal check failed (HTTP $PORTAL_STATUS)"
fi

echo ""
echo "4. Checking if LiveProgressTable.jsx was deployed..."
PORTAL_JS=$(curl -s https://projectbeacon.netlify.app | grep -o "LiveProgressTable" | head -1)
if [ -n "$PORTAL_JS" ]; then
    echo -e "${GREEN}✓${NC} Portal appears to have React components loaded"
else
    echo -e "${YELLOW}?${NC} Could not verify LiveProgressTable in portal"
fi

echo ""
echo "=== Next Steps ==="
echo "1. Visit https://projectbeacon.netlify.app"
echo "2. Submit a job or find an existing job with failed questions"
echo "3. Expand a region in the Live Progress Table"
echo "4. Look for yellow 'Retry' buttons on failed questions"
echo "5. Click 'Retry' and verify the toast notification appears"
echo ""
