#!/bin/bash
# Test script for distributed tracing

set -e

echo "🔍 Testing Distributed Tracing Integration"
echo "=========================================="
echo ""

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "❌ ERROR: DATABASE_URL not set"
    echo "Run: export DATABASE_URL='postgresql://...'"
    exit 1
fi

# Check if ENABLE_DB_TRACING is set
if [ "$ENABLE_DB_TRACING" != "true" ]; then
    echo "⚠️  WARNING: ENABLE_DB_TRACING is not set to 'true'"
    echo "Tracing will be disabled. To enable:"
    echo "export ENABLE_DB_TRACING=true"
    echo ""
fi

echo "✅ Environment configured"
echo "   DATABASE_URL: ${DATABASE_URL:0:50}..."
echo "   ENABLE_DB_TRACING: $ENABLE_DB_TRACING"
echo ""

# Check database connection
echo "🔌 Testing database connection..."
psql "$DATABASE_URL" -c "SELECT 1;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✅ Database connection successful"
else
    echo "❌ Database connection failed"
    exit 1
fi
echo ""

# Check if trace_spans table exists
echo "📊 Checking trace_spans table..."
TABLE_EXISTS=$(psql "$DATABASE_URL" -t -c "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'trace_spans');" | tr -d ' ')
if [ "$TABLE_EXISTS" = "t" ]; then
    echo "✅ trace_spans table exists"
    
    # Show current span count
    SPAN_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM trace_spans;" | tr -d ' ')
    echo "   Current spans in database: $SPAN_COUNT"
else
    echo "❌ trace_spans table not found"
    echo "   Run migration: psql \$DATABASE_URL < migrations/0011_add_trace_spans.up.sql"
    exit 1
fi
echo ""

# Check if diagnostic functions exist
echo "🔧 Checking diagnostic functions..."
FUNC_EXISTS=$(psql "$DATABASE_URL" -t -c "SELECT EXISTS (SELECT FROM pg_proc WHERE proname = 'diagnose_execution_trace');" | tr -d ' ')
if [ "$FUNC_EXISTS" = "t" ]; then
    echo "✅ Diagnostic functions exist"
else
    echo "❌ Diagnostic functions not found"
    exit 1
fi
echo ""

# Build the runner
echo "🔨 Building runner..."
go build -o /tmp/test-runner cmd/runner/main.go
if [ $? -eq 0 ]; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi
echo ""

echo "=========================================="
echo "✅ All pre-flight checks passed!"
echo ""
echo "Next steps:"
echo "1. Start runner: go run cmd/runner/main.go"
echo "2. Submit a test job"
echo "3. Query traces:"
echo "   psql \$DATABASE_URL -c \"SELECT * FROM trace_spans ORDER BY created_at DESC LIMIT 5;\""
echo "   psql \$DATABASE_URL -c \"SELECT * FROM diagnose_execution_trace(<execution_id>);\""
echo ""
