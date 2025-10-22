#!/bin/bash
# diagnose-execution-failure.sh - Quick diagnostic for execution failures
# Run this script to quickly identify the root cause of 10-30ms failures

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔬 Project Beacon Execution Failure Diagnostic"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Configuration
APP_NAME=${APP_NAME:-beacon-runner-production}
RUNNER_URL=${RUNNER_URL:-https://beacon-runner-production.fly.dev}
HYBRID_URL=${HYBRID_URL:-https://project-beacon-production.up.railway.app}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
}

fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
}

warn() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
}

info() {
    echo -e "${BLUE}ℹ️  INFO${NC}: $1"
}

section() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 $1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Check 1: Hybrid Environment Variables
section "Check 1: Hybrid Router Configuration"

echo "Checking environment variables in runner app..."
echo "(This may take 10-15 seconds to connect...)"
HYBRID_ENV=$(timeout 30 flyctl ssh console -a $APP_NAME -C 'printenv | grep -E "^(HYBRID|ENABLE_HYBRID)" || echo "NONE"' 2>&1 || echo "ERROR")

if [[ "$HYBRID_ENV" == "ERROR" ]] || [[ "$HYBRID_ENV" == *"timed out"* ]] || [[ "$HYBRID_ENV" == *"failed"* ]]; then
    warn "Could not connect to runner app via SSH"
    echo "   Trying alternative method: checking secrets..."
    
    # Fallback: Check Fly secrets instead
    HYBRID_SECRETS=$(flyctl secrets list -a $APP_NAME 2>/dev/null | grep -E "HYBRID" || echo "NONE")
    
    if [[ "$HYBRID_SECRETS" == "NONE" ]]; then
        fail "No HYBRID_* secrets found in Fly.io"
        echo ""
        echo "🔧 FIX: Set one of these secrets:"
        echo "   flyctl secrets set HYBRID_BASE=$HYBRID_URL -a $APP_NAME"
        echo "   OR"
        echo "   flyctl secrets set HYBRID_ROUTER_URL=$HYBRID_URL -a $APP_NAME"
        exit 1
    else
        pass "Found HYBRID secrets in Fly.io configuration"
        echo "$HYBRID_SECRETS" | sed 's/^/   /'
        echo ""
        info "Note: Cannot verify runtime values without SSH access"
        echo "   Proceeding with remaining checks..."
    fi
elif [[ "$HYBRID_ENV" == "NONE" ]]; then
    fail "No HYBRID_* environment variables found"
    echo ""
    echo "🔧 FIX: Set one of these environment variables:"
    echo "   flyctl secrets set HYBRID_BASE=$HYBRID_URL -a $APP_NAME"
    echo "   OR"
    echo "   flyctl secrets set HYBRID_ROUTER_URL=$HYBRID_URL -a $APP_NAME"
    exit 1
else
    pass "Hybrid environment variables found"
    echo "$HYBRID_ENV" | sed 's/^/   /'
fi

# Check 2: Startup Logs for Hybrid Initialization
section "Check 2: Hybrid Client Initialization"

echo "Checking startup logs for hybrid client initialization..."
HYBRID_LOGS=$(flyctl logs -a $APP_NAME --since 10m | grep -i "hybrid" | tail -5 || echo "NONE")

if [[ "$HYBRID_LOGS" == "NONE" ]]; then
    warn "No hybrid-related logs found in last 10 minutes"
    echo "   This might mean:"
    echo "   - App hasn't restarted recently"
    echo "   - Hybrid initialization is failing silently"
elif echo "$HYBRID_LOGS" | grep -q "Hybrid Router enabled"; then
    pass "Hybrid Router was enabled at startup"
    echo "$HYBRID_LOGS" | grep "Hybrid Router enabled" | sed 's/^/   /'
elif echo "$HYBRID_LOGS" | grep -q "Hybrid Router disabled"; then
    fail "Hybrid Router is DISABLED"
    echo "$HYBRID_LOGS" | grep "Hybrid Router disabled" | sed 's/^/   /'
    echo ""
    echo "🔧 FIX: Set HYBRID_BASE or enable default hybrid:"
    echo "   flyctl secrets set HYBRID_BASE=$HYBRID_URL -a $APP_NAME"
    exit 1
else
    warn "Hybrid logs found but unclear status"
    echo "$HYBRID_LOGS" | sed 's/^/   /'
fi

# Check 3: Hybrid Client Timeout Configuration
section "Check 3: HTTP Client Timeout"

