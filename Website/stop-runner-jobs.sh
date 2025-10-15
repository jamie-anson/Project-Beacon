#!/bin/bash
# Stop all running jobs in the Fly.io runner
# This will restart the runner to clear any in-flight jobs

set -e

echo "🛑 Stopping Runner Jobs"
echo "======================="

# Check if flyctl is available
if ! command -v fly &> /dev/null; then
    echo "❌ fly CLI not found. Please install: https://fly.io/docs/hands-on/install-flyctl/"
    exit 1
fi

# Runner app name
RUNNER_APP="${RUNNER_APP:-beacon-runner-production}"

echo ""
echo "📊 Current Runner Status:"
echo "-------------------------"
fly status -a "$RUNNER_APP"

echo ""
echo "🔄 Restarting runner to clear in-flight jobs..."
echo "------------------------------------------------"

# Restart the runner
fly apps restart "$RUNNER_APP"

echo ""
echo "⏳ Waiting for runner to come back up..."
sleep 5

echo ""
echo "📊 New Runner Status:"
echo "---------------------"
fly status -a "$RUNNER_APP"

echo ""
echo "✅ Runner restarted successfully!"
echo ""
echo "All in-flight jobs have been cleared."
echo "Jobs in 'processing' status will be recovered by job_recovery.go"
