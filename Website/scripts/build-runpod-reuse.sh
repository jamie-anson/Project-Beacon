#!/bin/bash
# Build RunPod images by reusing existing model images

set -e

DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-freelancejamie}"
REGION="apac"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}=== Building RunPod Images (Reusing Existing Models) ===${NC}"
echo ""

cd "$(dirname "$0")/../modal-deployment"

# Build Llama 3.2-1B (reusing existing image)
echo -e "${GREEN}Building Llama 3.2-1B (reusing jamieanson/beacon-llama-3.2-1b)...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg SOURCE_IMAGE=jamieanson/beacon-llama-3.2-1b:latest \
  --build-arg MODEL_NAME=llama3.2-1b \
  --build-arg HF_MODEL_ID=meta-llama/Llama-3.2-1B-Instruct \
  -f Dockerfile.runpod-reuse \
  -t ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest \
  --load \
  .

echo -e "${GREEN}Building Mistral 7B (reusing jamieanson/beacon-mistral-7b)...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg SOURCE_IMAGE=jamieanson/beacon-mistral-7b:latest \
  --build-arg MODEL_NAME=mistral-7b \
  --build-arg HF_MODEL_ID=mistralai/Mistral-7B-Instruct-v0.3 \
  -f Dockerfile.runpod-reuse \
  -t ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest \
  --load \
  .

echo -e "${GREEN}Building Qwen 2.5-1.5B (reusing jamieanson/beacon-qwen-2.5-1.5b)...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg SOURCE_IMAGE=jamieanson/beacon-qwen-2.5-1.5b:latest \
  --build-arg MODEL_NAME=qwen2.5-1.5b \
  --build-arg HF_MODEL_ID=Qwen/Qwen2.5-1.5B-Instruct \
  -f Dockerfile.runpod-reuse \
  -t ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest \
  --load \
  .

echo ""
echo -e "${YELLOW}Note: Models will be downloaded on first inference if not found in cache${NC}"
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
echo ""
echo "Next: Deploy these images to RunPod"
