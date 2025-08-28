#!/bin/bash

# Simple Golem Provider Setup for Testing
# Run this script directly on any x86_64 Linux system or cloud server

set -e

echo "=== Simple Golem Provider Setup ==="

# Install Yagna using the official installer
echo "Installing Yagna..."
curl -sSf https://join.golem.network/as-provider | bash

# Add to PATH for current session
export PATH="$HOME/.local/bin:$PATH"

# Start Yagna service
echo "Starting Yagna service..."
yagna service run --api-allow-origin='*' &
YAGNA_PID=$!

# Wait for service to start
sleep 10

# Create app key
echo "Creating app key..."
yagna app-key create requestor

# Get node information
echo "=== Node Information ==="
yagna id show

# Get the node ID for testnet tokens
NODE_ID=$(yagna id show --json | jq -r '.Ok.nodeId')
echo ""
echo "=== Next Steps ==="
echo "1. Get testnet GLM tokens:"
echo "   Visit: https://faucet.testnet.golem.network/"
echo "   Node ID: $NODE_ID"
echo ""
echo "2. Configure provider:"
echo "   yagna provider preset create --preset-name beacon-provider --exe-unit wasmtime --pricing linear --price-duration 0.1 --price-cpu 0.1 --price-initial 0.0"
echo "   yagna provider preset activate beacon-provider"
echo ""
echo "3. Start provider:"
echo "   yagna provider run"
echo ""
echo "Yagna service is running in background (PID: $YAGNA_PID)"
echo "Press Ctrl+C to stop"

# Keep script running
wait $YAGNA_PID
