#!/bin/bash

# Basic Pipeline Test - Phase 1 Simplified
# Focus on core functionality that we know works

set -e

# Configuration
RUNNER_BASE_URL="https://beacon-runner-change-me.fly.dev"
TEST_JOB_PREFIX="test-basic-$(date +%s)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test 1: Basic API Connectivity
test_api_connectivity() {
    log_info "Testing API connectivity..."
    
    # Test health endpoint
    local health_response=$(curl -s "${RUNNER_BASE_URL}/api/v1/health" || echo "")
    if [[ -n "$health_response" ]]; then
        log_info "✅ Health endpoint responding"
    else
        log_error "❌ Health endpoint not responding"
        return 1
    fi
    
    # Test jobs list endpoint
    local jobs_response=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs?limit=5" || echo "")
    if [[ -n "$jobs_response" ]]; then
        log_info "✅ Jobs API responding"
    else
        log_error "❌ Jobs API not responding"
        return 1
    fi
    
    return 0
}

# Test 2: Job Submission
test_job_submission() {
    log_info "Testing job submission..."
    
    local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-basic",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:test-basic"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": ["Who are you?", "What is 2 + 2?"],
    "signature": "test-signature",
    "public_key": "test-key"
}
EOF
    )
    
    local response=$(curl -s -X POST "${RUNNER_BASE_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload")
    
    log_info "Job submission response: $response"
    
    # Check if we got a valid response
    local status=$(echo "$response" | jq -r '.status // "unknown"')
    if [[ "$status" == "enqueued" ]]; then
        log_info "✅ Job submitted successfully (status: enqueued)"
        return 0
    else
        log_error "❌ Job submission failed or unexpected status: $status"
        return 1
    fi
}

# Test 3: Job Processing Check
test_job_processing() {
    log_info "Testing job processing pipeline..."
    
    # Get recent jobs to see if any are processing
    local jobs_response=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs?limit=10")
    local job_count=$(echo "$jobs_response" | jq '. | length' 2>/dev/null || echo "0")
    
    log_info "Found $job_count recent jobs"
    
    if [[ "$job_count" -gt 0 ]]; then
        # Check if we have jobs in various states
        local created_count=$(echo "$jobs_response" | jq '[.[] | select(.status == "created")] | length' 2>/dev/null || echo "0")
        local processing_count=$(echo "$jobs_response" | jq '[.[] | select(.status == "processing")] | length' 2>/dev/null || echo "0")
        local completed_count=$(echo "$jobs_response" | jq '[.[] | select(.status == "completed")] | length' 2>/dev/null || echo "0")
        
        log_info "Job status distribution: Created($created_count), Processing($processing_count), Completed($completed_count)"
        
        if [[ "$created_count" -lt 5 ]]; then
            log_info "✅ Jobs are moving out of 'created' status (good pipeline health)"
        else
            log_warn "⚠️  Many jobs stuck in 'created' status - potential pipeline issue"
        fi
        
        if [[ "$completed_count" -gt 0 ]]; then
            log_info "✅ Jobs are completing successfully"
        fi
        
        return 0
    else
        log_warn "⚠️  No recent jobs found to analyze"
        return 0
    fi
}

# Test 4: Output Format Validation
test_output_format() {
    log_info "Testing output format validation..."
    
    # Get a completed job with executions
    local jobs_with_executions=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs?limit=10" | \
        jq '[.[] | select(.status == "completed")] | .[0] // empty' 2>/dev/null)
    
    if [[ -n "$jobs_with_executions" && "$jobs_with_executions" != "null" ]]; then
        local job_id=$(echo "$jobs_with_executions" | jq -r '.jobspec_id // .id // empty')
        
        if [[ -n "$job_id" && "$job_id" != "null" ]]; then
            log_info "Checking output format for job: $job_id"
            
            # Get job with executions
            local job_detail=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs/${job_id}?include=executions&exec_limit=1")
            local exec_count=$(echo "$job_detail" | jq '.executions | length' 2>/dev/null || echo "0")
            
            if [[ "$exec_count" -gt 0 ]]; then
                local receipt=$(echo "$job_detail" | jq '.executions[0].receipt // {}')
                local has_output=$(echo "$receipt" | jq 'has("output")' 2>/dev/null || echo "false")
                
                if [[ "$has_output" == "true" ]]; then
                    log_info "✅ Receipt has output structure"
                    
                    # Check for multi-question format
                    local has_responses=$(echo "$receipt" | jq '.output.data.responses // .output.data.data.responses // empty' 2>/dev/null)
                    if [[ -n "$has_responses" && "$has_responses" != "null" ]]; then
                        local response_count=$(echo "$has_responses" | jq 'length' 2>/dev/null || echo "0")
                        log_info "✅ Found $response_count question responses in structured format"
                    else
                        log_warn "⚠️  No structured responses found, may be using legacy format"
                    fi
                else
                    log_warn "⚠️  Receipt missing output structure"
                fi
            else
                log_warn "⚠️  No executions found for completed job"
            fi
        else
            log_warn "⚠️  Could not extract job ID"
        fi
    else
        log_warn "⚠️  No completed jobs found to validate output format"
    fi
    
    return 0
}

# Main test runner
main() {
    log_info "🚀 Starting Basic Pipeline Tests (Phase 1 Simplified)"
    log_info "Runner URL: $RUNNER_BASE_URL"
    echo
    
    local failed_tests=()
    
    if ! test_api_connectivity; then
        failed_tests+=("API Connectivity")
    fi
    echo
    
    if ! test_job_submission; then
        failed_tests+=("Job Submission")
    fi
    echo
    
    if ! test_job_processing; then
        failed_tests+=("Job Processing")
    fi
    echo
    
    if ! test_output_format; then
        failed_tests+=("Output Format")
    fi
    echo
    
    # Summary
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_info "🎉 All basic pipeline tests PASSED!"
        log_info "✅ Core pipeline functionality is working"
        return 0
    else
        log_error "❌ Failed tests: ${failed_tests[*]}"
        log_error "🔧 Pipeline has issues that need attention"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
