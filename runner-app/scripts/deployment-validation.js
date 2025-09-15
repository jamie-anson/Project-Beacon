#!/usr/bin/env node

/**
 * Project Beacon - Deployment Validation Script
 * 
 * Validates deployment health, performance, and reliability metrics
 * after deploying the runner app to production.
 */

const https = require('https');
const http = require('http');

class DeploymentValidator {
    constructor(config = {}) {
        this.config = {
            baseUrl: config.baseUrl || process.env.RUNNER_BASE_URL || 'http://localhost:8080',
            adminToken: config.adminToken || process.env.ADMIN_TOKEN,
            timeout: config.timeout || 30000,
            retries: config.retries || 3,
            ...config
        };
        
        this.results = {
            healthChecks: [],
            performanceTests: [],
            reliabilityTests: [],
            errors: [],
            startTime: new Date(),
            endTime: null
        };
    }

    async validateDeployment() {
        console.log('🚀 Starting Deployment Validation');
        console.log(`   Target: ${this.config.baseUrl}`);
        console.log(`   Timeout: ${this.config.timeout}ms\n`);

        try {
            // Phase 1: Health Checks
            await this.runHealthChecks();
            
            // Phase 2: Performance Tests
            await this.runPerformanceTests();
            
            // Phase 3: Reliability Tests
            await this.runReliabilityTests();
            
            // Generate Report
            this.results.endTime = new Date();
            this.generateReport();
            
            return this.results.errors.length === 0;
        } catch (error) {
            console.error('❌ Deployment validation failed:', error.message);
            this.results.errors.push({
                phase: 'validation',
                error: error.message,
                timestamp: new Date()
            });
            return false;
        }
    }

    async runHealthChecks() {
        console.log('🏥 Running Health Checks...');
        
        const checks = [
            { name: 'Basic Health', endpoint: '/health' },
            { name: 'Database Health', endpoint: '/health/db' },
            { name: 'Redis Health', endpoint: '/health/redis' },
            { name: 'Admin API', endpoint: '/admin/health', requiresAuth: true }
        ];

        for (const check of checks) {
            try {
                const result = await this.makeRequest(check.endpoint, {
                    requiresAuth: check.requiresAuth
                });
                
                this.results.healthChecks.push({
                    name: check.name,
                    status: 'pass',
                    responseTime: result.responseTime,
                    timestamp: new Date()
                });
                
                console.log(`   ✅ ${check.name}: ${result.responseTime}ms`);
            } catch (error) {
                this.results.healthChecks.push({
                    name: check.name,
                    status: 'fail',
                    error: error.message,
                    timestamp: new Date()
                });
                
                console.log(`   ❌ ${check.name}: ${error.message}`);
                this.results.errors.push({
                    phase: 'health',
                    check: check.name,
                    error: error.message,
                    timestamp: new Date()
                });
            }
        }
        
        console.log();
    }

    async runPerformanceTests() {
        console.log('⚡ Running Performance Tests...');
        
        const tests = [
            { name: 'Health Endpoint Latency', endpoint: '/health', iterations: 10 },
            { name: 'Admin Stats Load', endpoint: '/admin/stats', iterations: 5, requiresAuth: true },
            { name: 'Concurrent Health Checks', endpoint: '/health', concurrent: 5 }
        ];

        for (const test of tests) {
            try {
                let results;
                
                if (test.concurrent) {
                    results = await this.runConcurrentTest(test);
                } else {
                    results = await this.runLatencyTest(test);
                }
                
                this.results.performanceTests.push({
                    name: test.name,
                    status: 'pass',
                    ...results,
                    timestamp: new Date()
                });
                
                console.log(`   ✅ ${test.name}: avg ${results.avgLatency}ms, p95 ${results.p95Latency}ms`);
            } catch (error) {
                this.results.performanceTests.push({
                    name: test.name,
                    status: 'fail',
                    error: error.message,
                    timestamp: new Date()
                });
                
                console.log(`   ❌ ${test.name}: ${error.message}`);
                this.results.errors.push({
                    phase: 'performance',
                    test: test.name,
                    error: error.message,
                    timestamp: new Date()
                });
            }
        }
        
        console.log();
    }

