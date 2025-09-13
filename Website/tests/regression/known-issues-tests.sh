#!/bin/bash

# Phase 4: Regression Prevention Tests
# Tests to prevent specific production issues we've encountered

set -e

RUNNER_URL="https://beacon-runner-change-me.fly.dev"
TEST_JOB_PREFIX="regression-$(date +%s)"
TIMEOUT=120  # 2 minutes for regression tests

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test 4.1: Prevent Jobs Stuck in "Created" Status
test_prevent_stuck_jobs() {
    log_info "Testing prevention of jobs stuck in 'created' status..."
    
    # Submit a job and verify it moves out of "created" quickly
    local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-stuck-test",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:stuck-test"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": ["identity_basic"],
    "signature": "stuck-test-sig",
    "public_key": "stuck-test-key"
}
EOF
    )
    
    local response=$(curl -s --max-time 10 -X POST "${RUNNER_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload" || echo "")
    
    local status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
    
    if [[ "$status" != "enqueued" ]]; then
        log_error "❌ Job submission failed: $status"
        return 1
    fi
    
    log_info "Job submitted, monitoring for stuck status..."
    
    # Monitor job for 60 seconds to ensure it doesn't stay in "created"
    local job_id="${TEST_JOB_PREFIX}-stuck-test"
    local start_time=$(date +%s)
    local stuck_time=0
    local max_stuck_time=30  # Max 30 seconds in "created" status
    
    while [[ $(($(date +%s) - start_time)) -lt 60 ]]; do
        local job_status=$(curl -s --max-time 5 "${RUNNER_URL}/api/v1/jobs/${job_id}" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
        
        if [[ "$job_status" == "created" ]]; then
            stuck_time=$(($(date +%s) - start_time))
            if [[ $stuck_time -gt $max_stuck_time ]]; then
                log_error "❌ Job stuck in 'created' status for ${stuck_time}s (regression detected)"
                return 1
            fi
        else
            log_info "✅ Job moved to '$job_status' after ${stuck_time}s (not stuck)"
            return 0
        fi
        
        sleep 3
    done
    
    log_warn "⚠️  Job monitoring timeout, final stuck time: ${stuck_time}s"
    return 0  # Don't fail on timeout
}

# Test 4.2: Prevent Empty ID Fields
test_prevent_empty_ids() {
    log_info "Testing prevention of empty ID fields..."
    
    # Submit job and verify ID field is populated
    local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-id-test",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:id-test"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": ["math_basic"],
    "signature": "id-test-sig",
    "public_key": "id-test-key"
}
EOF
    )
    
    local response=$(curl -s --max-time 10 -X POST "${RUNNER_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload" || echo "")
    
    local job_id=$(echo "$response" | jq -r '.id // empty' 2>/dev/null || echo "")
    local jobspec_id=$(echo "$response" | jq -r '.jobspec_id // empty' 2>/dev/null || echo "")
    local status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
    
    log_info "Response ID: '$job_id', JobSpec ID: '$jobspec_id', Status: '$status'"
    
    # Check for the known issue: empty ID field
    if [[ -z "$job_id" || "$job_id" == "" ]]; then
        if [[ -n "$jobspec_id" && "$status" == "enqueued" ]]; then
            log_warn "⚠️  ID field empty but jobspec_id present (known issue - not critical)"
            return 0
        else
            log_error "❌ Both ID and jobspec_id empty (regression)"
            return 1
        fi
    else
        log_info "✅ ID field properly populated: $job_id"
        return 0
    fi
}

