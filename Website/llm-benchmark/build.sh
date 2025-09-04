#!/bin/bash
set -e

echo "Building LLM Benchmark Containers for Project Beacon"
echo "===================================================="

# Copy shared files to each container directory
echo "Copying shared files..."
cp benchmark.py questions.json scoring.py llama-3.2-1b/
cp benchmark.py questions.json scoring.py qwen-2.5-1.5b/
cp benchmark.py questions.json scoring.py mistral-7b/

# Build Llama 3.2-1B container
echo "Building Llama 3.2-1B container..."
cd llama-3.2-1b
docker build -t beacon/llama-3.2-1b:latest .
cd ..

# Build Qwen 2.5-1.5B container
echo "Building Qwen 2.5-1.5B container..."
cd qwen-2.5-1.5b
docker build -t beacon/qwen-2.5-1.5b:latest .
cd ..

# Build Mistral 7B container
echo "Building Mistral 7B container..."
cd mistral-7b
docker build -t beacon/mistral-7b:latest .
cd ..

# Clean up copied files
echo "Cleaning up..."
rm llama-3.2-1b/benchmark.py llama-3.2-1b/questions.json llama-3.2-1b/scoring.py
rm qwen-2.5-1.5b/benchmark.py qwen-2.5-1.5b/questions.json qwen-2.5-1.5b/scoring.py
rm mistral-7b/benchmark.py mistral-7b/questions.json mistral-7b/scoring.py

echo "All containers built successfully!"
echo ""
echo "Available containers:"
echo "  - beacon/llama-3.2-1b:latest"
echo "  - beacon/qwen-2.5-1.5b:latest"
echo "  - beacon/mistral-7b:latest"
echo ""
echo "To run a container:"
echo "  docker run --rm -v \$(pwd)/results:/tmp beacon/llama-3.2-1b:latest"
