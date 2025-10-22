#!/bin/bash
# diagnose-router-health.sh - Comprehensive router health diagnostics

set -e

ROUTER_URL=${ROUTER_URL:-https://project-beacon-production.up.railway.app}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

pass() { echo -e "${GREEN}✅${NC} $1"; }
fail() { echo -e "${RED}❌${NC} $1"; }
warn() { echo -e "${YELLOW}⚠️${NC}  $1"; }
info() { echo -e "${BLUE}ℹ️${NC}  $1"; }
section() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}📋 $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔬 Router Health Comprehensive Diagnostics"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Test 1: Basic Connectivity
section "Test 1: Basic Connectivity"

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$ROUTER_URL/health" || echo "000")
if [[ "$HTTP_STATUS" == "200" ]]; then
    pass "Router is reachable"
else
    fail "Router returned HTTP $HTTP_STATUS or is unreachable"
    exit 1
fi

# Test 2: Startup Status
section "Test 2: Startup Health Check Status"

STARTUP=$(curl -s "$ROUTER_URL/debug/startup-status" 2>/dev/null || echo "{}")
if echo "$STARTUP" | jq -e '.startup_health_checks_completed' >/dev/null 2>&1; then
    COMPLETED=$(echo "$STARTUP" | jq -r '.startup_health_checks_completed')
    CHECKED=$(echo "$STARTUP" | jq -r '.providers_checked')
    TOTAL=$(echo "$STARTUP" | jq -r '.providers_total')
    HEALTHY=$(echo "$STARTUP" | jq -r '.providers_healthy')
    
    if [[ "$COMPLETED" == "true" ]]; then
        pass "Startup health checks completed ($CHECKED/$TOTAL providers checked)"
    else
        warn "Startup health checks incomplete ($CHECKED/$TOTAL providers checked)"
    fi
    
    info "Healthy providers: $HEALTHY/$TOTAL"
    
    NEVER_CHECKED=$(echo "$STARTUP" | jq -r '.providers_never_checked[]' 2>/dev/null || echo "")
    if [[ -n "$NEVER_CHECKED" ]]; then
        warn "Providers never checked: $NEVER_CHECKED"
    fi
else
    fail "Could not get startup status"
fi

# Test 3: Provider Details
section "Test 3: Provider Detailed Status"

PROVIDERS=$(curl -s "$ROUTER_URL/debug/providers" 2>/dev/null || echo "{}")
if echo "$PROVIDERS" | jq -e '.providers' >/dev/null 2>&1; then
    echo "$PROVIDERS" | jq -r '.providers[] | "  \(.name):"
    + "\n    Region: \(.region)"
    + "\n    Healthy: \(.healthy)"
    + "\n    Endpoint: \(.endpoint)"
    + "\n    Last check: \(if .last_health_check_ago_seconds then (.last_health_check_ago_seconds | tostring) + "s ago" else "Never" end)"
    + "\n"'
else
    fail "Could not get provider details"
fi

# Test 4: Health Check History
section "Test 4: Health Check History"

HISTORY=$(curl -s "$ROUTER_URL/debug/health-check-history" 2>/dev/null || echo "{}")
if echo "$HISTORY" | jq -e '.providers' >/dev/null 2>&1; then
    echo "$HISTORY" | jq -r '.providers[] | "  \(.provider): \(.healthy) (last: \(.last_check_ago_human))"'
else
    fail "Could not get health check history"
fi

# Test 5: Force Health Check
section "Test 5: Force Health Check"

info "Triggering manual health check..."
FORCE_RESULT=$(curl -s -X POST "$ROUTER_URL/debug/force-health-check" 2>/dev/null || echo "{}")

if echo "$FORCE_RESULT" | jq -e '.success' >/dev/null 2>&1; then
    SUCCESS=$(echo "$FORCE_RESULT" | jq -r '.success')
    DURATION=$(echo "$FORCE_RESULT" | jq -r '.duration_seconds')
    
    if [[ "$SUCCESS" == "true" ]]; then
        pass "Health check completed in ${DURATION}s"
        
        # Show before/after
        echo ""
        info "Provider states:"
        echo "$FORCE_RESULT" | jq -r '.after | to_entries[] | "  \(.key): \(.value)"'
        
        # Show changes
        CHANGES=$(echo "$FORCE_RESULT" | jq -r '.changes | length')
        if [[ "$CHANGES" -gt 0 ]]; then
            warn "$CHANGES provider(s) changed state:"
            echo "$FORCE_RESULT" | jq -r '.changes | to_entries[] | "  \(.key): \(.value.from) → \(.value.to)"'
        else
            info "No provider state changes"
        fi
    else
        fail "Health check failed"
    fi
else
    fail "Could not trigger health check"
fi

# Test 6: Test Individual Providers
section "Test 6: Test Individual Providers"

