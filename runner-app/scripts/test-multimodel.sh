#!/bin/bash

# Multi-Model Testing Script for Project Beacon Runner
# This script runs the comprehensive test suite for multi-model functionality

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
RUNNER_DIR="/Users/Jammie/Desktop/Project Beacon/runner-app"
TEST_TIMEOUT="30s"
VERBOSE=${VERBOSE:-false}

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_header() {
    echo
    print_status $BLUE "=================================="
    print_status $BLUE "$1"
    print_status $BLUE "=================================="
}

print_success() {
    print_status $GREEN "✅ $1"
}

print_error() {
    print_status $RED "❌ $1"
}

print_warning() {
    print_status $YELLOW "⚠️  $1"
}

# Function to run tests with proper error handling
run_test() {
    local test_name=$1
    local test_command=$2
    local test_dir=${3:-$RUNNER_DIR}
    
    print_status $BLUE "Running: $test_name"
    
    if [ "$VERBOSE" = "true" ]; then
        echo "Command: $test_command"
        echo "Directory: $test_dir"
    fi
    
    cd "$test_dir"
    
    if eval "$test_command"; then
        print_success "$test_name passed"
        return 0
    else
        print_error "$test_name failed"
        return 1
    fi
}

# Function to check if required dependencies are available
check_dependencies() {
    print_header "Checking Dependencies"
    
    local deps_ok=true
    
    # Check Go
    if command -v go >/dev/null 2>&1; then
        local go_version=$(go version | cut -d' ' -f3)
        print_success "Go found: $go_version"
    else
        print_error "Go not found"
        deps_ok=false
    fi
    
    # Check if we're in the right directory
    if [ -f "$RUNNER_DIR/go.mod" ]; then
        print_success "Runner app directory found"
    else
        print_error "Runner app directory not found at $RUNNER_DIR"
        deps_ok=false
    fi
    
    # Check for test files
    local test_files=(
        "internal/api/processors/jobspec_processor_test.go"
        "internal/worker/job_runner_multimodel_test.go"
        "internal/api/executions_handler_multimodel_test.go"
        "e2e/multimodel_e2e_test.go"
        "pkg/models/multimodel_signature_test.go"
    )
    
    for test_file in "${test_files[@]}"; do
        if [ -f "$RUNNER_DIR/$test_file" ]; then
            print_success "Test file found: $test_file"
        else
            print_warning "Test file not found: $test_file"
        fi
    done
    
    if [ "$deps_ok" = false ]; then
        print_error "Dependencies check failed"
        exit 1
    fi
}

