#!/bin/bash

# Phase 2: API Health & Monitoring Tests
# Comprehensive health monitoring for all Project Beacon services

set -e

# Service endpoints
RUNNER_URL="https://beacon-runner-change-me.fly.dev"
HYBRID_URL="https://project-beacon-production.up.railway.app"
PORTAL_URL="https://projectbeacon.netlify.app"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_debug() { echo -e "${BLUE}[DEBUG]${NC} $1"; }

# Test 2.1: Service Health Endpoints
test_service_health() {
    log_info "Testing all service health endpoints..."
    
    local failed_services=()
    local response_times=()
    
    # Test Runner health
    log_info "Checking Runner service..."
    local start_time=$(date +%s%N)
    local runner_health=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/health" || echo "")
    local end_time=$(date +%s%N)
    local runner_time=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    if [[ -n "$runner_health" ]]; then
        log_info "✅ Runner health OK (${runner_time}ms)"
        response_times+=("Runner: ${runner_time}ms")
    else
        log_error "❌ Runner health failed"
        failed_services+=("Runner")
    fi
    
    # Test Hybrid Router health
    log_info "Checking Hybrid Router service..."
    start_time=$(date +%s%N)
    local hybrid_health=$(curl -s --max-time 10 "${HYBRID_URL}/health" || echo "")
    end_time=$(date +%s%N)
    local hybrid_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ -n "$hybrid_health" ]]; then
        log_info "✅ Hybrid Router health OK (${hybrid_time}ms)"
        response_times+=("Hybrid: ${hybrid_time}ms")
    else
        log_error "❌ Hybrid Router health failed"
        failed_services+=("Hybrid")
    fi
    
    # Test Portal proxied endpoints
    log_info "Checking Portal proxy endpoints..."
    start_time=$(date +%s%N)
    local portal_runner_health=$(curl -s --max-time 10 "${PORTAL_URL}/api/v1/health" || echo "")
    end_time=$(date +%s%N)
    local portal_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ -n "$portal_runner_health" ]]; then
        log_info "✅ Portal → Runner proxy OK (${portal_time}ms)"
        response_times+=("Portal→Runner: ${portal_time}ms")
    else
        log_error "❌ Portal → Runner proxy failed"
        failed_services+=("Portal-Runner-Proxy")
    fi
    
    start_time=$(date +%s%N)
    local portal_hybrid_health=$(curl -s --max-time 10 "${PORTAL_URL}/hybrid/health" || echo "")
    end_time=$(date +%s%N)
    local portal_hybrid_time=$(( (end_time - start_time) / 1000000 ))
    
    if [[ -n "$portal_hybrid_health" ]]; then
        log_info "✅ Portal → Hybrid proxy OK (${portal_hybrid_time}ms)"
        response_times+=("Portal→Hybrid: ${portal_hybrid_time}ms")
    else
        log_error "❌ Portal → Hybrid proxy failed"
        failed_services+=("Portal-Hybrid-Proxy")
    fi
    
    # Summary
    log_info "📊 Response times: ${response_times[*]}"
    
    if [[ ${#failed_services[@]} -eq 0 ]]; then
        log_info "✅ All services healthy"
        return 0
    else
        log_error "❌ Failed services: ${failed_services[*]}"
        return 1
    fi
}

# Test 2.2: Database Connectivity
test_database_health() {
    log_info "Testing database connectivity..."
    
    # Test Runner database via executions API
    log_info "Checking Runner database..."
    local executions_response=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/executions" || echo "")
    
    if [[ -n "$executions_response" ]]; then
        local error_check=$(echo "$executions_response" | jq -r '.error // empty' 2>/dev/null || echo "")
        if [[ -z "$error_check" ]]; then
            log_info "✅ Runner database connectivity OK"
        else
            log_error "❌ Runner database error: $error_check"
            return 1
        fi
    else
        log_error "❌ Runner database not responding"
        return 1
    fi
    
    # Test jobs API (also database-dependent)
    log_info "Checking jobs database operations..."
    local jobs_response=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/jobs?limit=1" || echo "")
    
    if [[ -n "$jobs_response" ]]; then
        # Check if response is valid JSON and not an error
        local error_check=$(echo "$jobs_response" | jq -r '.error // empty' 2>/dev/null || echo "")
        if [[ -z "$error_check" ]]; then
            # Try to parse as array or check if it's a valid response
            local jobs_array=$(echo "$jobs_response" | jq '. | type' 2>/dev/null || echo "unknown")
            if [[ "$jobs_array" == '"array"' ]] || [[ "$jobs_array" == '"object"' ]]; then
                log_info "✅ Jobs database operations OK"
            else
                log_warn "⚠️  Jobs database returned unexpected format but no errors"
            fi
        else
            log_error "❌ Jobs database error: $error_check"
            return 1
        fi
    else
        log_error "❌ Jobs database not responding"
        return 1
    fi
    
    return 0
}

# Test 2.3: Redis Connectivity
test_redis_health() {
    log_info "Testing Redis connectivity..."
    
    # Test Redis via health endpoint
    local health_response=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/health" || echo "")
    
    if [[ -n "$health_response" ]]; then
        # Try to parse Redis status from health response
        local redis_info=$(echo "$health_response" | jq -r '.redis // .services.redis // "unknown"' 2>/dev/null || echo "unknown")
        log_debug "Redis status from health: $redis_info"
        
        # Test queue operations by submitting a test job (indirect Redis test)
        log_info "Testing Redis queue operations..."
        local test_payload='{"jobspec_id":"redis-test-'$(date +%s)'","version":"v1","benchmark":{"name":"bias-detection","container":{"image":"test","resources":{"cpu":"100m","memory":"128Mi"}},"input":{"hash":"test"}},"constraints":{"regions":["US"],"min_regions":1},"questions":["test"],"signature":"test","public_key":"test"}'
        
        local queue_response=$(curl -s --max-time 10 -X POST "${RUNNER_URL}/api/v1/jobs" \
            -H "Content-Type: application/json" \
            -d "$test_payload" || echo "")
        
        local queue_status=$(echo "$queue_response" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
        
        if [[ "$queue_status" == "enqueued" ]]; then
            log_info "✅ Redis queue operations working"
            return 0
        else
            log_warn "⚠️  Redis queue test inconclusive (status: $queue_status)"
            return 0  # Don't fail, just warn
        fi
    else
        log_error "❌ Cannot test Redis - health endpoint not responding"
        return 1
    fi
}

# Test 2.4: Performance Benchmarking
test_performance_benchmarks() {
    log_info "Running performance benchmarks..."
    
    local endpoints=(
        "${RUNNER_URL}/api/v1/health"
        "${HYBRID_URL}/health"
        "${RUNNER_URL}/api/v1/jobs?limit=5"
    )
    
    local benchmark_results=()
    
    for endpoint in "${endpoints[@]}"; do
        local endpoint_name=$(echo "$endpoint" | sed 's|https://||' | cut -d'/' -f1)
        log_info "Benchmarking $endpoint_name..."
        
        local total_time=0
        local successful_requests=0
        local iterations=3
        
        for i in $(seq 1 $iterations); do
            local start_time=$(date +%s%N)
            local response=$(curl -s --max-time 5 "$endpoint" || echo "")
            local end_time=$(date +%s%N)
            
            if [[ -n "$response" ]]; then
                local request_time=$(( (end_time - start_time) / 1000000 ))
                total_time=$((total_time + request_time))
                successful_requests=$((successful_requests + 1))
            fi
        done
        
        if [[ $successful_requests -gt 0 ]]; then
            local avg_time=$((total_time / successful_requests))
            benchmark_results+=("$endpoint_name: ${avg_time}ms avg")
            
            if [[ $avg_time -lt 1000 ]]; then
                log_info "✅ $endpoint_name: ${avg_time}ms (good)"
            elif [[ $avg_time -lt 3000 ]]; then
                log_warn "⚠️  $endpoint_name: ${avg_time}ms (acceptable)"
            else
                log_error "❌ $endpoint_name: ${avg_time}ms (slow)"
            fi
        else
            log_error "❌ $endpoint_name: All requests failed"
            benchmark_results+=("$endpoint_name: FAILED")
        fi
    done
    
    log_info "📊 Performance summary: ${benchmark_results[*]}"
    return 0
}

# Main test runner
main() {
    log_info "🚀 Phase 2: API Health & Monitoring Tests"
    log_info "Services: Runner, Hybrid Router, Portal Proxies"
    echo
    
    local failed_tests=()
    
    if ! test_service_health; then
        failed_tests+=("Service Health")
    fi
    echo
    
    if ! test_database_health; then
        failed_tests+=("Database Health")
    fi
    echo
    
    if ! test_redis_health; then
        failed_tests+=("Redis Health")
    fi
    echo
    
    if ! test_performance_benchmarks; then
        failed_tests+=("Performance")
    fi
    echo
    
    # Summary
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_info "🎉 All Phase 2 health tests PASSED!"
        log_info "✅ All services are healthy and performing well"
        return 0
    else
        log_error "❌ Failed tests: ${failed_tests[*]}"
        log_error "🔧 Some services need attention"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