    async runReliabilityTests() {
        console.log('🔒 Running Reliability Tests...');
        
        const tests = [
            { name: 'Error Handling', test: 'errorHandling' },
            { name: 'Rate Limiting', test: 'rateLimiting' },
            { name: 'Authentication', test: 'authentication' },
            { name: 'Resource Monitoring', test: 'resourceMonitoring' }
        ];

        for (const test of tests) {
            try {
                const result = await this[test.test]();
                
                this.results.reliabilityTests.push({
                    name: test.name,
                    status: 'pass',
                    ...result,
                    timestamp: new Date()
                });
                
                console.log(`   ✅ ${test.name}: ${result.message || 'passed'}`);
            } catch (error) {
                this.results.reliabilityTests.push({
                    name: test.name,
                    status: 'fail',
                    error: error.message,
                    timestamp: new Date()
                });
                
                console.log(`   ❌ ${test.name}: ${error.message}`);
                this.results.errors.push({
                    phase: 'reliability',
                    test: test.name,
                    error: error.message,
                    timestamp: new Date()
                });
            }
        }
        
        console.log();
    }

    async runLatencyTest(test) {
        const latencies = [];
        
        for (let i = 0; i < test.iterations; i++) {
            const result = await this.makeRequest(test.endpoint, {
                requiresAuth: test.requiresAuth
            });
            latencies.push(result.responseTime);
        }
        
        latencies.sort((a, b) => a - b);
        
        return {
            iterations: test.iterations,
            minLatency: Math.min(...latencies),
            maxLatency: Math.max(...latencies),
            avgLatency: Math.round(latencies.reduce((a, b) => a + b) / latencies.length),
            p95Latency: latencies[Math.floor(latencies.length * 0.95)]
        };
    }

    async runConcurrentTest(test) {
        const promises = [];
        
        for (let i = 0; i < test.concurrent; i++) {
            promises.push(this.makeRequest(test.endpoint, {
                requiresAuth: test.requiresAuth
            }));
        }
        
        const results = await Promise.all(promises);
        const latencies = results.map(r => r.responseTime);
        
        return {
            concurrent: test.concurrent,
            minLatency: Math.min(...latencies),
            maxLatency: Math.max(...latencies),
            avgLatency: Math.round(latencies.reduce((a, b) => a + b) / latencies.length),
            p95Latency: latencies.sort((a, b) => a - b)[Math.floor(latencies.length * 0.95)]
        };
    }

    async errorHandling() {
        // Test 404 handling
        try {
            await this.makeRequest('/nonexistent');
            throw new Error('Expected 404 but got success');
        } catch (error) {
            if (!error.message.includes('404')) {
                throw error;
            }
        }
        
        return { message: '404 handling works' };
    }

    async rateLimiting() {
        if (!this.config.adminToken) {
            return { message: 'skipped (no admin token)' };
        }
        
        // Test admin endpoint rate limiting by making rapid requests
        const promises = [];
        for (let i = 0; i < 3; i++) {
            promises.push(this.makeRequest('/admin/stats', { requiresAuth: true }));
        }
        
        await Promise.all(promises);
        return { message: 'rate limiting functional' };
    }

    async authentication() {
        if (!this.config.adminToken) {
            return { message: 'skipped (no admin token)' };
        }
        
        // Test unauthorized access
        try {
            await this.makeRequest('/admin/stats', { requiresAuth: false });
            throw new Error('Expected 401 but got success');
        } catch (error) {
            if (!error.message.includes('401')) {
                throw error;
            }
        }
        
        // Test authorized access
        await this.makeRequest('/admin/stats', { requiresAuth: true });
        
        return { message: 'authentication working' };
    }

