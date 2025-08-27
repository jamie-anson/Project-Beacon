#!/bin/bash
set -e

echo "Testing LLM Benchmark Containers"
echo "================================"

# Create results directory
mkdir -p results

# Test each container
models=("llama-3.2-1b" "qwen-2.5-1.5b" "mistral-7b")

for model in "${models[@]}"; do
    echo ""
    echo "Testing beacon/$model:latest..."
    echo "----------------------------------------"
    
    # Run container with timeout
    timeout 600s docker run --rm \
        -v $(pwd)/results:/tmp \
        beacon/$model:latest || {
        echo "⚠️  Container $model timed out or failed"
        continue
    }
    
    # Check if results were generated
    if [ -f "results/benchmark_results.json" ]; then
        echo "✅ $model benchmark completed successfully"
        
        # Run scoring
        python3 scoring.py results/benchmark_results.json
        
        # Rename results for this model
        mv results/benchmark_results.json results/${model}_results.json
        if [ -f "results/benchmark_results_scored.json" ]; then
            mv results/benchmark_results_scored.json results/${model}_results_scored.json
        fi
    else
        echo "❌ $model benchmark failed - no results generated"
    fi
done

echo ""
echo "Testing completed! Check results/ directory for outputs."
