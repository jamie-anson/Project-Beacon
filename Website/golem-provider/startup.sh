#!/bin/bash

# Golem Provider Node Startup Script
set -e

echo "=== Golem Provider Node Startup ==="

# Simple installation check - if Yagna not found, just exit with clear message
if [ ! -f "/home/golem/.local/bin/yagna" ]; then
    echo "Yagna not found. Installing from package..."
    
    cd /tmp
    wget -q https://github.com/golemfactory/yagna/releases/download/v0.17.3/golem-provider_v0.17.3_amd64.deb
    
    if [ -f "golem-provider_v0.17.3_amd64.deb" ]; then
        dpkg-deb -x golem-provider_v0.17.3_amd64.deb yagna-extract
        if [ -d "yagna-extract/usr/bin" ]; then
            cp yagna-extract/usr/bin/* /home/golem/.local/bin/
            chmod +x /home/golem/.local/bin/*
            rm -rf golem-provider_v0.17.3_amd64.deb yagna-extract
            echo "Yagna installed successfully"
        else
            echo "Failed to extract Yagna - package structure unexpected"
            ls -la yagna-extract/ || echo "Extract directory not found"
            exit 1
        fi
    else
        echo "Failed to download Yagna package"
        exit 1
    fi
fi

# Verify installation
if [ ! -f "/home/golem/.local/bin/yagna" ]; then
    echo "ERROR: Yagna installation verification failed"
    exit 1
fi

echo "Yagna found at: /home/golem/.local/bin/yagna"

# Start Yagna service
echo "Starting Yagna service..."
yagna service run --api-allow-origin='*' &
YAGNA_PID=$!

echo "Yagna service started with PID: $YAGNA_PID"
echo "Waiting for Yagna GSB socket /tmp/yagna.sock ..."

# Readiness loop: wait up to ~60s for GSB socket or id show to succeed
READY=0
for i in $(seq 1 30); do
    if [ -S "/tmp/yagna.sock" ]; then
        READY=1
        break
    fi
    if /home/golem/.local/bin/yagna id show >/dev/null 2>&1; then
        READY=1
        break
    fi
    echo "[wait:$i] yagna not ready yet; sleeping 2s"
    sleep 2
done

if [ "$READY" -ne 1 ]; then
    echo "WARNING: yagna not ready after wait; continuing, but CLI calls may fail briefly"
else
    echo "Yagna GSB is ready"
fi

# Create app key if needed
echo "Creating app key..."
yagna app-key create requestor 2>/dev/null || echo "App key already exists or creation failed"

# Show node info
echo "=== Node Information ==="
/home/golem/.local/bin/yagna id show || echo "Failed to get node info (will likely be ready shortly)"

echo "=== Yagna service is running ==="
echo "Starting FastAPI health/inference server on :8080..."

# Ensure Python can locate app.py and run uvicorn via module
export PYTHONPATH="/home/golem:${PYTHONPATH}"
PY_CMD="python3 -m uvicorn app:app --host 0.0.0.0 --port 8080"
bash -lc "$PY_CMD" &
API_PID=$!
echo "FastAPI started with PID: $API_PID"

echo "Container will keep running. Use 'docker compose logs -f' to monitor."

# Optionally start Golem provider service in background (non-blocking)
echo "Starting Golem provider daemon (ya-provider run) in background..."
nohup /home/golem/.local/bin/ya-provider run > /home/golem/golem-logs/provider.log 2>&1 &
PROV_PID=$!
echo "ya-provider started with PID: $PROV_PID (logs: /home/golem/golem-logs/provider.log)"

# Keep container alive (wait on Yagna)
wait $YAGNA_PID || true
