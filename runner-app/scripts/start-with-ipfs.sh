#!/bin/bash
set -e

# Set IPFS path
export IPFS_PATH=/app/.ipfs

# Initialize IPFS if not already done
if [ ! -d "$IPFS_PATH" ]; then
    echo "Initializing IPFS..."
    ipfs init --profile server
    
    # Configure IPFS for container environment
    ipfs config Addresses.API /ip4/0.0.0.0/tcp/5001
    ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080
    ipfs config --json API.HTTPHeaders.Access-Control-Allow-Origin '["*"]'
    ipfs config --json API.HTTPHeaders.Access-Control-Allow-Methods '["PUT", "POST", "GET"]'
    ipfs config --json Swarm.ConnMgr.LowWater 50
    ipfs config --json Swarm.ConnMgr.HighWater 200
    echo "IPFS initialized successfully!"
fi

# Start IPFS daemon in background
echo "Starting IPFS daemon..."
ipfs daemon --enable-gc --migrate &
IPFS_PID=$!

# Wait for IPFS to be ready
echo "Waiting for IPFS to be ready..."
for i in {1..60}; do
    if curl -s http://localhost:5001/api/v0/version >/dev/null 2>&1; then
        echo "IPFS daemon is ready!"
        break
    fi
    if [ $i -eq 60 ]; then
        echo "IPFS daemon failed to start within 60 seconds"
        exit 1
    fi
    sleep 1
done

# Run trusted keys initialization
echo "Initializing trusted keys..."
bash /app/scripts/init-trusted-keys.sh || true

# Start the runner app
echo "Starting Project Beacon Runner..."
exec /app/runner
