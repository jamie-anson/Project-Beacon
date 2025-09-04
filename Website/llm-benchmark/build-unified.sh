#!/bin/bash
set -e

echo "Building Unified LLM Benchmark Containers for Project Beacon"
echo "==========================================================="

# Copy unified files to each container directory
echo "Copying unified benchmark files..."
cp benchmark.py questions.json questions-phase2.json scoring-unified.py advanced-scoring.py response-clustering.py export-data.py llama-3.2-1b/
cp benchmark.py questions.json questions-phase2.json scoring-unified.py advanced-scoring.py response-clustering.py export-data.py qwen-2.5-1.5b/
cp benchmark.py questions.json questions-phase2.json scoring-unified.py advanced-scoring.py response-clustering.py export-data.py mistral-7b/

# Rename scoring-unified.py to scoring.py in each directory
mv llama-3.2-1b/scoring-unified.py llama-3.2-1b/scoring.py
mv qwen-2.5-1.5b/scoring-unified.py qwen-2.5-1.5b/scoring.py
mv mistral-7b/scoring-unified.py mistral-7b/scoring.py

# Build unified Llama 3.2-1B container
echo "Building unified Llama 3.2-1B container..."
cd llama-3.2-1b
docker build -t beacon/llama-3.2-1b:latest .
cd ..

# Build unified Qwen 2.5-1.5B container
echo "Building unified Qwen 2.5-1.5B container..."
cd qwen-2.5-1.5b
docker build -t beacon/qwen-2.5-1.5b:latest .
cd ..

# Build unified Mistral 7B container
echo "Building unified Mistral 7B container..."
cd mistral-7b
docker build -t beacon/mistral-7b:latest .
cd ..

# Clean up copied files
echo "Cleaning up..."
rm llama-3.2-1b/benchmark.py llama-3.2-1b/questions.json llama-3.2-1b/questions-phase2.json llama-3.2-1b/scoring.py llama-3.2-1b/advanced-scoring.py llama-3.2-1b/response-clustering.py llama-3.2-1b/export-data.py
rm qwen-2.5-1.5b/benchmark.py qwen-2.5-1.5b/questions.json qwen-2.5-1.5b/questions-phase2.json qwen-2.5-1.5b/scoring.py qwen-2.5-1.5b/advanced-scoring.py qwen-2.5-1.5b/response-clustering.py qwen-2.5-1.5b/export-data.py
rm mistral-7b/benchmark.py mistral-7b/questions.json mistral-7b/questions-phase2.json mistral-7b/scoring.py mistral-7b/advanced-scoring.py mistral-7b/response-clustering.py mistral-7b/export-data.py

echo "All unified containers built successfully!"
echo ""
echo "Available unified containers:"
echo "  - beacon/llama-3.2-1b:latest"
echo "  - beacon/qwen-2.5-1.5b:latest"
echo "  - beacon/mistral-7b:latest"
echo ""
echo "Usage examples:"
echo "  # Simple mode (Phase 1 equivalent)"
echo "  docker run -e BENCHMARK_MODE=simple beacon/llama-3.2-1b:latest"
echo ""
echo "  # Advanced mode (Phase 2 equivalent)"
echo "  docker run -e BENCHMARK_MODE=advanced beacon/llama-3.2-1b:latest"
