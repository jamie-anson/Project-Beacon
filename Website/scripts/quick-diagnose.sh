#!/bin/bash
# quick-diagnose.sh - Fast diagnostic without SSH (uses logs + API calls)

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚡ Quick Execution Failure Diagnostic (No SSH)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Configuration
APP_NAME=${APP_NAME:-beacon-runner-production}
RUNNER_URL=${RUNNER_URL:-https://beacon-runner-production.fly.dev}
HYBRID_URL=${HYBRID_URL:-https://project-beacon-production.up.railway.app}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

pass() { echo -e "${GREEN}✅ PASS${NC}: $1"; }
fail() { echo -e "${RED}❌ FAIL${NC}: $1"; }
warn() { echo -e "${YELLOW}⚠️  WARN${NC}: $1"; }
info() { echo -e "${BLUE}ℹ️  INFO${NC}: $1"; }
section() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 $1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Check 1: Fly Secrets (Fast)
section "Check 1: Hybrid Configuration (Fly Secrets)"

echo "Checking Fly.io secrets..."
HYBRID_SECRETS=$(flyctl secrets list -a $APP_NAME 2>/dev/null | grep -E "HYBRID" || echo "NONE")

if [[ "$HYBRID_SECRETS" == "NONE" ]]; then
    fail "No HYBRID_* secrets found"
    echo ""
    echo "🔧 FIX: Set hybrid router URL:"
    echo "   flyctl secrets set HYBRID_BASE=$HYBRID_URL -a $APP_NAME"
    FOUND_CONFIG=false
else
    pass "Found HYBRID secrets"
    echo "$HYBRID_SECRETS" | sed 's/^/   /'
    FOUND_CONFIG=true
fi

# Check 2: Startup Logs (Fast)
section "Check 2: Hybrid Initialization Logs"

echo "Checking recent logs for hybrid initialization..."
HYBRID_LOGS=$(flyctl logs -a $APP_NAME --since 30m 2>/dev/null | grep -i "hybrid" | tail -10 || echo "NONE")

if [[ "$HYBRID_LOGS" == "NONE" ]]; then
    warn "No hybrid logs in last 30 minutes"
elif echo "$HYBRID_LOGS" | grep -q "Hybrid Router enabled"; then
    pass "Hybrid Router was enabled at startup"
    echo "$HYBRID_LOGS" | grep "Hybrid Router enabled" | tail -1 | sed 's/^/   /'
elif echo "$HYBRID_LOGS" | grep -q "Hybrid Router disabled"; then
    fail "Hybrid Router is DISABLED"
    echo "$HYBRID_LOGS" | grep "Hybrid Router disabled" | tail -1 | sed 's/^/   /'
    FOUND_CONFIG=false
else
    info "Hybrid logs found:"
    echo "$HYBRID_LOGS" | head -3 | sed 's/^/   /'
fi

# Check 3: Timeout Configuration (Fast)
section "Check 3: HTTP Timeout Configuration"

TIMEOUT_LOG=$(flyctl logs -a $APP_NAME --since 30m 2>/dev/null | grep "\[HYBRID_CLIENT_INIT\]" | tail -1 || echo "NONE")

if [[ "$TIMEOUT_LOG" == "NONE" ]]; then
    warn "No HYBRID_CLIENT_INIT log found"
else
    pass "Found client initialization"
    echo "$TIMEOUT_LOG" | sed 's/^/   /'
    
    if echo "$TIMEOUT_LOG" | grep -q "timeout=300s"; then
        pass "Timeout is 300s (good for Modal)"
    elif echo "$TIMEOUT_LOG" | grep -qE "timeout=[0-9]+s"; then
        TIMEOUT=$(echo "$TIMEOUT_LOG" | grep -oE 'timeout=[0-9]+' | cut -d= -f2)
        if [ "$TIMEOUT" -lt 300 ]; then
            warn "Timeout is ${TIMEOUT}s (might be too short)"
            echo ""
            echo "🔧 FIX: Increase timeout:"
            echo "   flyctl secrets set HYBRID_ROUTER_TIMEOUT=300 -a $APP_NAME"
        fi
    fi
fi

# Check 4: Hybrid Router Health (Fast)
section "Check 4: Hybrid Router Health"

echo "Testing hybrid router..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 "$HYBRID_URL/health" 2>/dev/null || echo "000")

if [[ "$HTTP_STATUS" == "200" ]]; then
    pass "Hybrid router is healthy"
elif [[ "$HTTP_STATUS" == "000" ]]; then
    fail "Cannot reach hybrid router"
    echo ""
    echo "🔧 FIX: Check Railway deployment"
else
    fail "Hybrid router returned HTTP $HTTP_STATUS"
fi

# Check 5: Providers (Fast)
section "Check 5: Provider Status"

PROVIDERS=$(curl -s --connect-timeout 5 --max-time 10 "$HYBRID_URL/providers" 2>/dev/null || echo "ERROR")

if [[ "$PROVIDERS" == "ERROR" ]]; then
    warn "Cannot reach /providers endpoint"
elif echo "$PROVIDERS" | jq -e '.providers | length > 0' >/dev/null 2>&1; then
    PROVIDER_COUNT=$(echo "$PROVIDERS" | jq '.providers | length')
    HEALTHY_COUNT=$(echo "$PROVIDERS" | jq '[.providers[] | select(.healthy == true)] | length')
    
    pass "Found $PROVIDER_COUNT providers ($HEALTHY_COUNT healthy)"
    
    if [ "$HEALTHY_COUNT" -eq 0 ]; then
        warn "No healthy providers!"
    fi
else
    warn "Could not parse providers"
fi

# Check 6: Recent Execution Logs (Fast)
section "Check 6: Recent Execution Attempts"

echo "Checking for execution logs..."
EXEC_LOGS=$(flyctl logs -a $APP_NAME --since 30m 2>/dev/null | grep -E "(executeQuestion|executor\.Execute|TRACE)" | tail -5 || echo "NONE")

if [[ "$EXEC_LOGS" == "NONE" ]]; then
    warn "No execution logs in last 30 minutes"
    echo "   Either no jobs submitted or logging not enabled"
else
    info "Recent execution logs:"
    echo "$EXEC_LOGS" | sed 's/^/   /'
    
    # Check for critical errors
    if echo "$EXEC_LOGS" | grep -q "executor is NIL"; then
        fail "CRITICAL: Executor is NIL!"
        echo ""
        echo "🔧 ROOT CAUSE: Hybrid client not initialized"
        FOUND_CONFIG=false
    elif echo "$EXEC_LOGS" | grep -q "hybrid client is NIL"; then
        fail "CRITICAL: Hybrid client is NIL!"
        FOUND_CONFIG=false
    fi
fi

# Check 7: Circuit Breaker (Fast)
section "Check 7: Circuit Breaker"

METRICS=$(curl -s --connect-timeout 5 --max-time 10 "$RUNNER_URL/api/v1/metrics" 2>/dev/null || echo "ERROR")

if [[ "$METRICS" != "ERROR" ]] && echo "$METRICS" | grep -q "hybrid_circuit_breaker_state"; then
    CIRCUIT_OPEN=$(echo "$METRICS" | grep 'hybrid_circuit_breaker_state{.*state="open"}' | grep -oE '[0-9]+$' || echo "0")
    
    if [[ "$CIRCUIT_OPEN" == "1" ]]; then
        fail "Circuit breaker is OPEN (blocking requests)"
        echo ""
        echo "🔧 FIX: Restart app:"
        echo "   flyctl apps restart $APP_NAME"
    else
        pass "Circuit breaker is closed"
    fi
else
    info "No circuit breaker metrics found"
fi

# Final Summary
section "Summary & Recommendations"

echo ""
if [[ "$FOUND_CONFIG" == "false" ]]; then
    echo "🔴 CRITICAL ISSUE FOUND:"
    echo ""
    echo "   Hybrid router is NOT configured or disabled"
    echo ""
    echo "🔧 IMMEDIATE FIX:"
    echo "   flyctl secrets set HYBRID_BASE=$HYBRID_URL -a $APP_NAME"
    echo ""
    echo "   Then restart:"
    echo "   flyctl apps restart $APP_NAME"
    echo ""
elif [[ "$HTTP_STATUS" != "200" ]]; then
    echo "🔴 CRITICAL ISSUE FOUND:"
    echo ""
    echo "   Hybrid router is unreachable"
    echo ""
    echo "🔧 IMMEDIATE FIX:"
    echo "   Check Railway deployment at https://railway.app"
    echo ""
else
    echo "✅ Configuration looks correct"
    echo ""
    echo "📋 Next steps:"
    echo "   1. Add enhanced trace logging (see EXECUTION-FAILURE-DEBUG-PLAN-ENHANCED.md)"
    echo "   2. Submit test job"
    echo "   3. Check logs: flyctl logs -a $APP_NAME | grep '🔍 TRACE'"
    echo ""
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "⚡ Quick Diagnostic Complete"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
