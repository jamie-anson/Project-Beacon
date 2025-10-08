#!/bin/bash
# Build and push RunPod Docker images for APAC region

set -e

# Configuration
DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-your-username}"
REGION="apac"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Building RunPod Images for APAC ===${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running${NC}"
    exit 1
fi

# Check if logged in to Docker Hub
if ! docker info | grep -q "Username"; then
    echo -e "${BLUE}Logging in to Docker Hub...${NC}"
    docker login
fi

cd "$(dirname "$0")/../modal-deployment"

# Build Llama 3.2-1B
echo -e "${GREEN}Building Llama 3.2-1B (for x86_64/amd64)...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg MODEL_NAME=llama3.2-1b \
  --build-arg HF_MODEL_ID=meta-llama/Llama-3.2-1B-Instruct \
  -f Dockerfile.runpod \
  -t ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest \
  --load \
  .

# Build Mistral 7B
echo -e "${GREEN}Building Mistral 7B (for x86_64/amd64)...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg MODEL_NAME=mistral-7b \
  --build-arg HF_MODEL_ID=mistralai/Mistral-7B-Instruct-v0.3 \
  -f Dockerfile.runpod \
  -t ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest \
  --load \
  .

# Build Qwen 2.5-1.5B
echo -e "${GREEN}Building Qwen 2.5-1.5B (for x86_64/amd64)...${NC}"
docker buildx build \
  --platform linux/amd64 \
  --build-arg MODEL_NAME=qwen2.5-1.5b \
  --build-arg HF_MODEL_ID=Qwen/Qwen2.5-1.5B-Instruct \
  -f Dockerfile.runpod \
  -t ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest \
  --load \
  .

echo ""
echo -e "${BLUE}=== Pushing Images to Docker Hub ===${NC}"
echo ""

# Push all images
echo -e "${GREEN}Pushing Llama 3.2-1B...${NC}"
docker push ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest

echo -e "${GREEN}Pushing Mistral 7B...${NC}"
docker push ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest

echo -e "${GREEN}Pushing Qwen 2.5-1.5B...${NC}"
docker push ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest

echo ""
echo -e "${GREEN}=== Build Complete ===${NC}"
echo ""
echo "Images pushed:"
echo "  - ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest"
echo "  - ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest"
echo "  - ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest"
echo ""
echo "Next steps:"
echo "  1. Go to https://www.runpod.io/console/serverless"
echo "  2. Create endpoints for each image"
echo "  3. Copy endpoint IDs"
echo "  4. Update Railway environment variables"
echo ""
