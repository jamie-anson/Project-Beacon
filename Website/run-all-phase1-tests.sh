#!/bin/bash

# Complete Phase 1 Test Suite Runner
# Executes all pipeline integration tests

echo "🚀 Phase 1: Complete Pipeline Integration Test Suite"
echo "===================================================="

# Check dependencies
if ! command -v curl &> /dev/null; then
    echo "❌ curl is required but not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "❌ jq is required but not installed"
    exit 1
fi

# Test execution order
tests=(
    "tests/integration/basic-pipeline-test.sh"
    "tests/integration/advanced-pipeline-test.sh"
)

passed_tests=()
failed_tests=()

echo "📋 Running ${#tests[@]} test suites..."
echo

for test in "${tests[@]}"; do
    test_name=$(basename "$test" .sh)
    echo "🔄 Running: $test_name"
    echo "----------------------------------------"
    
    if ./"$test"; then
        passed_tests+=("$test_name")
        echo "✅ $test_name PASSED"
    else
        failed_tests+=("$test_name")
        echo "❌ $test_name FAILED"
    fi
    
    echo
done

# Final summary
echo "📊 Phase 1 Test Results Summary"
echo "==============================="
echo "✅ Passed: ${#passed_tests[@]} tests"
echo "❌ Failed: ${#failed_tests[@]} tests"

if [[ ${#passed_tests[@]} -gt 0 ]]; then
    echo
    echo "Passed tests:"
    for test in "${passed_tests[@]}"; do
        echo "  ✅ $test"
    done
fi

if [[ ${#failed_tests[@]} -gt 0 ]]; then
    echo
    echo "Failed tests:"
    for test in "${failed_tests[@]}"; do
        echo "  ❌ $test"
    done
    echo
    echo "🔧 Review failed tests and fix issues before proceeding to Phase 2"
    exit 1
else
    echo
    echo "🎉 All Phase 1 tests PASSED!"
    echo "✅ Pipeline is ready for production"
    echo "🚀 Ready to proceed to Phase 2: API Health Monitoring"
    exit 0
fi
