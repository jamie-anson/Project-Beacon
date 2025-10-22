#!/bin/bash
# Trace Query Helper Script
# Usage: ./query-traces.sh [command] [args]

DB_URL="postgresql://neondb_owner:npg_puA76KTFISkD@ep-broad-cherry-abdo0pru-pooler.eu-west-2.aws.neon.tech/neondb?sslmode=require&channel_binding=require"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

show_help() {
    echo -e "${BLUE}🔍 Trace Query Helper${NC}"
    echo ""
    echo "Usage: ./query-traces.sh [command] [args]"
    echo ""
    echo "Commands:"
    echo "  ${GREEN}recent${NC}              - Show last 10 trace spans"
    echo "  ${GREEN}diagnose <exec_id>${NC}  - Diagnose execution (auto-detect issues)"
    echo "  ${GREEN}waterfall <exec_id>${NC} - Show trace waterfall timeline"
    echo "  ${GREEN}execution <exec_id>${NC} - Show all spans for execution"
    echo "  ${GREEN}job <job_id>${NC}        - Show executions for job"
    echo "  ${GREEN}failures${NC}            - Find root cause of failures"
    echo "  ${GREEN}similar <exec_id>${NC}   - Find similar traces"
    echo "  ${GREEN}health${NC}              - Check tracing system health"
    echo "  ${GREEN}stats${NC}               - Show tracing statistics"
    echo "  ${GREEN}errors${NC}              - Show recent errors"
    echo "  ${GREEN}slow${NC}                - Show slow spans (>1000ms)"
    echo ""
    echo "Examples:"
    echo "  ./query-traces.sh recent"
    echo "  ./query-traces.sh diagnose 2359"
    echo "  ./query-traces.sh job 469"
    echo ""
}

query_recent() {
    echo -e "${BLUE}📋 Recent Trace Spans (Last 10)${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            id,
            LEFT(trace_id::text, 8) as trace,
            service,
            operation,
            status,
            duration_ms,
            TO_CHAR(created_at, 'HH24:MI:SS') as time
        FROM trace_spans 
        ORDER BY created_at DESC 
        LIMIT 10;
    "
}

query_diagnose() {
    if [ -z "$1" ]; then
        echo -e "${RED}❌ Error: Execution ID required${NC}"
        echo "Usage: ./query-traces.sh diagnose <execution_id>"
        exit 1
    fi
    
    echo -e "${BLUE}🔍 Diagnosing Execution $1${NC}"
    echo ""
    psql "$DB_URL" -c "SELECT * FROM diagnose_execution_trace($1);" -x
}

query_waterfall() {
    if [ -z "$1" ]; then
        echo -e "${RED}❌ Error: Execution ID required${NC}"
        echo "Usage: ./query-traces.sh waterfall <execution_id>"
        exit 1
    fi
    
    echo -e "${BLUE}📊 Trace Waterfall for Execution $1${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            level,
            service,
            operation,
            status,
            duration_ms,
            TO_CHAR(started_at, 'HH24:MI:SS.MS') as started,
            TO_CHAR(completed_at, 'HH24:MI:SS.MS') as completed
        FROM trace_waterfall 
        WHERE execution_id = $1 
        ORDER BY level, started_at;
    "
}

query_execution() {
    if [ -z "$1" ]; then
        echo -e "${RED}❌ Error: Execution ID required${NC}"
        echo "Usage: ./query-traces.sh execution <execution_id>"
        exit 1
    fi
    
    echo -e "${BLUE}📍 All Spans for Execution $1${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            id,
            service,
            operation,
            status,
            duration_ms,
            error_message,
            TO_CHAR(started_at, 'HH24:MI:SS.MS') as started
        FROM trace_spans 
        WHERE execution_id = $1 
        ORDER BY started_at;
    " -x
}

query_job() {
    if [ -z "$1" ]; then
        echo -e "${RED}❌ Error: Job ID required${NC}"
        echo "Usage: ./query-traces.sh job <job_id>"
        exit 1
    fi
    
    echo -e "${BLUE}📦 Executions for Job $1${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            id,
            region,
            model_id,
            status,
            EXTRACT(EPOCH FROM (completed_at - started_at)) as duration_sec,
            TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS') as created
        FROM executions 
        WHERE job_id = '$1' 
        ORDER BY created_at;
    "
}

query_failures() {
    echo -e "${BLUE}🔴 Root Cause Analysis for Failures${NC}"
    echo ""
    psql "$DB_URL" -c "SELECT * FROM identify_root_cause('failed');"
}

query_similar() {
    if [ -z "$1" ]; then
        echo -e "${RED}❌ Error: Execution ID required${NC}"
        echo "Usage: ./query-traces.sh similar <execution_id>"
        exit 1
    fi
    
    echo -e "${BLUE}🔗 Similar Traces to Execution $1${NC}"
    echo ""
    psql "$DB_URL" -c "SELECT * FROM find_similar_traces($1, 10);"
}

query_health() {
    echo -e "${BLUE}💚 Tracing System Health${NC}"
    echo ""
    psql "$DB_URL" -c "SELECT * FROM trace_spans_health;"
}

query_stats() {
    echo -e "${BLUE}📊 Tracing Statistics${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            COUNT(*) as total_spans,
            COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 hour') as last_hour,
            COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '1 day') as last_day,
            COUNT(*) FILTER (WHERE status = 'completed') as completed,
            COUNT(*) FILTER (WHERE status = 'failed') as failed,
            COUNT(*) FILTER (WHERE status = 'error') as errors,
            ROUND(AVG(duration_ms)::numeric, 2) as avg_duration_ms,
            MAX(duration_ms) as max_duration_ms
        FROM trace_spans;
    "
    
    echo ""
    echo -e "${BLUE}📈 Spans by Service${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            service,
            COUNT(*) as count,
            ROUND(AVG(duration_ms)::numeric, 2) as avg_ms,
            MAX(duration_ms) as max_ms
        FROM trace_spans 
        GROUP BY service 
        ORDER BY count DESC;
    "
}

query_errors() {
    echo -e "${BLUE}❌ Recent Errors (Last 20)${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            id,
            service,
            operation,
            error_message,
            TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS') as occurred
        FROM trace_spans 
        WHERE error_message IS NOT NULL 
        ORDER BY created_at DESC 
        LIMIT 20;
    " -x
}

query_slow() {
    echo -e "${BLUE}🐌 Slow Spans (>1000ms)${NC}"
    echo ""
    psql "$DB_URL" -c "
        SELECT 
            id,
            service,
            operation,
            duration_ms,
            status,
            TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI:SS') as occurred
        FROM trace_spans 
        WHERE duration_ms > 1000 
        ORDER BY duration_ms DESC 
        LIMIT 20;
    "
}

# Main command router
case "$1" in
    recent)
        query_recent
        ;;
    diagnose)
        query_diagnose "$2"
        ;;
    waterfall)
        query_waterfall "$2"
        ;;
    execution)
        query_execution "$2"
        ;;
    job)
        query_job "$2"
        ;;
    failures)
        query_failures
        ;;
    similar)
        query_similar "$2"
        ;;
    health)
        query_health
        ;;
    stats)
        query_stats
        ;;
    errors)
        query_errors
        ;;
    slow)
        query_slow
        ;;
    help|--help|-h|"")
        show_help
        ;;
    *)
        echo -e "${RED}❌ Unknown command: $1${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