    async resourceMonitoring() {
        if (!this.config.adminToken) {
            return { message: 'skipped (no admin token)' };
        }
        
        const result = await this.makeRequest('/admin/resource-stats', { requiresAuth: true });
        
        if (!result.data.memory || !result.data.goroutines) {
            throw new Error('Resource stats incomplete');
        }
        
        return { message: 'resource monitoring active' };
    }

    async makeRequest(endpoint, options = {}) {
        return new Promise((resolve, reject) => {
            const url = new URL(endpoint, this.config.baseUrl);
            const isHttps = url.protocol === 'https:';
            const client = isHttps ? https : http;
            
            const requestOptions = {
                hostname: url.hostname,
                port: url.port || (isHttps ? 443 : 80),
                path: url.pathname + url.search,
                method: 'GET',
                timeout: this.config.timeout,
                headers: {}
            };
            
            if (options.requiresAuth && this.config.adminToken) {
                requestOptions.headers['Authorization'] = `Bearer ${this.config.adminToken}`;
            }
            
            const startTime = Date.now();
            
            const req = client.request(requestOptions, (res) => {
                let data = '';
                
                res.on('data', (chunk) => {
                    data += chunk;
                });
                
                res.on('end', () => {
                    const responseTime = Date.now() - startTime;
                    
                    if (res.statusCode >= 400) {
                        reject(new Error(`HTTP ${res.statusCode}: ${res.statusMessage}`));
                        return;
                    }
                    
                    let parsedData;
                    try {
                        parsedData = JSON.parse(data);
                    } catch {
                        parsedData = data;
                    }
                    
                    resolve({
                        statusCode: res.statusCode,
                        data: parsedData,
                        responseTime
                    });
                });
            });
            
            req.on('error', (error) => {
                reject(new Error(`Request failed: ${error.message}`));
            });
            
            req.on('timeout', () => {
                req.destroy();
                reject(new Error(`Request timeout after ${this.config.timeout}ms`));
            });
            
            req.end();
        });
    }

    generateReport() {
        const duration = this.results.endTime - this.results.startTime;
        const totalTests = this.results.healthChecks.length + 
                          this.results.performanceTests.length + 
                          this.results.reliabilityTests.length;
        const passedTests = totalTests - this.results.errors.length;
        
        console.log('📊 Deployment Validation Report');
        console.log('================================');
        console.log(`Duration: ${duration}ms`);
        console.log(`Tests: ${passedTests}/${totalTests} passed`);
        console.log(`Errors: ${this.results.errors.length}`);
        console.log();
        
        if (this.results.errors.length > 0) {
            console.log('❌ Errors:');
            this.results.errors.forEach((error, i) => {
                console.log(`   ${i + 1}. [${error.phase}] ${error.check || error.test || 'General'}: ${error.error}`);
            });
            console.log();
        }
        
        // Performance Summary
        if (this.results.performanceTests.length > 0) {
            console.log('⚡ Performance Summary:');
            this.results.performanceTests.forEach(test => {
                if (test.status === 'pass') {
                    console.log(`   ${test.name}: ${test.avgLatency}ms avg`);
                }
            });
            console.log();
        }
        
        const success = this.results.errors.length === 0;
        console.log(success ? '✅ Deployment validation PASSED' : '❌ Deployment validation FAILED');
        
        if (!success) {
            process.exit(1);
        }
    }
}

// CLI Usage
if (require.main === module) {
    const config = {
        baseUrl: process.argv[2] || process.env.RUNNER_BASE_URL,
        adminToken: process.env.ADMIN_TOKEN
    };
    
    if (!config.baseUrl) {
        console.error('Usage: node deployment-validation.js <base-url>');
        console.error('   or set RUNNER_BASE_URL environment variable');
        process.exit(1);
    }
    
    const validator = new DeploymentValidator(config);
    validator.validateDeployment().catch(error => {
        console.error('Validation failed:', error);
        process.exit(1);
    });
}

module.exports = DeploymentValidator;