echo "Checking hybrid client timeout from logs..."
TIMEOUT_LOG=$(flyctl logs -a $APP_NAME --since 10m | grep "\[HYBRID_CLIENT_INIT\]" | tail -1 || echo "NONE")

if [[ "$TIMEOUT_LOG" == "NONE" ]]; then
    warn "No HYBRID_CLIENT_INIT log found"
    echo "   Cannot verify timeout setting"
else
    pass "Found client initialization log"
    echo "$TIMEOUT_LOG" | sed 's/^/   /'
    
    # Extract timeout value
    if echo "$TIMEOUT_LOG" | grep -q "timeout=300s"; then
        pass "Timeout is set to 300s (good for Modal cold starts)"
    elif echo "$TIMEOUT_LOG" | grep -q "timeout=[0-9]\+s"; then
        TIMEOUT=$(echo "$TIMEOUT_LOG" | grep -oP 'timeout=\K[0-9]+')
        if [ "$TIMEOUT" -lt 300 ]; then
            warn "Timeout is ${TIMEOUT}s (might be too short for Modal cold starts)"
            echo ""
            echo "🔧 FIX: Increase timeout:"
            echo "   flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 -a $APP_NAME"
        else
            pass "Timeout is ${TIMEOUT}s (sufficient)"
        fi
    fi
fi

# Check 4: Hybrid Router Health
section "Check 4: Hybrid Router Availability"

echo "Testing hybrid router connectivity..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "$HYBRID_URL/health" 2>/dev/null || echo "000")

if [[ "$HTTP_STATUS" == "200" ]]; then
    pass "Hybrid router is reachable and healthy"
elif [[ "$HTTP_STATUS" == "000" ]]; then
    fail "Cannot reach hybrid router (connection timeout or network error)"
    echo ""
    echo "🔧 FIX: Check hybrid router deployment:"
    echo "   1. Check Railway dashboard: https://railway.app"
    echo "   2. Test manually: curl $HYBRID_URL/health"
    echo "   3. Check DNS: nslookup project-beacon-production.up.railway.app"
    exit 1
else
    fail "Hybrid router returned HTTP $HTTP_STATUS"
    echo ""
    echo "🔧 FIX: Check hybrid router logs in Railway dashboard"
    exit 1
fi

# Check 5: Provider Endpoint
section "Check 5: Provider Discovery"

echo "Testing provider discovery endpoint..."
PROVIDERS=$(curl -s --connect-timeout 5 --max-time 10 "$HYBRID_URL/providers" 2>/dev/null || echo "ERROR")

if [[ "$PROVIDERS" == "ERROR" ]]; then
    fail "Cannot reach /providers endpoint"
elif echo "$PROVIDERS" | jq -e '.providers | length > 0' >/dev/null 2>&1; then
    PROVIDER_COUNT=$(echo "$PROVIDERS" | jq '.providers | length')
    pass "Found $PROVIDER_COUNT providers"
    
    # Show provider health
    HEALTHY_COUNT=$(echo "$PROVIDERS" | jq '[.providers[] | select(.healthy == true)] | length')
    if [ "$HEALTHY_COUNT" -gt 0 ]; then
        pass "$HEALTHY_COUNT providers are healthy"
    else
        warn "No healthy providers found"
    fi
    
    # List providers
    echo ""
    echo "Providers:"
    echo "$PROVIDERS" | jq -r '.providers[] | "   \(.name) - \(.type) - \(.region) - healthy: \(.healthy)"'
else
    warn "Could not parse provider response"
    echo "$PROVIDERS" | head -5 | sed 's/^/   /'
fi

# Check 6: Recent Execution Failures
section "Check 6: Recent Execution Logs"

echo "Checking for execution-related logs..."
EXEC_LOGS=$(flyctl logs -a $APP_NAME --since 10m | grep -E "(executeQuestion|executor\.Execute|TRACE)" | tail -10 || echo "NONE")

if [[ "$EXEC_LOGS" == "NONE" ]]; then
    warn "No execution trace logs found in last 10 minutes"
    echo "   This means either:"
    echo "   - No jobs have been submitted recently"
    echo "   - Trace logging is not enabled"
else
    info "Found execution logs (showing last 10):"
    echo "$EXEC_LOGS" | sed 's/^/   /'
    
    # Check for critical errors
    if echo "$EXEC_LOGS" | grep -q "executor is NIL"; then
        fail "CRITICAL: Executor is NIL - hybrid client not initialized!"
        echo ""
        echo "🔧 FIX: Hybrid executor was not set up properly"
        echo "   This confirms the root cause: Hybrid configuration issue"
        exit 1
    elif echo "$EXEC_LOGS" | grep -q "hybrid client is NIL"; then
        fail "CRITICAL: Hybrid client is NIL!"
        echo ""
        echo "🔧 FIX: Hybrid client creation failed"
        exit 1
    fi
