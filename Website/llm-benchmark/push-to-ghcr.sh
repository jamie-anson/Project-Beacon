#!/bin/bash
set -e

echo "Pushing LLM Benchmark Containers to GHCR"
echo "========================================"

# Re-tag containers with correct GHCR paths
echo "Tagging containers for GHCR..."

# Use the project repository name format that should work with existing permissions
docker tag beacon/llama-3.2-1b:latest ghcr.io/jamie-anson/project-beacon:llama-3.2-1b-latest
docker tag beacon/llama-3.2-1b:latest ghcr.io/jamie-anson/project-beacon:llama-3.2-1b-v1.0.0

docker tag beacon/qwen-2.5-1.5b:latest ghcr.io/jamie-anson/project-beacon:qwen-2.5-1.5b-latest
docker tag beacon/qwen-2.5-1.5b:latest ghcr.io/jamie-anson/project-beacon:qwen-2.5-1.5b-v1.0.0

docker tag beacon/mistral-7b:latest ghcr.io/jamie-anson/project-beacon:mistral-7b-latest
docker tag beacon/mistral-7b:latest ghcr.io/jamie-anson/project-beacon:mistral-7b-v1.0.0

echo "Pushing containers to GHCR..."

# Push Llama 3.2-1B
echo "Pushing Llama 3.2-1B container..."
docker push ghcr.io/jamie-anson/project-beacon:llama-3.2-1b-latest
docker push ghcr.io/jamie-anson/project-beacon:llama-3.2-1b-v1.0.0

# Push Qwen 2.5-1.5B
echo "Pushing Qwen 2.5-1.5B container..."
docker push ghcr.io/jamie-anson/project-beacon:qwen-2.5-1.5b-latest
docker push ghcr.io/jamie-anson/project-beacon:qwen-2.5-1.5b-v1.0.0

# Push Mistral 7B
echo "Pushing Mistral 7B container..."
docker push ghcr.io/jamie-anson/project-beacon:mistral-7b-latest
docker push ghcr.io/jamie-anson/project-beacon:mistral-7b-v1.0.0

echo "All containers pushed successfully!"
echo ""
echo "Available containers:"
echo "  - ghcr.io/jamie-anson/project-beacon:llama-3.2-1b-latest"
echo "  - ghcr.io/jamie-anson/project-beacon:llama-3.2-1b-v1.0.0"
echo "  - ghcr.io/jamie-anson/project-beacon:qwen-2.5-1.5b-latest"
echo "  - ghcr.io/jamie-anson/project-beacon:qwen-2.5-1.5b-v1.0.0"
echo "  - ghcr.io/jamie-anson/project-beacon:mistral-7b-latest"
echo "  - ghcr.io/jamie-anson/project-beacon:mistral-7b-v1.0.0"
