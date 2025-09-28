#!/bin/bash

# Multi-Model Test Validation Script
# This script validates that all multi-model tests compile and basic functionality works

set -e

RUNNER_DIR="/Users/Jammie/Desktop/Project Beacon/runner-app"
cd "$RUNNER_DIR"

echo "🔍 Validating Multi-Model Test Suite..."
echo "Directory: $RUNNER_DIR"
echo

# Check if test files exist
echo "📁 Checking test files..."
test_files=(
    "internal/api/processors/jobspec_processor_test.go"
    "internal/worker/job_runner_multimodel_test.go"
    "internal/api/executions_handler_multimodel_test.go"
    "e2e/multimodel_e2e_test.go"
    "pkg/models/multimodel_signature_test.go"
)

for test_file in "${test_files[@]}"; do
    if [ -f "$test_file" ]; then
        echo "✅ $test_file"
    else
        echo "❌ $test_file (missing)"
        exit 1
    fi
done

echo
echo "🔨 Checking compilation..."

# Check if tests compile
echo "Compiling processor tests..."
if go test -c ./internal/api/processors -o /tmp/processor_test >/dev/null 2>&1; then
    echo "✅ Processor tests compile"
    rm -f /tmp/processor_test
else
    echo "❌ Processor tests compilation failed"
    go test -c ./internal/api/processors 2>&1 | head -10
    exit 1
fi

echo "Compiling worker tests..."
if go test -c ./internal/worker -o /tmp/worker_test >/dev/null 2>&1; then
    echo "✅ Worker tests compile"
    rm -f /tmp/worker_test
else
    echo "❌ Worker tests compilation failed"
    go test -c ./internal/worker 2>&1 | head -10
    exit 1
fi

echo "Compiling API tests..."
if go test -c ./internal/api -o /tmp/api_test >/dev/null 2>&1; then
    echo "✅ API tests compile"
    rm -f /tmp/api_test
else
    echo "❌ API tests compilation failed"
    go test -c ./internal/api 2>&1 | head -10
    exit 1
fi

echo "Compiling models tests..."
if go test -c ./pkg/models -o /tmp/models_test >/dev/null 2>&1; then
    echo "✅ Models tests compile"
    rm -f /tmp/models_test
else
    echo "❌ Models tests compilation failed"
    go test -c ./pkg/models 2>&1 | head -10
    exit 1
fi

echo "Compiling E2E tests..."
if go test -c ./e2e -o /tmp/e2e_test >/dev/null 2>&1; then
    echo "✅ E2E tests compile"
    rm -f /tmp/e2e_test
else
    echo "❌ E2E tests compilation failed"
    go test -c ./e2e 2>&1 | head -10
    exit 1
fi

echo
echo "🧪 Running basic test validation..."

# Run a quick test to ensure basic functionality
echo "Testing JobSpec processor normalization..."
if go test -timeout 10s -run TestNormalizeModelsFromMetadata ./internal/api/processors >/dev/null 2>&1; then
    echo "✅ JobSpec processor normalization tests pass"
else
    echo "❌ JobSpec processor normalization tests failed"
    go test -timeout 10s -run TestNormalizeModelsFromMetadata ./internal/api/processors
    exit 1
fi

echo "Testing multi-model job execution..."
if go test -timeout 10s -run TestExecuteMultiModelJob ./internal/worker >/dev/null 2>&1; then
    echo "✅ Multi-model job execution tests pass"
else
    echo "❌ Multi-model job execution tests failed"
    go test -timeout 10s -run TestExecuteMultiModelJob ./internal/worker
    exit 1
fi

echo "Testing signature verification..."
if go test -timeout 10s -run TestMultiModelJobSignatureVerification ./pkg/models >/dev/null 2>&1; then
    echo "✅ Signature verification tests pass"
else
    echo "❌ Signature verification tests failed"
    go test -timeout 10s -run TestMultiModelJobSignatureVerification ./pkg/models
    exit 1
fi

echo
echo "📊 Test Coverage Check..."

# Generate coverage report for multi-model components
if command -v go >/dev/null 2>&1; then
    echo "Generating coverage report..."
    go test -coverprofile=coverage.out ./internal/api/processors ./internal/worker ./pkg/models >/dev/null 2>&1 || true
    
    if [ -f coverage.out ]; then
        coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
        echo "📈 Test coverage: $coverage"
        
        # Clean up
        rm -f coverage.out
        
        # Check if coverage is reasonable (>50%)
        coverage_num=$(echo $coverage | sed 's/%//')
        if (( $(echo "$coverage_num > 50" | bc -l) )); then
            echo "✅ Coverage is adequate (>50%)"
        else
            echo "⚠️  Coverage is low (<50%) - consider adding more tests"
        fi
    else
        echo "⚠️  Could not generate coverage report"
    fi
fi

echo
echo "🎯 Validation Summary:"
echo "✅ All test files present"
echo "✅ All tests compile successfully"
echo "✅ Basic functionality tests pass"
echo "✅ Multi-model test suite is ready"

echo
echo "🚀 Next Steps:"
echo "1. Run full test suite: ./scripts/test-multimodel.sh"
echo "2. Run specific tests: ./scripts/test-multimodel.sh unit"
echo "3. Run E2E tests: ./scripts/test-multimodel.sh e2e (requires running services)"
echo "4. Generate report: ./scripts/test-multimodel.sh report"

echo
echo "✨ Multi-model test validation completed successfully!"
