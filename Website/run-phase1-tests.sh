#!/bin/bash

# Quick runner for Phase 1 Pipeline Integration Tests

echo "🚀 Starting Phase 1: Pipeline Integration Tests"
echo "================================================"

# Check dependencies
if ! command -v curl &> /dev/null; then
    echo "❌ curl is required but not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    echo "❌ jq is required but not installed"
    exit 1
fi

# Run the tests
./tests/integration/pipeline-tests.sh

exit_code=$?

if [[ $exit_code -eq 0 ]]; then
    echo
    echo "✅ Phase 1 tests completed successfully!"
    echo "📊 All critical pipeline components are working correctly"
else
    echo
    echo "❌ Phase 1 tests failed (exit code: $exit_code)"
    echo "🔧 Check the output above for specific failures"
fi

exit $exit_code
