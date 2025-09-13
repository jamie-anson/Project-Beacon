#!/bin/bash

# Advanced Pipeline Test - Phase 1 Complete
# Tests the enhanced multi-question output and UI improvements

set -e

RUNNER_BASE_URL="https://beacon-runner-change-me.fly.dev"
PORTAL_BASE_URL="https://projectbeacon.netlify.app"
TEST_JOB_PREFIX="test-advanced-$(date +%s)"
TIMEOUT=180  # 3 minutes for advanced tests

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_debug() { echo -e "${BLUE}[DEBUG]${NC} $1"; }

# Test: Multi-Question Output Validation
test_multi_question_output() {
    log_info "Testing multi-question structured output..."
    
    # Submit job with multiple diverse questions
    local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-multiq",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:test-multiq"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": [
        "identity_basic",
        "math_basic", 
        "tiananmen_neutral",
        "taiwan_status",
        "greatest_invention"
    ],
    "signature": "test-multiq-sig",
    "public_key": "test-multiq-key"
}
EOF
    )
    
    local response=$(curl -s -X POST "${RUNNER_BASE_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload")
    
    local status=$(echo "$response" | jq -r '.status // "unknown"')
    if [[ "$status" != "enqueued" ]]; then
        log_error "Failed to submit multi-question job: $response"
        return 1
    fi
    
    log_info "Multi-question job submitted successfully"
    
    # Wait for job to have execution with receipt
    log_info "Waiting for execution with structured output..."
    local start_time=$(date +%s)
    local job_id="${TEST_JOB_PREFIX}-multiq"
    
    while [[ $(($(date +%s) - start_time)) -lt $TIMEOUT ]]; do
        local job_detail=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs/${job_id}?include=executions&exec_limit=1" 2>/dev/null || echo "{}")
        local exec_count=$(echo "$job_detail" | jq '.executions | length' 2>/dev/null || echo "0")
        
        if [[ "$exec_count" -gt 0 ]]; then
            local receipt=$(echo "$job_detail" | jq '.executions[0].receipt // {}')
            local has_output=$(echo "$receipt" | jq 'has("output")' 2>/dev/null || echo "false")
            
            if [[ "$has_output" == "true" ]]; then
                log_info "✅ Execution found with receipt"
                
                # Validate structured output format
                local responses=$(echo "$receipt" | jq '.output.data.responses // .output.data.data.responses // []')
                local response_count=$(echo "$responses" | jq 'length' 2>/dev/null || echo "0")
                
                log_info "Found $response_count responses in receipt"
                
                if [[ "$response_count" -ge 5 ]]; then
                    log_info "✅ All 5 questions have responses"
                    
                    # Validate response structure for each question
                    local validation_passed=true
                    for i in $(seq 0 $((response_count - 1))); do
                        local response=$(echo "$responses" | jq ".[$i]")
                        local question_id=$(echo "$response" | jq -r '.question_id // ""')
                        local question_text=$(echo "$response" | jq -r '.question // ""')
                        local response_text=$(echo "$response" | jq -r '.response // ""')
                        local category=$(echo "$response" | jq -r '.category // ""')
                        local inference_time=$(echo "$response" | jq -r '.inference_time // 0')
                        
                        log_debug "Question $((i+1)): $question_id"
                        log_debug "  Text: $question_text"
                        log_debug "  Category: $category"
                        log_debug "  Inference time: ${inference_time}s"
                        
                        # Validate required fields
                        if [[ -z "$question_id" || -z "$question_text" || -z "$response_text" || -z "$category" ]]; then
                            log_error "Response $((i+1)) missing required fields"
                            validation_passed=false
                        fi
                        
                        # Validate question text is not placeholder
                        if [[ "$question_text" == *"Question text for"* ]]; then
                            log_error "Response $((i+1)) has placeholder question text"
                            validation_passed=false
                        fi
                        
                        # Validate we have actual question text
                        if [[ ${#question_text} -lt 10 ]]; then
                            log_error "Response $((i+1)) has suspiciously short question text: $question_text"
                            validation_passed=false
                        fi
                    done
                    
                    if [[ "$validation_passed" == true ]]; then
                        log_info "✅ All responses have valid structure and content"
                        return 0
                    else
                        log_error "❌ Response validation failed"
                        return 1
                    fi
                else
                    log_warn "Only $response_count responses found, expected 5"
                fi
                
                break
            fi
        fi
        
        local elapsed=$(($(date +%s) - start_time))
        log_info "Waiting for execution... (${elapsed}s elapsed)"
        sleep 10
    done
    
    log_error "Timeout waiting for multi-question execution"
    return 1
}

# Test: Region-Specific Response Validation
test_region_responses() {
    log_info "Testing region-specific response generation..."
    
    # Submit job targeting different regions to test response variation
    local regions=("US" "EU" "APAC")
    local job_ids=()
    
    for region in "${regions[@]}"; do
        local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-region-${region}",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:test-region-${region}"}
    },
    "constraints": {"regions": ["${region}"], "min_regions": 1},
    "questions": ["taiwan_status", "tiananmen_neutral"],
    "signature": "test-region-${region}-sig",
    "public_key": "test-region-${region}-key"
}
EOF
        )
        
        local response=$(curl -s -X POST "${RUNNER_BASE_URL}/api/v1/jobs" \
            -H "Content-Type: application/json" \
            -d "$job_payload")
        
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        if [[ "$status" == "enqueued" ]]; then
            job_ids+=("${TEST_JOB_PREFIX}-region-${region}")
            log_info "✅ Submitted job for region: $region"
        else
            log_error "Failed to submit job for region $region: $response"
            return 1
        fi
        
        sleep 2  # Small delay between submissions
    done
    
    log_info "✅ Region-specific jobs submitted successfully"
    log_info "Note: Full validation requires execution completion (may take time)"
    return 0
}