# Function to run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    
    local unit_tests=(
        "JobSpec Processor Tests:go test -timeout $TEST_TIMEOUT -v ./internal/api/processors -run TestNormalizeModelsFromMetadata"
        "Multi-Model Job Runner Tests:go test -timeout $TEST_TIMEOUT -v ./internal/worker -run TestExecuteMultiModelJob"
        "Signature Verification Tests:go test -timeout $TEST_TIMEOUT -v ./pkg/models -run TestMultiModelJobSignatureVerification"
        "API Handler Tests:go test -timeout $TEST_TIMEOUT -v ./internal/api -run TestExecutionsHandler.*MultiModel"
    )
    
    local unit_passed=0
    local unit_total=${#unit_tests[@]}
    
    for test_spec in "${unit_tests[@]}"; do
        local test_name=$(echo "$test_spec" | cut -d':' -f1)
        local test_cmd=$(echo "$test_spec" | cut -d':' -f2)
        
        if run_test "$test_name" "$test_cmd"; then
            ((unit_passed++))
        fi
    done
    
    print_status $BLUE "Unit Tests Summary: $unit_passed/$unit_total passed"
    
    if [ $unit_passed -eq $unit_total ]; then
        print_success "All unit tests passed"
        return 0
    else
        print_error "Some unit tests failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    local integration_tests=(
        "Model Normalization Integration:go test -timeout $TEST_TIMEOUT -v ./internal/api/processors -run TestNormalizeModelsFromMetadata_EdgeCases"
        "Bounded Concurrency Tests:go test -timeout $TEST_TIMEOUT -v ./internal/worker -run TestExecuteMultiModelJob_BoundedConcurrency"
        "Metadata Safety Tests:go test -timeout $TEST_TIMEOUT -v ./internal/worker -run TestExecuteMultiModelJob_MetadataSafety"
        "Error Handling Tests:go test -timeout $TEST_TIMEOUT -v ./internal/worker -run TestExecuteMultiModelJob_ErrorHandling"
        "Signature Integrity Tests:go test -timeout $TEST_TIMEOUT -v ./pkg/models -run TestSignatureVerificationIntegrity"
    )
    
    local integration_passed=0
    local integration_total=${#integration_tests[@]}
    
    for test_spec in "${integration_tests[@]}"; do
        local test_name=$(echo "$test_spec" | cut -d':' -f1)
        local test_cmd=$(echo "$test_spec" | cut -d':' -f2)
        
        if run_test "$test_name" "$test_cmd"; then
            ((integration_passed++))
        fi
    done
    
    print_status $BLUE "Integration Tests Summary: $integration_passed/$integration_total passed"
    
    if [ $integration_passed -eq $integration_total ]; then
        print_success "All integration tests passed"
        return 0
    else
        print_error "Some integration tests failed"
        return 1
    fi
}

# Function to run E2E tests (optional, requires running services)
run_e2e_tests() {
    print_header "Running E2E Tests"
    
    # Check if runner is available
    local runner_url="http://localhost:8090"
    if curl -s --connect-timeout 5 "$runner_url/health" >/dev/null 2>&1; then
        print_success "Runner service is available at $runner_url"
        
        local e2e_tests=(
            "Multi-Model Workflow E2E:go test -timeout 60s -v ./e2e -run TestMultiModelWorkflow_E2E"
            "Job Normalization E2E:go test -timeout 30s -v ./e2e -run TestMultiModelJobNormalization_E2E"
        )
        
        local e2e_passed=0
        local e2e_total=${#e2e_tests[@]}
        
        for test_spec in "${e2e_tests[@]}"; do
            local test_name=$(echo "$test_spec" | cut -d':' -f1)
            local test_cmd=$(echo "$test_spec" | cut -d':' -f2)
            
            if run_test "$test_name" "$test_cmd"; then
                ((e2e_passed++))
            fi
        done
        
        print_status $BLUE "E2E Tests Summary: $e2e_passed/$e2e_total passed"
        
        if [ $e2e_passed -eq $e2e_total ]; then
            print_success "All E2E tests passed"
            return 0
        else
            print_error "Some E2E tests failed"
            return 1
        fi
    else
        print_warning "Runner service not available at $runner_url"
        print_warning "Skipping E2E tests (start runner with 'go run cmd/runner/main.go' to enable)"
        return 0
    fi
}

# Function to run all tests for a specific component
run_component_tests() {
    local component=$1
    
    case $component in
        "processor")
            run_test "JobSpec Processor Tests" "go test -timeout $TEST_TIMEOUT -v ./internal/api/processors"
            ;;
        "worker")
            run_test "Job Runner Tests" "go test -timeout $TEST_TIMEOUT -v ./internal/worker -run TestExecuteMultiModelJob"
            ;;
        "api")
            run_test "API Handler Tests" "go test -timeout $TEST_TIMEOUT -v ./internal/api -run TestExecutionsHandler"
            ;;
        "models")
            run_test "Models Package Tests" "go test -timeout $TEST_TIMEOUT -v ./pkg/models -run TestMultiModel"
            ;;
        "e2e")
            run_e2e_tests
            ;;
        *)
            print_error "Unknown component: $component"
            print_status $YELLOW "Available components: processor, worker, api, models, e2e"
            exit 1
            ;;
    esac
}

