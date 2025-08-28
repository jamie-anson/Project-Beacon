# User Onboarding Checklist

Welcome to Project Beacon! Follow this checklist to ensure a smooth setup process.

## Step 1: Resource Validation
- Run the `enhanced_resource_check.sh` script to validate Docker memory, CPU, and GPU allocations.
- Ensure Docker daemon is running and network connectivity is available.

## Step 2: Configuration Validation
- Run the `config_validation.sh` script to validate settings in `benchmark.py`.
- Check timeout and mode settings for correctness.

## Step 3: Documentation Review
- Review the `resource_troubleshooting.md` guide for resource requirements and troubleshooting steps.

## Step 4: Prewarm Models
- Use the provided Makefile to prewarm models before running benchmarks.
- Commands: `make prewarm-all`, `make prewarm-llama`, `make prewarm-qwen`, `make prewarm-mistral`

## Step 5: Run Benchmarks
- Follow instructions in the README to run benchmarks using Docker Compose.

## Step 6: GPU Setup (optional but recommended)
- Terminals mapping:
  - Terminal A: Yagna daemon
  - Terminal B: Go API server (http://localhost:8090)
  - Terminal C: Actions (curl, tests)
  - Terminal D: Postgres + Redis (docker compose)
  - Terminal E: Cloud infra (flyctl, Upstash, Neon)
  - Terminal G: Benchmarks

- macOS (Apple Silicon) with Metal acceleration:
  - Run Ollama natively on the host to use Metal (Docker on macOS cannot use GPU/ANE):
    - Terminal C:
      ```bash
      # if needed
      brew install ollama
      ollama serve  # starts on :11434 and uses Metal
      ```
    - Terminal G:
      ```bash
      export OLLAMA_URL=http://host.docker.internal:11434
      make -C Website/llm-benchmark run-llama  # or run-qwen / run-mistral
      ```
    - Verify:
      ```bash
      curl -s http://host.docker.internal:11434/api/tags | jq .
      ```

- Remote/NVIDIA GPU (Linux with CUDA or cloud GPU):
  - Start an Ollama GPU server on the GPU host:
    ```bash
    docker run --gpus all -p 11434:11434 --name ollama-gpu ollama/ollama
    ```
  - Terminal G (Benchmarks):
    ```bash
    export OLLAMA_URL=http://<gpu-host-or-ip>:11434
    make -C Website/llm-benchmark run-llama
    ```
  - Verify on GPU host:
    ```bash
    nvidia-smi
    ```

Notes:
- Prefer single benchmark runs first (avoid `make run-all`) to validate GPU path.
- Use quantized models for speed when appropriate.

## Step 7: Troubleshooting
- Refer to the troubleshooting guide for common issues and solutions.

This checklist is designed to help new users set up and validate their environment effectively.