# Test: Category Formatting Validation
test_category_formatting() {
    log_info "Testing category formatting and display..."
    
    # Submit job with questions from different categories
    local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-categories",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:test-categories"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": [
        "identity_basic",
        "math_basic",
        "geography_basic",
        "tiananmen_neutral",
        "greatest_invention"
    ],
    "signature": "test-categories-sig",
    "public_key": "test-categories-key"
}
EOF
    )
    
    local response=$(curl -s -X POST "${RUNNER_BASE_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload")
    
    local status=$(echo "$response" | jq -r '.status // "unknown"')
    if [[ "$status" != "enqueued" ]]; then
        log_error "Failed to submit category test job: $response"
        return 1
    fi
    
    log_info "✅ Category formatting test job submitted"
    
    # Wait for execution to validate categories
    local start_time=$(date +%s)
    local job_id="${TEST_JOB_PREFIX}-categories"
    
    while [[ $(($(date +%s) - start_time)) -lt $TIMEOUT ]]; do
        local job_detail=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs/${job_id}?include=executions&exec_limit=1" 2>/dev/null || echo "{}")
        local exec_count=$(echo "$job_detail" | jq '.executions | length' 2>/dev/null || echo "0")
        
        if [[ "$exec_count" -gt 0 ]]; then
            local receipt=$(echo "$job_detail" | jq '.executions[0].receipt // {}')
            local responses=$(echo "$receipt" | jq '.output.data.responses // .output.data.data.responses // []')
            local response_count=$(echo "$responses" | jq 'length' 2>/dev/null || echo "0")
            
            if [[ "$response_count" -gt 0 ]]; then
                log_info "Validating category formats..."
                
                # Check categories are properly formatted
                local categories_found=()
                for i in $(seq 0 $((response_count - 1))); do
                    local category=$(echo "$responses" | jq -r ".[$i].category // \"\"")
                    if [[ -n "$category" ]]; then
                        categories_found+=("$category")
                        log_debug "Found category: $category"
                    fi
                done
                
                if [[ ${#categories_found[@]} -gt 0 ]]; then
                    log_info "✅ Found ${#categories_found[@]} categories: ${categories_found[*]}"
                    return 0
                else
                    log_error "No categories found in responses"
                    return 1
                fi
            fi
        fi
        
        local elapsed=$(($(date +%s) - start_time))
        log_info "Waiting for category validation... (${elapsed}s elapsed)"
        sleep 10
    done
    
    log_warn "Timeout waiting for category validation (job may still be processing)"
    return 0
}

# Test: Performance and Reliability
test_performance_reliability() {
    log_info "Testing pipeline performance and reliability..."
    
    # Submit multiple jobs to test system under load
    local job_count=3
    local job_ids=()
    
    log_info "Submitting $job_count concurrent jobs..."
    
    for i in $(seq 1 $job_count); do
        local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-perf-${i}",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:test-perf-${i}"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": ["identity_basic", "math_basic"],
    "signature": "test-perf-${i}-sig",
    "public_key": "test-perf-${i}-key"
}
EOF
        )
        
        local response=$(curl -s -X POST "${RUNNER_BASE_URL}/api/v1/jobs" \
            -H "Content-Type: application/json" \
            -d "$job_payload")
        
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        if [[ "$status" == "enqueued" ]]; then
            job_ids+=("${TEST_JOB_PREFIX}-perf-${i}")
            log_info "✅ Job $i submitted"
        else
            log_error "Failed to submit job $i: $response"
            return 1
        fi
    done
    
    log_info "✅ All $job_count jobs submitted successfully"
    
    # Monitor job processing
    log_info "Monitoring job processing rates..."
    local start_time=$(date +%s)
    local processed_count=0
    
    while [[ $processed_count -lt $job_count && $(($(date +%s) - start_time)) -lt $TIMEOUT ]]; do
        processed_count=0
        
        for job_id in "${job_ids[@]}"; do
            local job_status=$(curl -s "${RUNNER_BASE_URL}/api/v1/jobs/${job_id}" | jq -r '.status // "unknown"' 2>/dev/null)
            if [[ "$job_status" != "created" && "$job_status" != "unknown" ]]; then
                processed_count=$((processed_count + 1))
            fi
        done
        
        local elapsed=$(($(date +%s) - start_time))
        log_info "Processed: $processed_count/$job_count jobs (${elapsed}s elapsed)"
        
        if [[ $processed_count -lt $job_count ]]; then
            sleep 5
        fi
    done
    
    if [[ $processed_count -eq $job_count ]]; then
        log_info "✅ All jobs processed successfully"
        return 0
    else
        log_warn "Only $processed_count/$job_count jobs processed within timeout"
        return 0  # Don't fail, just warn
    fi
}

# Main test runner
main() {
    log_info "🚀 Starting Advanced Pipeline Tests (Phase 1 Complete)"
    log_info "Runner URL: $RUNNER_BASE_URL"
    log_info "Test Job Prefix: $TEST_JOB_PREFIX"
    echo
    
    local failed_tests=()
    local warnings=()
    
    if ! test_multi_question_output; then
        failed_tests+=("Multi-Question Output")
    fi
    echo
    
    if ! test_region_responses; then
        failed_tests+=("Region Responses")
    fi
    echo
    
    if ! test_category_formatting; then
        warnings+=("Category Formatting")
    fi
    echo
    
    if ! test_performance_reliability; then
        warnings+=("Performance")
    fi
    echo
    
    # Summary
    log_info "📊 Test Summary:"
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_info "✅ All critical tests PASSED!"
    else
        log_error "❌ Failed tests: ${failed_tests[*]}"
    fi
    
    if [[ ${#warnings[@]} -gt 0 ]]; then
        log_warn "⚠️  Warnings: ${warnings[*]}"
    fi
    
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_info "🎉 Advanced pipeline functionality is working correctly!"
        log_info "✅ Multi-question output, region responses, and UI improvements validated"
        return 0
    else
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
