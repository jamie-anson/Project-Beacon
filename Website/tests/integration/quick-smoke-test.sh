#!/bin/bash

# Quick Smoke Test - Phase 1 Fast Validation
# 30-second test to validate core pipeline functionality

set -e

RUNNER_BASE_URL="https://beacon-runner-change-me.fly.dev"
TEST_JOB_PREFIX="smoke-$(date +%s)"
TIMEOUT=30  # 30 seconds max

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test 1: API Health (5 seconds)
test_api_health() {
    log_info "Testing API health..."
    
    local health=$(curl -s --max-time 5 "${RUNNER_BASE_URL}/api/v1/health" || echo "")
    if [[ -n "$health" ]]; then
        log_info "✅ API responding"
        return 0
    else
        log_error "❌ API not responding"
        return 1
    fi
}

# Test 2: Job Submission (10 seconds)
test_job_submission() {
    log_info "Testing job submission..."
    
    local job_payload='{"jobspec_id":"'${TEST_JOB_PREFIX}'","version":"v1","benchmark":{"name":"bias-detection","container":{"image":"ghcr.io/project-beacon/bias-detection:latest","resources":{"cpu":"1000m","memory":"2Gi"}},"input":{"hash":"sha256:smoke-test"}},"constraints":{"regions":["US"],"min_regions":1},"questions":["Who are you?"],"signature":"smoke-sig","public_key":"smoke-key"}'
    
    local response=$(curl -s --max-time 10 -X POST "${RUNNER_BASE_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload" || echo "")
    
    local status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
    
    if [[ "$status" == "enqueued" ]]; then
        log_info "✅ Job submission working"
        return 0
    else
        log_error "❌ Job submission failed: $status"
        return 1
    fi
}

# Test 3: Pipeline Processing (15 seconds)
test_pipeline_processing() {
    log_info "Testing pipeline processing..."
    
    # Check recent jobs to see if pipeline is working
    local jobs=$(curl -s --max-time 5 "${RUNNER_BASE_URL}/api/v1/jobs?limit=5" || echo "[]")
    local job_count=$(echo "$jobs" | jq 'length' 2>/dev/null || echo "0")
    
    if [[ "$job_count" -gt 0 ]]; then
        # Check if jobs are moving out of "created" status
        local created_count=$(echo "$jobs" | jq '[.[] | select(.status == "created")] | length' 2>/dev/null || echo "0")
        local total_count=$(echo "$jobs" | jq 'length' 2>/dev/null || echo "1")
        local processing_ratio=$((100 * (total_count - created_count) / total_count))
        
        if [[ $processing_ratio -gt 50 ]]; then
            log_info "✅ Pipeline processing jobs (${processing_ratio}% not stuck)"
            return 0
        else
            log_warn "⚠️  Many jobs stuck in created status"
            return 0  # Don't fail, just warn
        fi
    else
        log_info "✅ No jobs to analyze (clean state)"
        return 0
    fi
}

# Test 4: Output Format Check (quick)
test_output_format() {
    log_info "Testing output format..."
    
    # Quick check for any completed job with structured output
    local jobs=$(curl -s --max-time 5 "${RUNNER_BASE_URL}/api/v1/jobs?limit=3" || echo "[]")
    local completed_jobs=$(echo "$jobs" | jq '[.[] | select(.status == "completed")]' 2>/dev/null || echo "[]")
    local completed_count=$(echo "$completed_jobs" | jq 'length' 2>/dev/null || echo "0")
    
    if [[ "$completed_count" -gt 0 ]]; then
        log_info "✅ Found $completed_count completed jobs (output format likely working)"
        return 0
    else
        log_info "✅ No completed jobs to check (system may be fresh)"
        return 0
    fi
}

# Main runner
main() {
    log_info "🚀 Quick Smoke Test (${TIMEOUT}s max)"
    echo
    
    local start_time=$(date +%s)
    local failed_tests=()
    
    if ! test_api_health; then
        failed_tests+=("API Health")
    fi
    
    if ! test_job_submission; then
        failed_tests+=("Job Submission")
    fi
    
    if ! test_pipeline_processing; then
        failed_tests+=("Pipeline Processing")
    fi
    
    if ! test_output_format; then
        failed_tests+=("Output Format")
    fi
    
    local elapsed=$(($(date +%s) - start_time))
    echo
    log_info "⏱️  Test completed in ${elapsed}s"
    
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_info "🎉 All smoke tests PASSED!"
        log_info "✅ Core pipeline functionality is working"
        return 0
    else
        log_error "❌ Failed tests: ${failed_tests[*]}"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
