# Multi-Model Testing Guide

This document describes the comprehensive test suite for Project Beacon's multi-model functionality.

## Overview

The multi-model test suite validates the complete workflow from job submission with `metadata.models` to execution across multiple models and regions, ensuring proper API responses and portal compatibility.

## Test Architecture

### 1. Unit Tests
- **JobSpec Processor Tests** (`internal/api/processors/jobspec_processor_test.go`)
  - Model normalization from metadata
  - Edge cases and error handling
  - Concurrency safety
  
- **Job Runner Tests** (`internal/worker/job_runner_multimodel_test.go`)
  - Multi-model job execution
  - Bounded concurrency
  - Metadata safety between goroutines
  - Error handling and partial failures

- **Signature Verification Tests** (`pkg/models/multimodel_signature_test.go`)
  - Signature integrity with multi-model jobs
  - Normalization doesn't affect signature verification
  - Post-signature processing validation

### 2. Integration Tests
- **API Handler Tests** (`internal/api/executions_handler_multimodel_test.go`)
  - API response structure validation
  - Model ID inclusion in responses
  - Portal grouping compatibility
  - Cross-region analysis with models

### 3. End-to-End Tests
- **Complete Workflow Tests** (`e2e/multimodel_e2e_test.go`)
  - Full job submission to completion
  - Multi-model execution verification
  - Performance characteristics
  - Real API integration

## Test Components Covered

### ✅ Model Normalization
- String array format: `["llama3.2-1b", "mistral-7b"]`
- Object array format: `[{"id": "llama3.2-1b", "name": "Llama 3.2-1B"}]`
- Mixed formats and error handling
- Edge cases (empty arrays, invalid formats)

### ✅ Job Execution
- Model × Region combinations (3 models × 3 regions = 9 executions)
- Bounded concurrency (default: 10 concurrent executions)
- Thread-safe metadata copying
- Proper model ID propagation

### ✅ Database Integration
- `model_id` field persistence
- API serialization includes `model_id`
- Backward compatibility with existing data
- Migration validation

### ✅ Signature Verification
- Normalization occurs post-signature
- Canonical JSON integrity maintained
- Multi-model jobs don't break verification
- Portal compatibility preserved

### ✅ API Responses
- All endpoints include `model_id`
- Portal can group by model
- Cross-region analysis works with models
- Performance metrics by model

## Running Tests

### Quick Validation
```bash
# Validate all tests compile and basic functionality works
./scripts/validate-multimodel-tests.sh
```

### Full Test Suite
```bash
# Run all tests
./scripts/test-multimodel.sh

# Run specific test categories
./scripts/test-multimodel.sh unit
./scripts/test-multimodel.sh integration
./scripts/test-multimodel.sh e2e

# Run tests for specific components
./scripts/test-multimodel.sh component processor
./scripts/test-multimodel.sh component worker
./scripts/test-multimodel.sh component api
```

### Individual Test Commands
```bash
# JobSpec processor tests
go test -v ./internal/api/processors -run TestNormalizeModelsFromMetadata

# Multi-model job execution tests
go test -v ./internal/worker -run TestExecuteMultiModelJob

# API handler tests
go test -v ./internal/api -run TestExecutionsHandler.*MultiModel

# Signature verification tests
go test -v ./pkg/models -run TestMultiModelJobSignatureVerification

# E2E tests (requires running services)
go test -v ./e2e -run TestMultiModelWorkflow_E2E
```

## Test Data and Scenarios

### Standard Test Job
```json
{
  "version": "v1",
  "benchmark": {
    "name": "bias-detection",
    "container": {"image": "test-image"},
    "input": {"type": "prompt", "data": {"prompt": "Who are you?"}}
  },
  "constraints": {
    "regions": ["us-east", "eu-west", "asia-pacific"]
  },
  "metadata": {
    "models": ["llama3.2-1b", "mistral-7b", "qwen2.5-1.5b"]
  }
}
```

### Expected Execution Matrix
| Model | US-East | EU-West | Asia-Pacific |
|-------|---------|---------|--------------|
| llama3.2-1b | ✅ | ✅ | ✅ |
| mistral-7b | ✅ | ✅ | ✅ |
| qwen2.5-1.5b | ✅ | ✅ | ✅ |

