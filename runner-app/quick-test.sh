#!/bin/bash
# Quick test script for tracing

echo "🔍 Quick Tracing Test"
echo "===================="
echo ""

# Set environment variables
export ENABLE_DB_TRACING=true
export DATABASE_URL="postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

echo "✅ Environment configured"
echo "   ENABLE_DB_TRACING: $ENABLE_DB_TRACING"
echo ""

# Check current span count
echo "📊 Current spans in database:"
SPAN_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM trace_spans;" 2>/dev/null | tr -d ' ')
if [ $? -eq 0 ]; then
    echo "   Total spans: $SPAN_COUNT"
    
    # Show recent spans
    echo ""
    echo "📋 Recent spans (last 5):"
    psql "$DATABASE_URL" -c "SELECT id, service, operation, status, created_at FROM trace_spans ORDER BY created_at DESC LIMIT 5;" 2>/dev/null
else
    echo "   ⚠️  Could not query database"
fi

echo ""
echo "===================="
echo ""
echo "To test tracing:"
echo "1. Start runner in another terminal:"
echo "   export ENABLE_DB_TRACING=true"
echo "   export DATABASE_URL=\"postgresql://...\""
echo "   go run cmd/runner/main.go"
echo ""
echo "2. Submit a test job via portal or:"
echo "   curl -X POST http://localhost:8090/api/v1/jobs/cross-region \\"
echo "     -H \"Content-Type: application/json\" \\"
echo "     -d '{\"models\": [\"llama-3.2-1b\"], \"regions\": [\"us-east\"], \"questions\": [\"test\"]}'"
echo ""
echo "3. Re-run this script to see new spans"
echo ""
