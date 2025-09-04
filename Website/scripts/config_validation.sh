#!/bin/bash

# Configuration Validation Script for benchmark.py

# Check for benchmark.py file
if [ ! -f "llm-benchmark/benchmark.py" ]; then
    echo "Error: benchmark.py not found."
    exit 1
fi

# Check timeout setting in benchmark.py
if ! grep -q "timeout=120" llm-benchmark/benchmark.py; then
    echo "Warning: Timeout setting is not 120 seconds."
else
    echo "Timeout setting is correct."
fi

# Check mode setting in benchmark.py
if ! grep -q "mode: str = \"simple\"" llm-benchmark/benchmark.py; then
    echo "Warning: Default mode is not set to 'simple'."
else
    echo "Default mode is set to 'simple'."
fi

# Verify Ollama URL
if ! grep -q "ollama_url = \"http://localhost:11434\"" llm-benchmark/benchmark.py; then
    echo "Warning: Ollama URL is not set to http://localhost:11434."
else
    echo "Ollama URL is correct."
fi
