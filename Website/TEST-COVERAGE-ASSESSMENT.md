# Test Coverage Assessment & Improvement Plan

## Current Testing Gaps Analysis

### Issues That Reached Production
1. **Double slash URL concatenation bug** (`//questions`, `//jobs`)
2. **Variable scope error** (`url is not defined`)
3. **Vite build optimization removing CORS settings**
4. **Browser vs curl CORS behavior differences**

### Current Test Coverage

#### ✅ What We Test Well
- **API Payload Validation** (`scripts/test-job-payload.js`)
  - Required fields validation
  - Ed25519 cryptographic signing
  - API response validation
  - Trust policy enforcement

- **End-to-End Signing** (`scripts/test-portal-signing.js`)
  - Portal cryptographic workflow
  - Job submission with signatures

#### ✅ Recently Completed

##### 1. **Portal API Client Testing** ✅
- **Implemented**: Jest unit tests for `portal/src/lib/api.js` (12 tests)
- **Coverage**: URL concatenation, CORS settings, error handling, response parsing
- **Status**: All tests passing, prevents URL/scope bugs

##### 2. **Build Process Validation** ✅
- **Implemented**: `scripts/test-build-output.js` validation script
- **Coverage**: CORS settings survival, API base URL embedding, fetch call verification
- **Status**: Catches Vite optimization issues automatically

##### 3. **Browser Environment Testing** ✅
- **Implemented**: Playwright browser integration tests (5 tests)
- **Coverage**: Real browser CORS behavior, preflight requests, portal loading
- **Status**: Automated browser validation working

##### 4. **CI/CD Integration** ✅
- **Implemented**: Complete GitHub Actions workflow suite
- **Coverage**: test.yml, pr-checks.yml, deploy.yml, security-scan.yml
- **Status**: Automated testing on every commit/PR

##### 5. **Pre-deployment Validation** ✅
- **Implemented**: `scripts/pre-deploy-validation.js` comprehensive pipeline
- **Coverage**: All tests + build validation + environment checks
- **Status**: Prevents broken deployments

#### ❌ Remaining Gaps

##### 1. **Cryptographic Key Management**
- **Issue**: Portal generates new keypairs, but API rejects "untrusted signing key"
- **Impact**: Job submissions fail with trust_violation:unknown error
- **Risk**: High - blocks all job execution

##### 2. **API Error Handling Enhancement**
- **Partially Done**: Enhanced error parsing in API client
- **Missing**: User-friendly error messages in UI components
- **Risk**: Medium - poor user experience on errors

## Improvement Plan

### Phase 1: Critical Infrastructure Tests

#### 1.1 Portal API Client Unit Tests
```javascript
// portal/src/lib/__tests__/api.test.js
describe('API Client', () => {
  test('URL concatenation handles paths correctly', () => {
    // Test /path, path, //path scenarios
  });
  
  test('CORS options are preserved in fetch calls', () => {
    // Mock fetch and verify options
  });
  
  test('Error handling includes constructed URL', () => {
    // Verify error messages show actual URLs
  });
});
```

#### 1.2 Build Verification Tests
```javascript
// scripts/test-build-output.js
describe('Build Output Validation', () => {
  test('Built JavaScript contains CORS mode settings', () => {
    // Parse built files and verify CORS code exists
  });
  
  test('API base URL is correctly embedded', () => {
    // Verify environment variables are properly substituted
  });
});
```

### Phase 2: Browser Environment Testing

#### 2.1 Automated Browser CORS Tests
```javascript
// tests/browser/cors.test.js
describe('Browser CORS Behavior', () => {
  test('Preflight requests include proper headers', () => {
    // Use Playwright/Puppeteer to test actual browser requests
  });
  
  test('Portal can make successful API calls', () => {
    // End-to-end browser testing
  });
});
```

#### 2.2 Integration Test Suite
```javascript
// tests/integration/portal-api.test.js
describe('Portal API Integration', () => {
  test('Job submission workflow', () => {
    // Test full job creation flow
  });
  
  test('Cryptographic signing integration', () => {
    // Test portal signing + API validation
  });
});
```

### Phase 3: Deployment & Environment Testing

#### 3.1 Environment Configuration Tests
```javascript
// scripts/test-environment.js
describe('Environment Configuration', () => {
  test('API base URL is accessible', () => {
    // Health check against configured API
  });
  
  test('CORS configuration allows portal origin', () => {
    // Verify CORS headers for portal domain
  });
});
```

#### 3.2 Pre-deployment Validation
```bash
# scripts/pre-deploy-tests.sh
#!/bin/bash
# Run before each deployment
npm run test:unit
npm run test:build-output
npm run test:environment
npm run test:integration
```

## Implementation Priority

### ✅ Completed (All High Priority Items Done!)
1. **Portal API Client Unit Tests** ✅ - Prevents URL/scope bugs
2. **Build Output Validation** ✅ - Catches optimization issues  
3. **Browser CORS Testing** ✅ - Automated browser validation
4. **Pre-deployment Test Suite** ✅ - CI/CD integration
5. **Environment Configuration Tests** ✅ - Deployment safety

### 🚨 Critical Priority (Blocking Job Execution)
1. **Cryptographic Key Trust Management** - Portal keys not in API allowlist
2. **User-Friendly Error Messages** - Better UX for API errors

### Low Priority (Future)
1. **Performance Testing** - API response times
2. **Security Testing** - Signature validation edge cases
3. **Cross-browser Testing** - Multiple browser support

## Test Framework Recommendations

### Frontend Testing
- **Jest** + **@testing-library/react** for React components
- **MSW** (Mock Service Worker) for API mocking
- **Playwright** for browser automation

### Backend/Integration Testing
- **Node.js** native test runner or **Jest**
- **Supertest** for API testing
- **Docker** for environment consistency

### CI/CD Integration
- **GitHub Actions** workflows
- **Netlify** build hooks for deployment validation
- **Fly.io** health checks for API validation

## Success Metrics

### Coverage Targets
- **Unit Tests**: 80%+ coverage for `portal/src/lib/`
- **Integration Tests**: 100% coverage for critical user flows
- **Build Validation**: 100% verification of production assets

### Quality Gates
- All tests pass before deployment
- Build output validation in CI/CD
- Automated CORS verification
- Environment health checks

## Next Steps

1. **Create test infrastructure** - Set up Jest, testing-library
2. **Implement Phase 1 tests** - API client and build validation
3. **Add CI/CD integration** - Automate test execution
4. **Expand to browser testing** - Playwright setup
5. **Monitor and iterate** - Track test effectiveness

This plan addresses the root causes of our production issues and establishes comprehensive testing to prevent similar problems in the future.
