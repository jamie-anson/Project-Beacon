#!/usr/bin/env node

/**
 * Project Beacon - Performance Testing Script
 * 
 * Comprehensive performance testing for the runner app including
 * load testing, stress testing, and resource monitoring.
 */

const https = require('https');
const http = require('http');
const { Worker, isMainThread, parentPort, workerData } = require('worker_threads');

class PerformanceTester {
    constructor(config = {}) {
        this.config = {
            baseUrl: config.baseUrl || process.env.RUNNER_BASE_URL || 'http://localhost:8080',
            adminToken: config.adminToken || process.env.ADMIN_TOKEN,
            duration: config.duration || 60000, // 1 minute
            concurrency: config.concurrency || 10,
            rampUp: config.rampUp || 10000, // 10 seconds
            ...config
        };
        
        this.metrics = {
            requests: 0,
            responses: 0,
            errors: 0,
            latencies: [],
            throughput: [],
            resourceUsage: [],
            startTime: null,
            endTime: null
        };
    }

    async runPerformanceTest() {
        console.log('🚀 Starting Performance Test');
        console.log(`   Target: ${this.config.baseUrl}`);
        console.log(`   Duration: ${this.config.duration}ms`);
        console.log(`   Concurrency: ${this.config.concurrency}`);
        console.log(`   Ramp-up: ${this.config.rampUp}ms\n`);

        this.metrics.startTime = Date.now();

        try {
            // Start resource monitoring
            const resourceMonitor = this.startResourceMonitoring();
            
            // Run load test
            await this.runLoadTest();
            
            // Stop resource monitoring
            clearInterval(resourceMonitor);
            
            this.metrics.endTime = Date.now();
            this.generateReport();
            
            return this.calculateScore();
        } catch (error) {
            console.error('❌ Performance test failed:', error.message);
            throw error;
        }
    }

    async runLoadTest() {
        console.log('📈 Running Load Test...');
        
        const workers = [];
        const workerPromises = [];
        
        // Create worker threads for concurrent load
        for (let i = 0; i < this.config.concurrency; i++) {
            const worker = new Worker(__filename, {
                workerData: {
                    workerId: i,
                    baseUrl: this.config.baseUrl,
                    adminToken: this.config.adminToken,
                    duration: this.config.duration,
                    rampUpDelay: (this.config.rampUp / this.config.concurrency) * i
                }
            });
            
            workers.push(worker);
            
            const promise = new Promise((resolve, reject) => {
                worker.on('message', (data) => {
                    if (data.type === 'metrics') {
                        this.aggregateWorkerMetrics(data.metrics);
                    } else if (data.type === 'complete') {
                        resolve(data.metrics);
                    }
                });
                
                worker.on('error', reject);
            });
            
            workerPromises.push(promise);
        }
        
        // Wait for all workers to complete
        const workerResults = await Promise.all(workerPromises);
        
        // Cleanup workers
        workers.forEach(worker => worker.terminate());
        
        console.log('✅ Load test completed\n');
        return workerResults;
    }

    aggregateWorkerMetrics(workerMetrics) {
        this.metrics.requests += workerMetrics.requests;
        this.metrics.responses += workerMetrics.responses;
        this.metrics.errors += workerMetrics.errors;
        this.metrics.latencies.push(...workerMetrics.latencies);
    }