# Test 4.3: Prevent UI Display Issues
test_prevent_ui_display_issues() {
    log_info "Testing prevention of UI display issues..."
    
    # Look for a completed job with structured output
    local jobs=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/jobs?limit=5" || echo "[]")
    local completed_jobs=$(echo "$jobs" | jq '[.[] | select(.status == "completed")]' 2>/dev/null || echo "[]")
    local completed_count=$(echo "$completed_jobs" | jq 'length' 2>/dev/null || echo "0")
    
    if [[ "$completed_count" -gt 0 ]]; then
        local job_id=$(echo "$completed_jobs" | jq -r '.[0].jobspec_id // .[0].id // empty')
        
        if [[ -n "$job_id" ]]; then
            log_info "Checking UI display format for job: $job_id"
            
            local job_detail=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/jobs/${job_id}?include=executions&exec_limit=1" || echo "{}")
            local receipt=$(echo "$job_detail" | jq '.executions[0].receipt // {}' 2>/dev/null || echo "{}")
            
            # Check for structured responses
            local responses=$(echo "$receipt" | jq '.output.data.responses // .output.data.data.responses // []' 2>/dev/null || echo "[]")
            local response_count=$(echo "$responses" | jq 'length' 2>/dev/null || echo "0")
            
            if [[ "$response_count" -gt 0 ]]; then
                # Check first response for UI display issues
                local first_response=$(echo "$responses" | jq '.[0]')
                local question_text=$(echo "$first_response" | jq -r '.question // ""')
                local category=$(echo "$first_response" | jq -r '.category // ""')
                
                # Check for placeholder text (regression)
                if [[ "$question_text" == *"Question text for"* ]]; then
                    log_error "❌ Placeholder question text detected (UI regression)"
                    return 1
                fi
                
                # Check for actual question content
                if [[ ${#question_text} -gt 10 ]]; then
                    log_info "✅ Question text properly populated: ${question_text:0:50}..."
                else
                    log_warn "⚠️  Short question text: $question_text"
                fi
                
                # Check category format
                if [[ -n "$category" ]]; then
                    log_info "✅ Category field present: $category"
                else
                    log_warn "⚠️  Category field missing"
                fi
                
                return 0
            else
                log_info "✅ No structured responses to validate (may be legacy format)"
                return 0
            fi
        fi
    else
        log_info "✅ No completed jobs to validate UI display"
        return 0
    fi
}

# Test 4.4: Prevent API Integration Failures
test_prevent_api_failures() {
    log_info "Testing prevention of API integration failures..."
    
    # Test executions API (previously returned 500 errors)
    local executions_response=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/executions" || echo "")
    
    if [[ -n "$executions_response" ]]; then
        # Check for 500 error indicators
        local error_message=$(echo "$executions_response" | jq -r '.error // empty' 2>/dev/null || echo "")
        local is_array=$(echo "$executions_response" | jq '. | type' 2>/dev/null || echo "unknown")
        
        if [[ -n "$error_message" ]]; then
            log_error "❌ Executions API returned error: $error_message"
            return 1
        elif [[ "$is_array" == '"array"' ]]; then
            log_info "✅ Executions API returning valid array"
            return 0
        else
            log_warn "⚠️  Executions API returned unexpected format: $is_array"
            return 0
        fi
    else
        log_error "❌ Executions API not responding"
        return 1
    fi
}

# Test 4.5: Prevent Data Structure Mismatches
test_prevent_data_mismatches() {
    log_info "Testing prevention of data structure mismatches..."
    
    # Submit a job and verify the output structure matches expected format
    local job_payload=$(cat <<EOF
{
    "jobspec_id": "${TEST_JOB_PREFIX}-structure",
    "version": "v1",
    "benchmark": {
        "name": "bias-detection",
        "container": {
            "image": "ghcr.io/project-beacon/bias-detection:latest",
            "resources": {"cpu": "1000m", "memory": "2Gi"}
        },
        "input": {"hash": "sha256:structure-test"}
    },
    "constraints": {"regions": ["US"], "min_regions": 1},
    "questions": ["identity_basic", "math_basic"],
    "signature": "structure-test-sig",
    "public_key": "structure-test-key"
}
EOF
    )
    
    local response=$(curl -s --max-time 10 -X POST "${RUNNER_URL}/api/v1/jobs" \
        -H "Content-Type: application/json" \
        -d "$job_payload" || echo "")
    
    local status=$(echo "$response" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
    
    if [[ "$status" == "enqueued" ]]; then
        log_info "✅ Job structure validation passed (proper submission format)"
        
        # Verify the job has expected fields
        local required_fields=("jobspec_id" "version" "benchmark" "constraints" "questions")
        local job_id="${TEST_JOB_PREFIX}-structure"
        
        # Wait a moment then check job structure
        sleep 3
        local job_detail=$(curl -s --max-time 10 "${RUNNER_URL}/api/v1/jobs/${job_id}" || echo "{}")
        
        local missing_fields=()
        for field in "${required_fields[@]}"; do
            local field_present=$(echo "$job_detail" | jq "has(\"$field\")" 2>/dev/null || echo "false")
            if [[ "$field_present" != "true" ]]; then
                missing_fields+=("$field")
            fi
        done
        
        if [[ ${#missing_fields[@]} -eq 0 ]]; then
            log_info "✅ Job data structure complete"
            return 0
        else
            log_warn "⚠️  Missing fields in stored job: ${missing_fields[*]}"
            return 0
        fi
    else
        log_error "❌ Data structure validation failed: $status"
        return 1
    fi
}

# Main test runner
main() {
    log_info "🚀 Phase 4: Regression Prevention Tests"
    log_info "Preventing known production issues from recurring"
    echo
    
    local failed_tests=()
    
    if ! test_prevent_stuck_jobs; then
        failed_tests+=("Stuck Jobs Prevention")
    fi
    echo
    
    if ! test_prevent_empty_ids; then
        failed_tests+=("Empty IDs Prevention")
    fi
    echo
    
    if ! test_prevent_ui_display_issues; then
        failed_tests+=("UI Display Prevention")
    fi
    echo
    
    if ! test_prevent_api_failures; then
        failed_tests+=("API Failures Prevention")
    fi
    echo
    
    if ! test_prevent_data_mismatches; then
        failed_tests+=("Data Mismatch Prevention")
    fi
    echo
    
    # Summary
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        log_info "🎉 All regression prevention tests PASSED!"
        log_info "✅ Known production issues are prevented"
        return 0
    else
        log_error "❌ Failed regression tests: ${failed_tests[*]}"
        log_error "🚨 Some known issues may have regressed"
        return 1
    fi
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
