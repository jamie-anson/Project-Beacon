#!/bin/bash

# Emergency Provider Health Check Script
# Principal Engineer Emergency Response Tool

set -e

echo "🚨 EMERGENCY PROVIDER HEALTH CHECK 🚨"
echo "======================================"
echo "Timestamp: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Railway Hybrid Router URL
RAILWAY_URL="https://project-beacon-production.up.railway.app"

echo "1. Testing Railway Hybrid Router Health..."
echo "----------------------------------------"

# Test basic health
echo -n "Basic Health Check: "
HEALTH_RESPONSE=$(curl -s -w "%{http_code}" "$RAILWAY_URL/health" -o /tmp/health_response.json)
HTTP_CODE=${HEALTH_RESPONSE: -3}

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ HEALTHY${NC}"
    echo "Response: $(cat /tmp/health_response.json)"
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "Response: $(cat /tmp/health_response.json 2>/dev/null || echo 'No response')"
fi

echo ""

# Test provider discovery
echo -n "Provider Discovery: "
PROVIDERS_RESPONSE=$(curl -s -w "%{http_code}" "$RAILWAY_URL/providers" -o /tmp/providers_response.json)
HTTP_CODE=${PROVIDERS_RESPONSE: -3}

if [ "$HTTP_CODE" = "200" ]; then
    PROVIDER_COUNT=$(cat /tmp/providers_response.json | jq '.providers | length' 2>/dev/null || echo "0")
    if [ "$PROVIDER_COUNT" = "0" ]; then
        echo -e "${RED}❌ ZERO PROVIDERS CONFIGURED${NC}"
        echo "Response: $(cat /tmp/providers_response.json)"
    else
        echo -e "${GREEN}✅ $PROVIDER_COUNT PROVIDERS FOUND${NC}"
        echo "Providers:"
        cat /tmp/providers_response.json | jq '.providers[] | {name, type, region, healthy}' 2>/dev/null || echo "Failed to parse providers"
    fi
else
    echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    echo "Response: $(cat /tmp/providers_response.json 2>/dev/null || echo 'No response')"
fi

echo ""

# Test environment variables endpoint
echo -n "Environment Check: "
ENV_RESPONSE=$(curl -s -w "%{http_code}" "$RAILWAY_URL/env" -o /tmp/env_response.json)
HTTP_CODE=${ENV_RESPONSE: -3}

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✅ ACCESSIBLE${NC}"
    echo "Environment Variables:"
    cat /tmp/env_response.json | jq . 2>/dev/null || echo "Failed to parse environment"
else
    echo -e "${YELLOW}⚠️  NOT AVAILABLE (HTTP $HTTP_CODE)${NC}"
    echo "Note: /env endpoint may not exist in Go server"
fi

echo ""

# Test regional provider endpoints
echo "2. Testing Regional Provider Endpoints..."
echo "----------------------------------------"

REGIONS=("us-east" "eu-west" "asia-pacific")

for region in "${REGIONS[@]}"; do
    echo -n "Region $region: "
    REGION_RESPONSE=$(curl -s -w "%{http_code}" "$RAILWAY_URL/providers?region=$region" -o /tmp/region_${region}_response.json)
    HTTP_CODE=${REGION_RESPONSE: -3}
    
    if [ "$HTTP_CODE" = "200" ]; then
        REGION_PROVIDER_COUNT=$(cat /tmp/region_${region}_response.json | jq '.providers | length' 2>/dev/null || echo "0")
        if [ "$REGION_PROVIDER_COUNT" = "0" ]; then
            echo -e "${RED}❌ NO PROVIDERS${NC}"
        else
            echo -e "${GREEN}✅ $REGION_PROVIDER_COUNT PROVIDERS${NC}"
        fi
    else
        echo -e "${RED}❌ FAILED (HTTP $HTTP_CODE)${NC}"
    fi
done

echo ""

# Test Golem provider endpoints directly
echo "3. Testing Golem Provider Endpoints Directly..."
echo "----------------------------------------------"

GOLEM_ENDPOINTS=(
    "https://beacon-golem-us.fly.dev"
    "https://beacon-golem-eu.fly.dev" 
    "https://beacon-golem-apac.fly.dev"
)

for endpoint in "${GOLEM_ENDPOINTS[@]}"; do
    echo -n "$(basename $endpoint): "
    GOLEM_RESPONSE=$(curl -s -w "%{http_code}" "$endpoint/health" -o /tmp/golem_response.json --max-time 10)
    HTTP_CODE=${GOLEM_RESPONSE: -3}
    
    if [ "$HTTP_CODE" = "200" ]; then
        echo -e "${GREEN}✅ HEALTHY${NC}"
    else
        echo -e "${RED}❌ UNHEALTHY (HTTP $HTTP_CODE)${NC}"
    fi
done

echo ""

# Summary and recommendations
echo "4. Summary and Recommendations..."
echo "--------------------------------"

# Check if Railway is running the correct service
if grep -q '"providers"' /tmp/providers_response.json 2>/dev/null; then
    echo -e "${GREEN}✅ Railway running Python hybrid router${NC}"
else
    echo -e "${RED}❌ Railway running wrong service (likely Go server)${NC}"
    echo "🔧 ACTION REQUIRED: Deploy Python hybrid router to Railway"
fi

# Check provider configuration
TOTAL_PROVIDERS=$(cat /tmp/providers_response.json | jq '.providers | length' 2>/dev/null || echo "0")
if [ "$TOTAL_PROVIDERS" = "0" ]; then
    echo -e "${RED}❌ No providers configured${NC}"
    echo "🔧 ACTION REQUIRED: Configure provider environment variables"
else
    echo -e "${GREEN}✅ $TOTAL_PROVIDERS providers configured${NC}"
fi

echo ""
echo "🚀 NEXT STEPS:"
echo "1. If Railway is running wrong service: railway up flyio-deployment/hybrid_router.py"
echo "2. If no providers: Configure GOLEM_*_ENDPOINT environment variables"
echo "3. Test again with: ./scripts/emergency-provider-check.sh"
echo ""
echo "📊 Full logs available in /tmp/*_response.json"

# Cleanup
rm -f /tmp/health_response.json /tmp/providers_response.json /tmp/env_response.json
rm -f /tmp/region_*_response.json /tmp/golem_response.json

echo "Emergency check completed at $(date)"
