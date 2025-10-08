#!/bin/bash
# Convert existing images to RunPod format by adding handler

set -e

DOCKERHUB_USERNAME="${DOCKERHUB_USERNAME:-freelancejamie}"
REGION="apac"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Converting Existing Images to RunPod Format ===${NC}"
echo ""

cd "$(dirname "$0")/../modal-deployment"

# Create simple conversion Dockerfile
cat > Dockerfile.convert << 'EOF'
ARG BASE_IMAGE
FROM ${BASE_IMAGE}

# Install runpod package
RUN pip3 install --no-cache-dir runpod>=1.5.0 || pip install --no-cache-dir runpod>=1.5.0

# Copy RunPod handler
COPY runpod_handler.py /app/handler.py

# Set environment
ARG MODEL_NAME
ENV MODEL_NAME=${MODEL_NAME}

WORKDIR /app
CMD ["python3", "-u", "handler.py"]
EOF

# Convert Llama
echo -e "${GREEN}Converting Llama 3.2-1B...${NC}"
docker build \
  --build-arg BASE_IMAGE=beacon/llama-3.2-1b:latest \
  --build-arg MODEL_NAME=llama3.2-1b \
  -f Dockerfile.convert \
  -t ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest \
  .

# Convert Mistral
echo -e "${GREEN}Converting Mistral 7B...${NC}"
docker build \
  --build-arg BASE_IMAGE=beacon/mistral-7b:latest \
  --build-arg MODEL_NAME=mistral-7b \
  -f Dockerfile.convert \
  -t ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest \
  .

# Convert Qwen
echo -e "${GREEN}Converting Qwen 2.5-1.5B...${NC}"
docker build \
  --build-arg BASE_IMAGE=beacon/qwen-2.5-1.5b:latest \
  --build-arg MODEL_NAME=qwen2.5-1.5b \
  -f Dockerfile.convert \
  -t ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest \
  .

echo ""
echo -e "${BLUE}=== Pushing to Docker Hub ===${NC}"
echo ""

docker push ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest &
docker push ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest &
docker push ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest &

wait

echo ""
echo -e "${GREEN}✅ All images converted and pushed!${NC}"
echo ""
echo "Images ready for RunPod:"
echo "  - ${DOCKERHUB_USERNAME}/beacon-llama-${REGION}:latest"
echo "  - ${DOCKERHUB_USERNAME}/beacon-mistral-${REGION}:latest"
echo "  - ${DOCKERHUB_USERNAME}/beacon-qwen-${REGION}:latest"
