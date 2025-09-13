# Production Hardening Tests

Based on recent production issues, this document outlines critical tests needed to prevent future failures and ensure system reliability.

## Critical Issues Discovered

### 1. Job Processing Pipeline Failures
- **Issue**: Jobs stuck in "created" status due to missing outbox entries
- **Root Cause**: Payload format mismatch between OutboxPublisher and JobRunner
- **Impact**: 43 jobs required manual republishing

### 2. Data Structure Inconsistencies  
- **Issue**: Runner mock output didn't match actual container output format
- **Root Cause**: Mock generator produced single `text_output` vs structured multi-question responses
- **Impact**: Portal UI couldn't display individual question answers

### 3. API Integration Failures
- **Issue**: Production executions API returning 500 errors
- **Root Cause**: Database connection or handler issues on production
- **Impact**: Receipt viewing functionality broken

### 4. Field Mapping Inconsistencies
- **Issue**: `jobspec_id` vs `id` field mapping confusion
- **Root Cause**: Different services expecting different field names
- **Impact**: Empty ID fields in API responses

## Proposed Testing Strategy

### Phase 1: Pipeline Integration Tests

#### 1.1 Job Lifecycle Tests
```bash
# Test: End-to-end job processing
test_job_lifecycle() {
  # Submit job via portal
  # Verify outbox entry created
  # Verify Redis queue entry
  # Verify worker picks up job
  # Verify execution completes
  # Verify receipt generated
}

# Test: Outbox publisher reliability
test_outbox_publisher() {
  # Create job in database
  # Verify outbox entry generated
  # Verify correct payload format
  # Test retry logic on failures
}

# Test: Redis queue processing
test_redis_queue() {
  # Enqueue job with correct format
  # Verify worker consumes job
  # Test queue failure scenarios
  # Verify dead letter queue handling
}
```

#### 1.2 Data Contract Tests
```javascript
// Test: Runner output format validation
describe('Runner Output Contract', () => {
  it('should match expected multi-question format', () => {
    const output = runnerService.generateOutput(jobspec);
    expect(output).toMatchSchema(multiQuestionSchema);
    expect(output.data.responses).toBeArray();
    expect(output.data.responses[0]).toHaveProperty('question_id');
    expect(output.data.responses[0]).toHaveProperty('question');
    expect(output.data.responses[0]).toHaveProperty('response');
  });
});

// Test: Portal data handling
describe('Portal Receipt Display', () => {
  it('should handle nested data structure', () => {
    const receipt = { output: { data: { data: { responses: [...] } } } };
    const component = render(<ExecutionDetail receipt={receipt} />);
    expect(component.find('.question')).toHaveLength(responses.length);
  });
});
```

### Phase 2: API Health & Monitoring Tests

#### 2.1 Health Check Tests
```bash
# Test: All service endpoints
test_service_health() {
  curl -f "https://beacon-runner-change-me.fly.dev/api/v1/health"
  curl -f "https://project-beacon-production.up.railway.app/health"
  curl -f "https://projectbeacon.netlify.app/api/v1/health"
}

# Test: Database connectivity
test_database_health() {
  # Test runner database connection
  # Test executions API endpoint
  # Verify query performance
}

# Test: Redis connectivity
test_redis_health() {
  # Test Redis connection
  # Test queue operations
  # Verify pub/sub functionality
}
```

#### 2.2 Load & Stress Tests
```bash
# Test: Concurrent job submissions
test_concurrent_jobs() {
  # Submit 10 jobs simultaneously
  # Verify all jobs process correctly
  # Check for race conditions
  # Monitor resource usage
}

# Test: Queue backlog handling
test_queue_backlog() {
  # Submit 100 jobs rapidly
  # Verify queue doesn't overflow
  # Check processing order
  # Monitor memory usage
}
```

### Phase 3: Cross-Service Integration Tests

#### 3.1 Portal → Runner Integration
```javascript
// Test: Job submission flow
describe('Job Submission Integration', () => {
  it('should submit job with correct format', async () => {
    const jobData = createTestJob();
    const response = await portalAPI.submitJob(jobData);
    expect(response.status).toBe(202);
    expect(response.data.id).toBeDefined();
    
    // Verify job appears in runner
    const runnerJob = await runnerAPI.getJob(response.data.id);
    expect(runnerJob.status).toBe('created');
  });
});
```

