#!/bin/bash
# Test RunPod APAC endpoints

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check environment variables
if [ -z "$RUNPOD_API_KEY" ]; then
    echo -e "${RED}Error: RUNPOD_API_KEY not set${NC}"
    exit 1
fi

echo -e "${BLUE}=== Testing RunPod APAC Endpoints ===${NC}"
echo ""

# Test function
test_endpoint() {
    local name=$1
    local endpoint_id=$2
    local model=$3
    
    if [ -z "$endpoint_id" ]; then
        echo -e "${YELLOW}⚠ Skipping $name (endpoint not configured)${NC}"
        return
    fi
    
    echo -e "${BLUE}Testing $name ($model)...${NC}"
    
    response=$(curl -s -X POST \
        "https://api.runpod.ai/v2/${endpoint_id}/runsync" \
        -H "Authorization: Bearer ${RUNPOD_API_KEY}" \
        -H "Content-Type: application/json" \
        -d "{
            \"input\": {
                \"prompt\": \"What is artificial intelligence? Answer in one sentence.\",
                \"temperature\": 0.1,
                \"max_tokens\": 50
            }
        }")
    
    status=$(echo "$response" | jq -r '.status // "UNKNOWN"')
    
    if [ "$status" = "COMPLETED" ]; then
        success=$(echo "$response" | jq -r '.output.success // false')
        inference_time=$(echo "$response" | jq -r '.output.inference_time // 0')
        response_text=$(echo "$response" | jq -r '.output.response // "N/A"' | head -c 100)
        
        if [ "$success" = "true" ]; then
            echo -e "${GREEN}✅ SUCCESS${NC}"
            echo "   Inference time: ${inference_time}s"
            echo "   Response: ${response_text}..."
        else
            error=$(echo "$response" | jq -r '.output.error // "Unknown error"')
            echo -e "${RED}❌ FAILED${NC}"
            echo "   Error: $error"
        fi
    else
        echo -e "${RED}❌ FAILED${NC}"
        echo "   Status: $status"
        echo "   Response: $response"
    fi
    
    echo ""
}

# Test all endpoints
test_endpoint "Llama 3.2-1B" "$RUNPOD_APAC_LLAMA_ENDPOINT" "llama3.2-1b"
test_endpoint "Mistral 7B" "$RUNPOD_APAC_MISTRAL_ENDPOINT" "mistral-7b"
test_endpoint "Qwen 2.5-1.5B" "$RUNPOD_APAC_QWEN_ENDPOINT" "qwen2.5-1.5b"

echo -e "${BLUE}=== Test Complete ===${NC}"
echo ""
echo "To test via hybrid router:"
echo "  curl https://project-beacon-production.up.railway.app/inference \\"
echo "    -d '{\"model\": \"llama3.2-1b\", \"prompt\": \"test\", \"region_preference\": \"asia-pacific\"}'"
echo ""