fi

# Check 7: Circuit Breaker State
section "Check 7: Circuit Breaker Status"

echo "Checking circuit breaker state..."
METRICS=$(curl -s --connect-timeout 5 --max-time 10 "$RUNNER_URL/api/v1/metrics" 2>/dev/null || echo "ERROR")

if [[ "$METRICS" == "ERROR" ]]; then
    warn "Could not fetch metrics endpoint"
elif echo "$METRICS" | grep -q "hybrid_circuit_breaker_state"; then
    CIRCUIT_STATE=$(echo "$METRICS" | grep "hybrid_circuit_breaker_state" | grep 'state="open"' | grep -oP 'hybrid_circuit_breaker_state\{[^}]+\} \K\d+' || echo "0")
    
    if [[ "$CIRCUIT_STATE" == "1" ]]; then
        fail "Circuit breaker is OPEN (blocking all requests)"
        echo ""
        echo "🔧 FIX: Reset circuit breaker:"
        echo "   flyctl apps restart $APP_NAME"
        exit 1
    else
        pass "Circuit breaker is CLOSED (allowing requests)"
    fi
else
    info "No circuit breaker metrics found (might not be instrumented)"
fi

# Check 8: Recent Job Status
section "Check 8: Recent Job Executions"

echo "Checking database for recent job failures..."
if command -v psql &> /dev/null && [ -n "$DATABASE_URL" ]; then
    RECENT_JOBS=$(psql "$DATABASE_URL" -t -c "
        SELECT 
            e.id,
            e.job_id,
            e.region,
            e.status,
            EXTRACT(EPOCH FROM (e.completed_at - e.started_at)) * 1000 AS duration_ms
        FROM executions e
        ORDER BY e.created_at DESC
        LIMIT 5;
    " 2>/dev/null || echo "ERROR")
    
    if [[ "$RECENT_JOBS" == "ERROR" ]]; then
        warn "Could not query database (DATABASE_URL not set or connection failed)"
    else
        info "Recent executions:"
        echo "$RECENT_JOBS" | sed 's/^/   /'
        
        # Check for suspiciously fast executions
        if echo "$RECENT_JOBS" | awk '{print $5}' | grep -qE '^[0-9]{1,2}\.'; then
            warn "Found executions < 100ms - likely immediate failures"
        fi
    fi
else
    warn "Skipping database check (psql not available or DATABASE_URL not set)"
fi

# Final Summary
section "Diagnostic Summary"

echo ""
echo "🎯 Recommendations:"
echo ""

# Determine most likely issue
if echo "$HYBRID_ENV" | grep -q "HYBRID_ROUTER_DISABLE=true"; then
    echo "1. 🔴 CRITICAL: Hybrid router is explicitly DISABLED"
    echo "   Remove HYBRID_ROUTER_DISABLE or set to false"
    echo ""
elif ! echo "$HYBRID_ENV" | grep -qE "(HYBRID_BASE|HYBRID_ROUTER_URL)"; then
    echo "1. 🔴 CRITICAL: No hybrid router URL configured"
    echo "   Set HYBRID_BASE or HYBRID_ROUTER_URL:"
    echo "   $ flyctl secrets set HYBRID_BASE=$HYBRID_URL -a $APP_NAME"
    echo ""
elif [[ "$HTTP_STATUS" != "200" ]]; then
    echo "1. 🔴 CRITICAL: Hybrid router is not reachable"
    echo "   Check Railway deployment and network connectivity"
    echo ""
elif [[ "$CIRCUIT_STATE" == "1" ]]; then
    echo "1. 🔴 CRITICAL: Circuit breaker is blocking requests"
    echo "   Restart the app to reset circuit breaker"
    echo ""
else
    echo "1. ✅ Configuration looks correct"
    echo "   Next steps:"
    echo "   - Add enhanced trace logging (see EXECUTION-FAILURE-DEBUG-PLAN-ENHANCED.md Phase 2)"
    echo "   - Submit a test job and check logs with:"
    echo "     $ flyctl logs -a $APP_NAME | grep '🔍 TRACE'"
    echo ""
fi

echo "2. 📚 For detailed debugging, see:"
echo "   EXECUTION-FAILURE-DEBUG-PLAN-ENHANCED.md"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Diagnostic Complete"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