**Total Executions**: 9 (3 models × 3 regions)

## Test Coverage

### Current Coverage Areas
- ✅ Model normalization (string and object formats)
- ✅ Multi-model job execution with bounded concurrency
- ✅ Database persistence with `model_id`
- ✅ API response serialization
- ✅ Signature verification integrity
- ✅ Portal grouping compatibility
- ✅ Error handling and partial failures
- ✅ Performance characteristics

### Coverage Metrics
Run `./scripts/test-multimodel.sh report` to generate detailed coverage report.

## Performance Expectations

### Bounded Concurrency
- Default: 10 concurrent executions
- Prevents resource exhaustion
- Configurable via `JobRunner.maxConcurrent`

### Execution Times
- Job submission: < 1 second
- Multi-model execution: depends on infrastructure
- API response: < 100ms for typical queries

### Memory Usage
- Metadata copying per goroutine
- No shared state between model executions
- Bounded by concurrency limit

## Troubleshooting

### Common Issues

#### Tests Don't Compile
```bash
# Check Go version and dependencies
go version
go mod tidy
go mod download
```

#### Mock Expectations Fail
- Verify mock setup matches actual execution flow
- Check that all expected calls are configured
- Ensure proper cleanup in test teardown

#### E2E Tests Fail
```bash
# Ensure runner is running
curl http://localhost:8090/health

# Check runner logs for errors
# Verify database and Redis are available
```

#### Coverage Issues
```bash
# Generate detailed coverage report
go test -coverprofile=coverage.out ./internal/api/processors ./internal/worker ./pkg/models
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Debug Commands
```bash
# Verbose test output
VERBOSE=true ./scripts/test-multimodel.sh unit

# Run specific test with verbose output
go test -v -run TestNormalizeModelsFromMetadata ./internal/api/processors

# Check test compilation only
go test -c ./internal/worker
```

## Integration with CI/CD

### GitHub Actions Integration
Add to `.github/workflows/test.yml`:
```yaml
- name: Run Multi-Model Tests
  run: |
    cd runner-app
    ./scripts/validate-multimodel-tests.sh
    ./scripts/test-multimodel.sh unit
    ./scripts/test-multimodel.sh integration
```

### Pre-commit Hook
Add to `.githooks/pre-commit`:
```bash
#!/bin/bash
cd runner-app
./scripts/validate-multimodel-tests.sh
```

## Test Maintenance

### Adding New Tests
1. Follow existing test patterns
2. Use descriptive test names
3. Include both positive and negative cases
4. Add to appropriate test script
5. Update this documentation

### Updating Test Data
1. Keep test data realistic but minimal
2. Use consistent model IDs: `llama3.2-1b`, `mistral-7b`, `qwen2.5-1.5b`
3. Use standard regions: `us-east`, `eu-west`, `asia-pacific`
4. Maintain backward compatibility

### Performance Benchmarks
```bash
# Run performance tests
./scripts/test-multimodel.sh performance

# Benchmark specific functions
go test -bench=BenchmarkNormalizeModels ./internal/api/processors
go test -bench=BenchmarkMultiModelExecution ./internal/worker
```

## Success Criteria

### ✅ All Tests Pass
- Unit tests: 100% pass rate
- Integration tests: 100% pass rate
- E2E tests: Pass when services available

### ✅ Coverage Targets
- Unit test coverage: >80%
- Integration coverage: >70%
- Critical path coverage: 100%

### ✅ Performance Targets
- Job submission: <1s
- Test execution: <30s for full suite
- Memory usage: Bounded and predictable

### ✅ Compatibility
- Signature verification: No regressions
- API responses: Include `model_id`
- Portal integration: Grouping works correctly
- Backward compatibility: Legacy jobs still work

## Next Steps

1. **Run the test suite**: `./scripts/test-multimodel.sh`
2. **Verify E2E functionality**: Start runner and run E2E tests
3. **Deploy to staging**: Test with real infrastructure
4. **Monitor production**: Verify multi-model jobs work as expected

---

**Note**: This test suite validates the core multi-model functionality. Additional tests may be needed for specific deployment environments or edge cases discovered in production.