PROVIDER_NAMES=$(echo "$PROVIDERS" | jq -r '.providers[].name' 2>/dev/null || echo "")
if [[ -n "$PROVIDER_NAMES" ]]; then
    for PROVIDER in $PROVIDER_NAMES; do
        info "Testing $PROVIDER..."
        TEST_RESULT=$(curl -s -X POST "$ROUTER_URL/debug/test-provider/$PROVIDER" 2>/dev/null || echo "{}")
        
        if echo "$TEST_RESULT" | jq -e '.success' >/dev/null 2>&1; then
            SUCCESS=$(echo "$TEST_RESULT" | jq -r '.success')
            DURATION=$(echo "$TEST_RESULT" | jq -r '.duration_seconds')
            AFTER_HEALTHY=$(echo "$TEST_RESULT" | jq -r '.after.healthy')
            
            if [[ "$SUCCESS" == "true" ]] && [[ "$AFTER_HEALTHY" == "true" ]]; then
                pass "$PROVIDER is healthy (${DURATION}s)"
            elif [[ "$SUCCESS" == "true" ]] && [[ "$AFTER_HEALTHY" == "false" ]]; then
                fail "$PROVIDER is unhealthy (${DURATION}s)"
            else
                ERROR=$(echo "$TEST_RESULT" | jq -r '.error // "unknown error"')
                fail "$PROVIDER test failed: $ERROR"
            fi
        fi
    done
else
    warn "No providers found to test"
fi

# Test 7: Test Inference
section "Test 7: Test Inference Request"

info "Testing inference with diagnostics..."
INFERENCE_TEST=$(curl -s -X POST "$ROUTER_URL/debug/test-inference" 2>/dev/null || echo "{}")

if echo "$INFERENCE_TEST" | jq -e '.success' >/dev/null 2>&1; then
    SUCCESS=$(echo "$INFERENCE_TEST" | jq -r '.success')
    
    if [[ "$SUCCESS" == "true" ]]; then
        PROVIDER=$(echo "$INFERENCE_TEST" | jq -r '.provider_used')
        DURATION=$(echo "$INFERENCE_TEST" | jq -r '.duration_seconds')
        pass "Inference succeeded via $PROVIDER (${DURATION}s)"
    else
        ERROR=$(echo "$INFERENCE_TEST" | jq -r '.error // "unknown error"')
        fail "Inference failed: $ERROR"
        
        # Show provider states
        echo ""
        info "Provider states at time of failure:"
        echo "$INFERENCE_TEST" | jq -r '.provider_states | to_entries[] | "  \(.key): healthy=\(.value.healthy), region=\(.value.region)"'
        
        HEALTHY_COUNT=$(echo "$INFERENCE_TEST" | jq -r '.healthy_count // 0')
        warn "Healthy providers at time of request: $HEALTHY_COUNT"
    fi
else
    fail "Could not test inference"
fi

# Test 8: Direct Modal Endpoint Test
section "Test 8: Direct Modal Endpoint Test"

info "Testing Modal US endpoint directly..."
MODAL_TEST=$(curl -s -X POST "https://jamie-anson--project-beacon-hf-us-inference.modal.run" \
    -H "Content-Type: application/json" \
    -d '{"model":"llama3.2-1b","prompt":"test","temperature":0.1,"max_tokens":5}' 2>/dev/null || echo "{}")

if echo "$MODAL_TEST" | jq -e '.success' >/dev/null 2>&1; then
    SUCCESS=$(echo "$MODAL_TEST" | jq -r '.success')
    if [[ "$SUCCESS" == "true" ]]; then
        pass "Modal US endpoint is working"
    else
        fail "Modal US endpoint returned success=false"
    fi
else
    fail "Modal US endpoint is not responding correctly"
fi

# Summary
section "Summary"

echo ""
info "Diagnostic complete. Key findings:"
echo ""

# Determine overall status
if [[ "$HTTP_STATUS" == "200" ]] && [[ "$SUCCESS" == "true" ]]; then
    pass "Router is healthy and inference is working"
    echo ""
    echo "✅ No issues detected. System is operational."
elif [[ "$HTTP_STATUS" == "200" ]] && [[ "$SUCCESS" != "true" ]]; then
    warn "Router is reachable but inference is failing"
    echo ""
    echo "🔍 Recommendations:"
    echo "   1. Check provider health status above"
    echo "   2. Review health check timing (last_check_ago)"
    echo "   3. Check Railway logs for health check errors"
    echo "   4. Verify Modal endpoints are accessible from Railway"
else
    fail "Router is not reachable"
    echo ""
    echo "🔍 Recommendations:"
    echo "   1. Check Railway deployment status"
    echo "   2. Verify DNS resolution"
    echo "   3. Check Railway service logs"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