    startResourceMonitoring() {
        if (!this.config.adminToken) {
            console.log('⚠️  Resource monitoring disabled (no admin token)');
            return null;
        }
        
        console.log('📊 Starting resource monitoring...');
        
        return setInterval(async () => {
            try {
                const result = await this.makeRequest('/admin/resource-stats', { requiresAuth: true });
                this.metrics.resourceUsage.push({
                    timestamp: Date.now(),
                    memory: result.data.memory,
                    goroutines: result.data.goroutines,
                    gc_pause: result.data.gc_pause_ns
                });
            } catch (error) {
                // Ignore monitoring errors during load test
            }
        }, 5000); // Every 5 seconds
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
                timeout: 30000,
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
                reject(new Error('Request timeout'));
            });
            
            req.end();
        });
    }

    generateReport() {
        const duration = this.metrics.endTime - this.metrics.startTime;
        const rps = Math.round((this.metrics.responses / duration) * 1000);
        const errorRate = ((this.metrics.errors / this.metrics.requests) * 100).toFixed(2);
        
        // Calculate latency percentiles
        this.metrics.latencies.sort((a, b) => a - b);
        const p50 = this.metrics.latencies[Math.floor(this.metrics.latencies.length * 0.5)] || 0;
        const p95 = this.metrics.latencies[Math.floor(this.metrics.latencies.length * 0.95)] || 0;
        const p99 = this.metrics.latencies[Math.floor(this.metrics.latencies.length * 0.99)] || 0;
        const avg = this.metrics.latencies.length > 0 ? 
            Math.round(this.metrics.latencies.reduce((a, b) => a + b) / this.metrics.latencies.length) : 0;
        
        console.log('📊 Performance Test Report');
        console.log('==========================');
        console.log(`Duration: ${duration}ms`);
        console.log(`Total Requests: ${this.metrics.requests}`);
        console.log(`Successful Responses: ${this.metrics.responses}`);
        console.log(`Errors: ${this.metrics.errors} (${errorRate}%)`);
        console.log(`Requests/sec: ${rps}`);
        console.log();
        
        console.log('⏱️  Latency Distribution:');
        console.log(`   Average: ${avg}ms`);
        console.log(`   50th percentile: ${p50}ms`);
        console.log(`   95th percentile: ${p95}ms`);
        console.log(`   99th percentile: ${p99}ms`);
        console.log();
        
        if (this.metrics.resourceUsage.length > 0) {
            const avgMemory = Math.round(
                this.metrics.resourceUsage.reduce((sum, r) => sum + r.memory.heap_alloc, 0) / 
                this.metrics.resourceUsage.length / 1024 / 1024
            );
            const maxGoroutines = Math.max(...this.metrics.resourceUsage.map(r => r.goroutines));
            
            console.log('💾 Resource Usage:');
            console.log(`   Average Memory: ${avgMemory}MB`);
            console.log(`   Peak Goroutines: ${maxGoroutines}`);
            console.log();
        }
        
        const score = this.calculateScore();
        const scoreIcon = score >= 90 ? '🟢' : score >= 70 ? '🟡' : '🔴';
        
        console.log(`${scoreIcon} Performance Score: ${score}/100`);
        
        if (score < 90) {
            console.log('\n💡 Recommendations:');
            if (errorRate > 1) {
                console.log('   • High error rate - investigate error handling');
            }
            if (p95 > 1000) {
                console.log('   • High latency - optimize response times');
            }
            if (rps < 50) {
                console.log('   • Low throughput - consider scaling or optimization');
            }
        }
    }

    calculateScore() {
        const errorRate = (this.metrics.errors / this.metrics.requests) * 100;
        const p95 = this.metrics.latencies[Math.floor(this.metrics.latencies.length * 0.95)] || 0;
        const rps = (this.metrics.responses / (this.metrics.endTime - this.metrics.startTime)) * 1000;
        
        let score = 100;
        
        // Deduct for errors
        score -= Math.min(errorRate * 10, 50);
        
        // Deduct for high latency
        if (p95 > 500) score -= 20;
        if (p95 > 1000) score -= 30;
        
        // Deduct for low throughput
        if (rps < 10) score -= 30;
        if (rps < 50) score -= 20;
        
        return Math.max(0, Math.round(score));
    }
}

// Worker thread implementation
if (!isMainThread) {
    const { workerId, baseUrl, adminToken, duration, rampUpDelay } = workerData;
    
    const workerMetrics = {
        requests: 0,
        responses: 0,
        errors: 0,
        latencies: []
    };
    
    async function runWorkerLoad() {
        // Wait for ramp-up delay
        await new Promise(resolve => setTimeout(resolve, rampUpDelay));
        
        const startTime = Date.now();
        const endTime = startTime + duration;
        
        while (Date.now() < endTime) {
            try {
                const requestStart = Date.now();
                workerMetrics.requests++;
                
                const result = await makeWorkerRequest('/health');
                
                workerMetrics.responses++;
                workerMetrics.latencies.push(Date.now() - requestStart);
                
                // Send periodic updates
                if (workerMetrics.requests % 10 === 0) {
                    parentPort.postMessage({
                        type: 'metrics',
                        metrics: { ...workerMetrics }
                    });
                }
                
                // Small delay to prevent overwhelming
                await new Promise(resolve => setTimeout(resolve, 100));
            } catch (error) {
                workerMetrics.errors++;
            }
        }
        
        parentPort.postMessage({
            type: 'complete',
            metrics: workerMetrics
        });
    }
    
    function makeWorkerRequest(endpoint) {
        return new Promise((resolve, reject) => {
            const url = new URL(endpoint, baseUrl);
            const isHttps = url.protocol === 'https:';
            const client = isHttps ? https : http;
            
            const req = client.request({
                hostname: url.hostname,
                port: url.port || (isHttps ? 443 : 80),
                path: url.pathname,
                method: 'GET',
                timeout: 10000
            }, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => resolve({ statusCode: res.statusCode, data }));
            });
            
            req.on('error', reject);
            req.on('timeout', () => {
                req.destroy();
                reject(new Error('Timeout'));
            });
            
            req.end();
        });
    }
    
    runWorkerLoad().catch(error => {
        parentPort.postMessage({
            type: 'error',
            error: error.message
        });
    });
}

// CLI Usage
if (require.main === module && isMainThread) {
    const config = {
        baseUrl: process.argv[2] || process.env.RUNNER_BASE_URL,
        adminToken: process.env.ADMIN_TOKEN,
        duration: parseInt(process.argv[3]) || 60000,
        concurrency: parseInt(process.argv[4]) || 10
    };
    
    if (!config.baseUrl) {
        console.error('Usage: node performance-test.js <base-url> [duration-ms] [concurrency]');
        console.error('   or set RUNNER_BASE_URL environment variable');
        process.exit(1);
    }
    
    const tester = new PerformanceTester(config);
    tester.runPerformanceTest().catch(error => {
        console.error('Performance test failed:', error);
        process.exit(1);
    });
}

module.exports = PerformanceTester;
