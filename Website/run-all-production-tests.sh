#!/bin/bash

# Complete Production Hardening Test Suite
# Runs all phases of testing for comprehensive validation

echo "🚀 Project Beacon Production Hardening Test Suite"
echo "=================================================="
echo

# Check dependencies
missing_deps=()
if ! command -v curl &> /dev/null; then
    missing_deps+=("curl")
fi
if ! command -v jq &> /dev/null; then
    missing_deps+=("jq")
fi

if [[ ${#missing_deps[@]} -gt 0 ]]; then
    echo "❌ Missing dependencies: ${missing_deps[*]}"
    echo "Please install missing tools and try again"
    exit 1
fi

# Test phases
phases=(
    "tests/integration/quick-smoke-test.sh|Phase 1: Quick Smoke Test|30s"
    "tests/monitoring/health-check-tests.sh|Phase 2: Health Monitoring|60s"
    "tests/integration/cross-service-tests.sh|Phase 3: Cross-Service Integration|300s"
    "tests/regression/known-issues-tests.sh|Phase 4: Regression Prevention|120s"
)

# Results tracking
passed_phases=()
failed_phases=()
warning_phases=()
total_start_time=$(date +%s)

echo "📋 Test Plan:"
for phase in "${phases[@]}"; do
    IFS='|' read -r script name duration <<< "$phase"
    echo "  • $name ($duration max)"
done
echo

# Run each phase
for phase in "${phases[@]}"; do
    IFS='|' read -r script name duration <<< "$phase"
    
    echo "🔄 Running: $name"
    echo "----------------------------------------"
    
    phase_start_time=$(date +%s)
    
    if ./"$script"; then
        phase_end_time=$(date +%s)
        phase_duration=$((phase_end_time - phase_start_time))
        passed_phases+=("$name (${phase_duration}s)")
        echo "✅ $name PASSED in ${phase_duration}s"
    else
        phase_end_time=$(date +%s)
        phase_duration=$((phase_end_time - phase_start_time))
        exit_code=$?
        
        if [[ $exit_code -eq 2 ]]; then
            warning_phases+=("$name (${phase_duration}s)")
            echo "⚠️  $name completed with warnings in ${phase_duration}s"
        else
            failed_phases+=("$name (${phase_duration}s)")
            echo "❌ $name FAILED in ${phase_duration}s"
        fi
    fi
    
    echo
done

# Final summary
total_end_time=$(date +%s)
total_duration=$((total_end_time - total_start_time))

echo "📊 Production Hardening Test Results"
echo "===================================="
echo "⏱️  Total execution time: ${total_duration}s"
echo "✅ Passed: ${#passed_phases[@]} phases"
echo "⚠️  Warnings: ${#warning_phases[@]} phases"
echo "❌ Failed: ${#failed_phases[@]} phases"
echo

if [[ ${#passed_phases[@]} -gt 0 ]]; then
    echo "Passed phases:"
    for phase in "${passed_phases[@]}"; do
        echo "  ✅ $phase"
    done
    echo
fi

if [[ ${#warning_phases[@]} -gt 0 ]]; then
    echo "Warning phases:"
    for phase in "${warning_phases[@]}"; do
        echo "  ⚠️  $phase"
    done
    echo
fi

if [[ ${#failed_phases[@]} -gt 0 ]]; then
    echo "Failed phases:"
    for phase in "${failed_phases[@]}"; do
        echo "  ❌ $phase"
    done
    echo
fi

# Overall assessment
if [[ ${#failed_phases[@]} -eq 0 ]]; then
    if [[ ${#warning_phases[@]} -eq 0 ]]; then
        echo "🎉 ALL TESTS PASSED!"
        echo "✅ Production system is fully hardened and ready"
        echo "🚀 All critical issues have been prevented"
    else
        echo "✅ TESTS PASSED with warnings"
        echo "⚠️  Some non-critical issues detected"
        echo "🔧 Review warnings but system is operational"
    fi
    exit 0
else
    echo "❌ TESTS FAILED"
    echo "🚨 Critical issues detected in production hardening"
    echo "🔧 Fix failed tests before deploying to production"
    exit 1
fi
