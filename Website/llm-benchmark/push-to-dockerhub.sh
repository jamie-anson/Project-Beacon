#!/bin/bash
set -e

echo "Pushing LLM Benchmark Containers to Docker Hub"
echo "=============================================="

# Get Docker Hub username
DOCKER_USER="jamieanson"

# Re-tag containers for Docker Hub
echo "Tagging containers for Docker Hub..."

docker tag beacon/llama-3.2-1b:latest ${DOCKER_USER}/beacon-llama-3.2-1b:latest
docker tag beacon/llama-3.2-1b:latest ${DOCKER_USER}/beacon-llama-3.2-1b:v1.0.0

docker tag beacon/qwen-2.5-1.5b:latest ${DOCKER_USER}/beacon-qwen-2.5-1.5b:latest
docker tag beacon/qwen-2.5-1.5b:latest ${DOCKER_USER}/beacon-qwen-2.5-1.5b:v1.0.0

docker tag beacon/mistral-7b:latest ${DOCKER_USER}/beacon-mistral-7b:latest
docker tag beacon/mistral-7b:latest ${DOCKER_USER}/beacon-mistral-7b:v1.0.0

echo "Pushing containers to Docker Hub..."

# Push Llama 3.2-1B
echo "Pushing Llama 3.2-1B container..."
docker push ${DOCKER_USER}/beacon-llama-3.2-1b:latest
docker push ${DOCKER_USER}/beacon-llama-3.2-1b:v1.0.0

# Push Qwen 2.5-1.5B
echo "Pushing Qwen 2.5-1.5B container..."
docker push ${DOCKER_USER}/beacon-qwen-2.5-1.5b:latest
docker push ${DOCKER_USER}/beacon-qwen-2.5-1.5b:v1.0.0

# Push Mistral 7B
echo "Pushing Mistral 7B container..."
docker push ${DOCKER_USER}/beacon-mistral-7b:latest
docker push ${DOCKER_USER}/beacon-mistral-7b:v1.0.0

echo "All containers pushed successfully!"
echo ""
echo "Available containers:"
echo "  - ${DOCKER_USER}/beacon-llama-3.2-1b:latest"
echo "  - ${DOCKER_USER}/beacon-llama-3.2-1b:v1.0.0"
echo "  - ${DOCKER_USER}/beacon-qwen-2.5-1.5b:latest"
echo "  - ${DOCKER_USER}/beacon-qwen-2.5-1.5b:v1.0.0"
echo "  - ${DOCKER_USER}/beacon-mistral-7b:latest"
echo "  - ${DOCKER_USER}/beacon-mistral-7b:v1.0.0"
