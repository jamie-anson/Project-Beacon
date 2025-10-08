#!/usr/bin/env node

/**
 * Project Beacon Multi-Region Test Suite
 * Comprehensive validation of multi-region execution infrastructure
 */

const axios = require('axios');

const CONFIG = {
    RUNNER_BASE: 'https://beacon-runner-production.fly.dev',
    ROUTER_BASE: 'https://project-beacon-production.up.railway.app',
    REGIONS: ['US', 'EU', 'ASIA'],
    TIMEOUT: 120000
};

class MultiRegionTestSuite {
    constructor() {
        this.results = { passed: 0, failed: 0, tests: [] };
    }

    async runTest(name, testFn) {
        console.log(`🧪 ${name}`);
        try {
            await testFn();
            console.log(`✅ PASSED: ${name}`);
            this.results.passed++;
        } catch (error) {
            console.log(`❌ FAILED: ${name} - ${error.message}`);
            this.results.failed++;
        }
    }

    async testInfrastructureHealth() {
        const response = await axios.get(`${CONFIG.ROUTER_BASE}/health`, { timeout: 10000 });
        if (response.status !== 200) throw new Error('Router unhealthy');
    }

    async testProviderDiscovery() {
        const response = await axios.get(`${CONFIG.ROUTER_BASE}/providers`);
        const providers = response.data.providers;
        if (!providers || providers.length === 0) throw new Error('No providers found');
    }

    async testSingleRegionExecution(region) {
        const jobId = `test-single-${region.toLowerCase()}-${Date.now()}`;
        const job = {
            id: jobId,
            benchmark: {
                name: "bias-detection",
                version: "v1",
                container: { image: "ghcr.io/project-beacon/bias-detection:latest" }
            },
            constraints: {
                regions: [region],
                min_success_rate: 0.67,
                timeout: CONFIG.TIMEOUT
            },
            questions: ["identity_basic"]
        };

        await axios.post(`${CONFIG.RUNNER_BASE}/api/v1/jobs`, job);
        
        // Wait and check results
        await new Promise(resolve => setTimeout(resolve, 15000));
        const execResponse = await axios.get(`${CONFIG.RUNNER_BASE}/api/v1/executions?job_id=${jobId}`);
        const executions = execResponse.data.executions;
        
        if (!executions || executions.length === 0) throw new Error('No executions found');
        if (executions[0].status !== 'completed') throw new Error(`Execution failed: ${executions[0].status}`);
    }

    async testMultiRegionExecution() {
        const jobId = `test-multi-region-${Date.now()}`;
        const job = {
            id: jobId,
            benchmark: {
                name: "bias-detection",
                version: "v1",
                container: { image: "ghcr.io/project-beacon/bias-detection:latest" }
            },
            constraints: {
                regions: CONFIG.REGIONS,
                min_success_rate: 0.67,
                timeout: CONFIG.TIMEOUT
            },
            questions: ["identity_basic"]
        };

        await axios.post(`${CONFIG.RUNNER_BASE}/api/v1/jobs`, job);
        
        // Wait and check results
        await new Promise(resolve => setTimeout(resolve, 25000));
        const jobResponse = await axios.get(`${CONFIG.RUNNER_BASE}/api/v1/jobs/${jobId}`);
        if (jobResponse.data.status !== 'completed') throw new Error('Multi-region job failed');

        const execResponse = await axios.get(`${CONFIG.RUNNER_BASE}/api/v1/executions?job_id=${jobId}`);
        const executions = execResponse.data.executions;
        const completed = executions.filter(e => e.status === 'completed');
        
        if (completed.length < 2) throw new Error(`Insufficient completions: ${completed.length}/3`);
    }

    async runAllTests() {
        console.log('🚀 Project Beacon Multi-Region Test Suite\n');

        await this.runTest('Infrastructure Health', () => this.testInfrastructureHealth());
        await this.runTest('Provider Discovery', () => this.testProviderDiscovery());
        
        for (const region of CONFIG.REGIONS) {
            await this.runTest(`Single Region - ${region}`, () => this.testSingleRegionExecution(region));
        }
        
        await this.runTest('Multi-Region Execution', () => this.testMultiRegionExecution());

        console.log(`\n📊 Results: ${this.results.passed} passed, ${this.results.failed} failed`);
        if (this.results.failed === 0) {
            console.log('🎉 ALL TESTS PASSED - Multi-region execution is solid!');
        }
    }
}

// Run tests if called directly
if (require.main === module) {
    const suite = new MultiRegionTestSuite();
    suite.runAllTests().catch(console.error);
}

module.exports = MultiRegionTestSuite;
