#!/bin/bash
# Watch for new trace spans

DB_URL="postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

echo "🔍 Watching for trace spans..."
echo "Press Ctrl+C to stop"
echo ""

while true; do
    clear
    echo "🔍 Trace Spans Monitor - $(date '+%H:%M:%S')"
    echo "=========================================="
    echo ""
    
    # Summary
    psql "$DB_URL" -c "SELECT COUNT(*) as total, COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 minute') as last_1min, COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '5 minutes') as last_5min FROM trace_spans;"
    
    echo ""
    echo "Recent Spans (last 10):"
    echo "----------------------------------------"
    psql "$DB_URL" -c "SELECT id, service, operation, status, EXTRACT(EPOCH FROM (NOW() - created_at))::INT as seconds_ago FROM trace_spans ORDER BY created_at DESC LIMIT 10;"
    
    echo ""
    echo "Recent Executions (last 5):"
    echo "----------------------------------------"
    psql "$DB_URL" -c "SELECT id, job_id, region, model_id, status, EXTRACT(EPOCH FROM (NOW() - created_at))::INT as seconds_ago FROM executions ORDER BY created_at DESC LIMIT 5;"
    
    echo ""
    echo "Refreshing in 5 seconds... (Ctrl+C to stop)"
    sleep 5
done
