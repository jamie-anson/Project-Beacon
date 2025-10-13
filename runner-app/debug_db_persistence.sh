#!/bin/bash
# Database Persistence Diagnostic Script

echo "=== Database Persistence Diagnostic ==="
echo ""

# Get the test job ID from the bug report
JOB_ID="bias-detection-1760288315095"
RUNNER_URL="https://beacon-runner-production.fly.dev"

echo "1. Checking job status via API..."
curl -s "$RUNNER_URL/api/v1/jobs/$JOB_ID" | jq '{id, status, created_at, updated_at}'
echo ""

echo "2. Checking executions via API..."
curl -s "$RUNNER_URL/api/v1/jobs/$JOB_ID/executions/all" | jq '{count: (.executions | length), executions: .executions}'
echo ""

echo "3. Checking recent logs for this job..."
flyctl logs -a beacon-runner-production | grep "$JOB_ID" | tail -20
echo ""

echo "4. Checking for database errors..."
flyctl logs -a beacon-runner-production | grep -i "failed to update\|database\|rows affected" | tail -10
echo ""

echo "=== Next Steps ==="
echo "If job status is 'queued':"
echo "  - Check if goroutine completed: grep 'Cross-region goroutine completed'"
echo "  - Check if UpdateJobStatus was called: grep 'failed to update job status'"
echo "  - Enable DB_AUDIT=1 to see actual SQL execution"
echo ""
echo "To enable DB_AUDIT:"
echo "  flyctl secrets set DB_AUDIT=1 -a beacon-runner-production"
echo "  flyctl deploy -a beacon-runner-production"
