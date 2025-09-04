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
echo "Waiting for service to be ready..."
sleep 15

# Create app key if needed
echo "Creating app key..."
yagna app-key create requestor 2>/dev/null || echo "App key already exists or creation failed"

# Show node info
echo "=== Node Information ==="
yagna id show || echo "Failed to get node info"

echo "=== Yagna service is running ==="
echo "Container will keep running. Use 'docker compose logs -f' to monitor."

# Keep container alive
tail -f /dev/null