# Function to run performance tests
run_performance_tests() {
    print_header "Running Performance Tests"
    
    # Check if runner is available for performance tests
    local runner_url="http://localhost:8090"
    if curl -s --connect-timeout 5 "$runner_url/health" >/dev/null 2>&1; then
        run_test "Multi-Model Performance Test" "go test -timeout 120s -v ./e2e -run TestMultiModelJobPerformance_E2E"
    else
        print_warning "Runner service not available - skipping performance tests"
    fi
}

# Function to generate test report
generate_test_report() {
    print_header "Test Report Generation"
    
    local report_file="$RUNNER_DIR/test-report-multimodel.txt"
    
    {
        echo "Multi-Model Test Report"
        echo "======================"
        echo "Generated: $(date)"
        echo "Runner Directory: $RUNNER_DIR"
        echo ""
        
        echo "Test Coverage:"
        go test -coverprofile=coverage.out ./internal/api/processors ./internal/worker ./pkg/models 2>/dev/null || true
        if [ -f coverage.out ]; then
            go tool cover -func=coverage.out | tail -1
            rm -f coverage.out
        fi
        
        echo ""
        echo "Test Files:"
        find . -name "*multimodel*test.go" -o -name "*test.go" | grep -E "(processor|worker|multimodel)" | sort
        
    } > "$report_file"
    
    print_success "Test report generated: $report_file"
}

# Main function
main() {
    local command=${1:-"all"}
    
    print_header "Project Beacon Multi-Model Test Suite"
    print_status $BLUE "Command: $command"
    print_status $BLUE "Directory: $RUNNER_DIR"
    print_status $BLUE "Timeout: $TEST_TIMEOUT"
    
    case $command in
        "deps"|"dependencies")
            check_dependencies
            ;;
        "unit")
            check_dependencies
            run_unit_tests
            ;;
        "integration")
            check_dependencies
            run_integration_tests
            ;;
        "e2e")
            check_dependencies
            run_e2e_tests
            ;;
        "performance"|"perf")
            check_dependencies
            run_performance_tests
            ;;
        "component")
            if [ -z "$2" ]; then
                print_error "Component name required"
                print_status $YELLOW "Usage: $0 component <processor|worker|api|models|e2e>"
                exit 1
            fi
            check_dependencies
            run_component_tests "$2"
            ;;
        "report")
            check_dependencies
            generate_test_report
            ;;
        "all")
            check_dependencies
            local all_passed=true
            
            if ! run_unit_tests; then
                all_passed=false
            fi
            
            if ! run_integration_tests; then
                all_passed=false
            fi
            
            # E2E tests are optional (require running services)
            run_e2e_tests || true
            
            generate_test_report
            
            if [ "$all_passed" = true ]; then
                print_header "🎉 All Critical Tests Passed!"
                print_success "Multi-model functionality is ready for deployment"
            else
                print_header "❌ Some Tests Failed"
                print_error "Please fix failing tests before deployment"
                exit 1
            fi
            ;;
        "help"|"-h"|"--help")
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  all           Run all tests (default)"
            echo "  deps          Check dependencies only"
            echo "  unit          Run unit tests only"
            echo "  integration   Run integration tests only"
            echo "  e2e           Run E2E tests only (requires running services)"
            echo "  performance   Run performance tests only"
            echo "  component     Run tests for specific component"
            echo "  report        Generate test report only"
            echo "  help          Show this help"
            echo ""
            echo "Environment Variables:"
            echo "  VERBOSE=true  Enable verbose output"
            echo ""
            echo "Examples:"
            echo "  $0                          # Run all tests"
            echo "  $0 unit                     # Run unit tests only"
            echo "  $0 component processor      # Run processor tests only"
            echo "  VERBOSE=true $0 integration # Run integration tests with verbose output"
            ;;
        *)
            print_error "Unknown command: $command"
            print_status $YELLOW "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"
