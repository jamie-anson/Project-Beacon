#!/bin/bash

# Modal deployment script for Project Beacon
# Run this script to deploy the Modal inference service

set -e

echo "🚀 Deploying Project Beacon Modal inference service..."

# Check if Modal is installed
if ! command -v modal &> /dev/null; then
    echo "❌ Modal CLI not found. Installing..."
    pip install modal
fi

# Check if authenticated
if ! modal token current &> /dev/null; then
    echo "🔐 Setting up Modal authentication..."
    modal setup
fi

# Deploy the app
echo "📦 Deploying Modal app..."
modal deploy modal_inference.py

echo "✅ Deployment completed!"
echo ""
echo "🔗 Your app is available at:"
echo "   Inference API: https://your-app-id--inference-api.modal.run"
echo ""
echo "🧪 To test the deployment:"
echo "   python test_modal.py"
echo ""
echo "📊 To monitor your app:"
echo "   modal app list"
echo "   modal app logs project-beacon-inference"
