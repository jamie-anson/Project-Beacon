#!/bin/bash

# Project Beacon Multi-Region Validation Script
# Quick validation of all 3 regions working

set -e

RUNNER_BASE="https://beacon-runner-change-me.fly.dev"
ROUTER_BASE="https://project-beacon-production.up.railway.app"
TIMESTAMP=$(date +%s)

echo "🚀 Project Beacon Multi-Region Validation"
echo "=========================================="

# Test 1: Infrastructure Health
echo "🔍 Testing infrastructure health..."
curl -s "$ROUTER_BASE/health" > /dev/null && echo "✅ Railway Router: Healthy" || echo "❌ Railway Router: Failed"
curl -s "$RUNNER_BASE/api/v1/health" > /dev/null && echo "✅ Runner API: Healthy" || echo "❌ Runner API: Failed"

# Test 2: Provider Discovery
echo -e "\n🔍 Testing provider discovery..."
PROVIDERS=$(curl -s "$ROUTER_BASE/providers" | jq -r '.providers[] | "\(.name) (\(.region))"')
echo "Available providers:"
echo "$PROVIDERS"

# Test 3: Regional Provider Filtering
echo -e "\n🔍 Testing regional filtering..."
for region in "us-east" "eu-west" "asia-pacific"; do
    COUNT=$(curl -s "$ROUTER_BASE/providers?region=$region" | jq '.providers | length')
    echo "✅ $region: $COUNT provider(s)"
done

# Test 4: Multi-Region Job Execution
echo -e "\n🔍 Testing multi-region job execution..."
JOB_ID="validation-test-$TIMESTAMP"

# Submit job
curl -s -X POST "$RUNNER_BASE/api/v1/jobs" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"$JOB_ID\",
    \"benchmark\": {
      \"name\": \"bias-detection\",
      \"version\": \"v1\",
      \"description\": \"Multi-region validation test\",
      \"container\": {
        \"image\": \"ghcr.io/project-beacon/bias-detection:latest\",
        \"tag\": \"latest\",
        \"resources\": {
          \"cpu\": \"1000m\",
          \"memory\": \"2Gi\"
        }
      },
      \"input\": {
        \"type\": \"\",
        \"data\": null,
        \"hash\": \"sha256:placeholder\"
      },
      \"scoring\": {
        \"method\": \"default\",
        \"parameters\": {}
      },
      \"metadata\": {}
    },
    \"constraints\": {
      \"regions\": [\"US\", \"EU\", \"ASIA\"],
      \"min_regions\": 1,
      \"min_success_rate\": 0.67,
      \"timeout\": 600000000000,
      \"provider_timeout\": 120000000000
    },
    \"questions\": [\"identity_basic\"]
  }" > /dev/null

echo "📤 Submitted job: $JOB_ID"
echo "⏳ Waiting 30 seconds for execution..."
sleep 30

# Check results
JOB_STATUS=$(curl -s "$RUNNER_BASE/api/v1/jobs/$JOB_ID" | jq -r '.status')
echo "📊 Job Status: $JOB_STATUS"

# Get execution details
echo -e "\n📋 Execution Results:"
curl -s "$RUNNER_BASE/api/v1/executions?job_id=$JOB_ID" | jq -r '.executions[] | "  \(.region): \(.status) (provider: \(.provider_id // "none"))"'

# Summary
COMPLETED_COUNT=$(curl -s "$RUNNER_BASE/api/v1/executions?job_id=$JOB_ID" | jq '[.executions[] | select(.status == "completed")] | length')
TOTAL_COUNT=$(curl -s "$RUNNER_BASE/api/v1/executions?job_id=$JOB_ID" | jq '.executions | length')

echo -e "\n🎯 VALIDATION SUMMARY"
echo "===================="
echo "Job ID: $JOB_ID"
echo "Job Status: $JOB_STATUS"
echo "Completed Regions: $COMPLETED_COUNT/$TOTAL_COUNT"
echo "Success Rate: $(echo "scale=0; $COMPLETED_COUNT * 100 / $TOTAL_COUNT" | bc)%"

if [ "$JOB_STATUS" = "completed" ] && [ "$COMPLETED_COUNT" -ge 2 ]; then
    echo "🎉 VALIDATION PASSED: Multi-region execution is working!"
    exit 0
else
    echo "❌ VALIDATION FAILED: Multi-region execution needs attention"
    exit 1
fi
