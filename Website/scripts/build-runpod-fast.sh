#!/bin/bash
# Fast RunPod build using local HF cache

set -e

DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-freelancejamie}"
REGION="apac"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Fast RunPod Build (using local cache) ===${NC}"
echo ""

cd "$(dirname "$0")/../modal-deployment"

# Build with cache mount (much faster!)
echo -e "${GREEN}Building Llama 3.2-1B...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg MODEL_NAME=llama3.2-1b \
  --build-arg HF_MODEL_ID=meta-llama/Llama-3.2-1B-Instruct \
  --cache-from type=registry,ref=${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:cache \
  -f Dockerfile.runpod \
  -t ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest \
  --load \
  .

echo -e "${GREEN}Building Mistral 7B...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg MODEL_NAME=mistral-7b \
  --build-arg HF_MODEL_ID=mistralai/Mistral-7B-Instruct-v0.3 \
  --cache-from type=registry,ref=${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:cache \
  -f Dockerfile.runpod \
  -t ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest \
  --load \
  .

echo -e "${GREEN}Building Qwen 2.5-1.5B...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg MODEL_NAME=qwen2.5-1.5b \
  --build-arg HF_MODEL_ID=Qwen/Qwen2.5-1.5B-Instruct \
  --cache-from type=registry,ref=${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:cache \
  -f Dockerfile.runpod \
  -t ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest \
  --load \
  .

echo ""
echo -e "${BLUE}=== Pushing to Docker Hub ===${NC}"
echo ""

docker push ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest
docker push ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest
docker push ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest

echo ""
echo -e "${GREEN}✅ Done!${NC}"
echo ""
echo "Images:"
echo "  - ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest"
echo "  - ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest"
echo "  - ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest"
