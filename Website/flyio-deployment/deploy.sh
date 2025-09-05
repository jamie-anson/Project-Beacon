#!/bin/bash

# Fly.io deployment script for Project Beacon hybrid router
# Run this script to deploy the hybrid routing service

set -e

echo "🚀 Deploying Project Beacon hybrid router to Fly.io..."

# Check if flyctl is installed
if ! command -v flyctl &> /dev/null; then
    echo "❌ flyctl not found. Please install Fly.io CLI:"
    echo "   curl -L https://fly.io/install.sh | sh"
    exit 1
fi

# Check if authenticated
if ! flyctl auth whoami &> /dev/null; then
    echo "🔐 Please authenticate with Fly.io:"
    flyctl auth login
fi

# Create app if it doesn't exist
if ! flyctl apps list | grep -q "beacon-hybrid-router"; then
    echo "📦 Creating new Fly.io app..."
    flyctl apps create beacon-hybrid-router
fi

# Set required secrets
echo "🔑 Setting up secrets..."
echo "Please set the following secrets manually:"
echo "  flyctl secrets set MODAL_API_TOKEN=your_modal_token -a beacon-hybrid-router"
echo "  flyctl secrets set RUNPOD_API_KEY=your_runpod_key -a beacon-hybrid-router"
echo "  flyctl secrets set GOLEM_PROVIDER_ENDPOINTS=endpoint1,endpoint2,endpoint3 -a beacon-hybrid-router"

# Deploy the app
echo "🚀 Deploying to Fly.io..."
flyctl deploy

echo "✅ Deployment completed!"
echo ""
echo "🔗 Your hybrid router is available at:"
echo "   https://beacon-hybrid-router.fly.dev"
echo ""
echo "🧪 Test endpoints:"
echo "   Health: https://beacon-hybrid-router.fly.dev/health"
echo "   Providers: https://beacon-hybrid-router.fly.dev/providers"
echo "   Metrics: https://beacon-hybrid-router.fly.dev/metrics"
echo ""
echo "📊 To monitor your app:"
echo "   flyctl logs -a beacon-hybrid-router"
echo "   flyctl status -a beacon-hybrid-router"