#### 3.2 Runner → Golem Integration
```bash
# Test: Provider communication
test_golem_integration() {
  # Submit job to runner
  # Verify Golem provider receives job
  # Check execution on Golem network
  # Validate result format
}
```

### Phase 4: Data Validation Tests

#### 4.1 Schema Validation
```javascript
// Test: JobSpec schema validation
const jobSpecSchema = {
  type: 'object',
  required: ['jobspec_id', 'version', 'benchmark', 'constraints', 'questions'],
  properties: {
    jobspec_id: { type: 'string' },
    questions: { type: 'array', items: { type: 'string' } },
    constraints: {
      type: 'object',
      required: ['regions', 'min_regions'],
      properties: {
        min_regions: { type: 'number', minimum: 1 }
      }
    }
  }
};

// Test: Receipt schema validation
const receiptSchema = {
  type: 'object',
  required: ['output'],
  properties: {
    output: {
      type: 'object',
      required: ['data'],
      properties: {
        data: {
          oneOf: [
            // Direct responses format
            { 
              type: 'object',
              required: ['responses'],
              properties: {
                responses: { type: 'array' }
              }
            },
            // Nested responses format
            {
              type: 'object',
              required: ['data'],
              properties: {
                data: {
                  type: 'object',
                  required: ['responses'],
                  properties: {
                    responses: { type: 'array' }
                  }
                }
              }
            }
          ]
        }
      }
    }
  }
};
```

### Phase 5: Regression Tests

#### 5.1 Known Issue Prevention
```bash
# Test: Prevent job stuck in "created" status
test_job_processing_regression() {
  # Submit job
  # Wait 30 seconds
  # Verify job status is not "created"
  # Verify outbox entry exists
}

# Test: Prevent empty ID fields
test_field_mapping_regression() {
  # Submit job with jobspec_id
  # Verify response has populated id field
  # Check both portal and runner APIs
}

# Test: Prevent UI display issues
test_ui_display_regression() {
  # Submit multi-question job
  # Wait for completion
  # Verify all questions display with actual text
  # Verify categories are properly formatted
}
```

## Implementation Plan

### Phase 1: Critical Pipeline Tests (Week 1)
- [ ] Job lifecycle end-to-end test
- [ ] Outbox publisher reliability test
- [ ] Redis queue processing test
- [ ] Data contract validation tests

### Phase 2: API Health Monitoring (Week 2)
- [ ] Automated health check suite
- [ ] Database connectivity tests
- [ ] Load testing framework
- [ ] Performance benchmarking

### Phase 3: Integration Test Suite (Week 3)
- [ ] Portal-Runner integration tests
- [ ] Runner-Golem integration tests
- [ ] Cross-service communication tests
- [ ] Error handling validation

### Phase 4: Continuous Monitoring (Week 4)
- [ ] Automated test execution in CI/CD
- [ ] Production monitoring dashboards
- [ ] Alert system for test failures
- [ ] Regular regression test runs

## Test Infrastructure Requirements

### Tools & Frameworks
- **Backend Testing**: Go testing framework, testify
- **Frontend Testing**: Jest, React Testing Library
- **Integration Testing**: Playwright for E2E
- **Load Testing**: k6 or Artillery
- **API Testing**: Postman/Newman or custom scripts

### Test Environments
- **Staging Environment**: Mirror of production for integration tests
- **Load Testing Environment**: Isolated environment for performance tests
- **Local Testing**: Docker Compose setup for development testing

### Monitoring & Alerting
- **Test Results Dashboard**: Track test success/failure rates
- **Performance Metrics**: Response times, throughput, error rates
- **Alert Integration**: Slack/email notifications for test failures

## Success Metrics

### Reliability Targets
- **Job Success Rate**: >99% of submitted jobs complete successfully
- **API Uptime**: >99.9% availability for all endpoints
- **Processing Time**: <2 minutes from submission to execution start
- **Error Rate**: <0.1% of API requests result in 5xx errors

### Test Coverage Goals
- **Unit Tests**: >80% code coverage
- **Integration Tests**: All critical user journeys covered
- **E2E Tests**: Complete job lifecycle from portal to receipt
- **Performance Tests**: All APIs tested under expected load

---

**Owner**: Jamie  
**Created**: 2025-09-13  
**Status**: Proposed - Ready for implementation
