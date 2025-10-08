#!/bin/bash
# Test RunPod Qwen endpoint

set -e

ENDPOINT_ID="${RUNPOD_APAC_QWEN_ENDPOINT:-nd7eqzpfnbwvsy}"
RUNPOD_API_KEY="${RUNPOD_API_KEY}"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=== Testing RunPod Qwen Endpoint ===${NC}"
echo ""
echo "Endpoint ID: $ENDPOINT_ID"
echo ""

# Test inference
echo -e "${BLUE}Sending test request...${NC}"
response=$(curl -s -X POST "https://api.runpod.ai/v2/${ENDPOINT_ID}/runsync" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${RUNPOD_API_KEY}" \
  -d '{
    "input": {
      "prompt": "What is artificial intelligence? Answer in one sentence.",
      "temperature": 0.1,
      "max_tokens": 50
    }
  }')

echo ""
echo -e "${BLUE}Response:${NC}"
echo "$response" | jq '.'

# Check if successful
status=$(echo "$response" | jq -r '.status // "UNKNOWN"')
success=$(echo "$response" | jq -r '.output.success // false')

echo ""
if [ "$status" = "COMPLETED" ] && [ "$success" = "true" ]; then
  inference_time=$(echo "$response" | jq -r '.output.inference_time // 0')
  response_text=$(echo "$response" | jq -r '.output.response // "N/A"')
  
  echo -e "${GREEN}✅ SUCCESS${NC}"
  echo "Inference time: ${inference_time}s"
  echo "Response: $response_text"
else
  echo -e "${RED}❌ FAILED${NC}"
  echo "Status: $status"
  error=$(echo "$response" | jq -r '.error // .output.error // "Unknown error"')
  echo "Error: $error"
fi

echo ""
echo -e "${BLUE}=== Test Complete ===${NC}"
