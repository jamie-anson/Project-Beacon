#!/usr/bin/env node

/**
 * Job Queue Health Monitor for Project Beacon
 * 
 * Real-time monitoring dashboard for job queue health and system metrics.
 * Provides continuous monitoring with alerts and trend analysis.
 * 
 * Usage:
 *   node job-queue-monitor.js --interval 30 --alert-threshold 10
 */

const https = require('https');
const http = require('http');
const { URL } = require('url');

class JobQueueMonitor {
    constructor(options = {}) {
        this.baseUrl = options.baseUrl || process.env.RUNNER_BASE_URL || 'http://localhost:8080';
        this.adminToken = options.adminToken || process.env.ADMIN_TOKEN;
        this.interval = options.interval || 30; // seconds
        this.alertThreshold = options.alertThreshold || 5; // stuck jobs threshold
        this.memoryAlertMB = options.memoryAlertMB || 512; // memory alert threshold
        this.goroutineAlert = options.goroutineAlert || 1000; // goroutine alert threshold
        
        this.history = [];
        this.maxHistory = 20; // Keep last 20 readings
        this.running = false;
        
        if (!this.adminToken) {
            throw new Error('ADMIN_TOKEN environment variable required');
        }
    }

    async makeRequest(path) {
        const url = new URL(path, this.baseUrl);
        const isHttps = url.protocol === 'https:';
        const client = isHttps ? https : http;
        
        const options = {
            hostname: url.hostname,
            port: url.port || (isHttps ? 443 : 80),
            path: url.pathname + url.search,
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${this.adminToken}`,
                'User-Agent': 'Project-Beacon-Monitor/1.0'
            },
            timeout: 10000
        };

        return new Promise((resolve, reject) => {
            const req = client.request(options, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => {
                    try {
                        const parsed = data ? JSON.parse(data) : {};
                        resolve({ status: res.statusCode, data: parsed });
                    } catch (e) {
                        resolve({ status: res.statusCode, data: null, error: e.message });
                    }
                });
            });

            req.on('error', reject);
            req.on('timeout', () => {
                req.destroy();
                reject(new Error('Request timeout'));
            });
            
            req.end();
        });
    }

    async collectMetrics() {
        try {
            const [stuckJobsResp, resourceResp] = await Promise.all([
                this.makeRequest('/admin/stuck-jobs-stats'),
                this.makeRequest('/admin/resource-stats')
            ]);

            const timestamp = new Date();
            const metrics = {
                timestamp,
                stuckJobs: {
                    created: stuckJobsResp.data?.stats?.created_jobs || 0,
                    running: stuckJobsResp.data?.stats?.running_jobs || 0,
                    total: (stuckJobsResp.data?.stats?.created_jobs || 0) + (stuckJobsResp.data?.stats?.running_jobs || 0)
                },
                resources: {
                    heapAllocMB: Math.round((resourceResp.data?.heap_alloc || 0) / 1024 / 1024),
                    heapSysMB: Math.round((resourceResp.data?.heap_sys || 0) / 1024 / 1024),
                    stackInUseMB: Math.round((resourceResp.data?.stack_in_use || 0) / 1024 / 1024),
                    goroutines: resourceResp.data?.goroutine_count || 0,
                    gcPauseMs: Math.round((resourceResp.data?.gc_pause_ns || 0) / 1000000)
                },
                health: {
                    stuckJobsOk: (stuckJobsResp.data?.stats?.created_jobs || 0) + (stuckJobsResp.data?.stats?.running_jobs || 0) < this.alertThreshold,
                    memoryOk: Math.round((resourceResp.data?.heap_alloc || 0) / 1024 / 1024) < this.memoryAlertMB,
                    goroutinesOk: (resourceResp.data?.goroutine_count || 0) < this.goroutineAlert,
                    apiOk: stuckJobsResp.status === 200 && resourceResp.status === 200
                }
            };

            // Add to history
            this.history.push(metrics);
            if (this.history.length > this.maxHistory) {
                this.history.shift();
            }

            return metrics;
        } catch (error) {
            return {
                timestamp: new Date(),
                error: error.message,
                health: { apiOk: false }
            };
        }
    }

    formatMetrics(metrics) {
        const time = metrics.timestamp.toLocaleTimeString();
        
        if (metrics.error) {
            return `[${time}] ❌ ERROR: ${metrics.error}`;
        }

        const { stuckJobs, resources, health } = metrics;
        
        // Health indicators
        const healthIcon = health.apiOk && health.stuckJobsOk && health.memoryOk && health.goroutinesOk ? '✅' : '⚠️';
        
        // Stuck jobs status
        const stuckIcon = health.stuckJobsOk ? '✅' : '🚨';
        const stuckText = `${stuckIcon} Jobs: ${stuckJobs.total} stuck (${stuckJobs.created} created, ${stuckJobs.running} running)`;
        
        // Memory status
        const memIcon = health.memoryOk ? '✅' : '🚨';
        const memText = `${memIcon} Memory: ${resources.heapAllocMB}MB heap, ${resources.stackInUseMB}MB stack`;
        
        // Goroutines status
        const gorIcon = health.goroutinesOk ? '✅' : '🚨';
        const gorText = `${gorIcon} Goroutines: ${resources.goroutines}`;
        
        // GC info
        const gcText = `GC: ${resources.gcPauseMs}ms pause`;

        return `[${time}] ${healthIcon} ${stuckText} | ${memText} | ${gorText} | ${gcText}`;
    }

    generateTrendAnalysis() {
        if (this.history.length < 3) return '';
        
        const recent = this.history.slice(-3);
        const trends = [];
        
        // Stuck jobs trend
        const stuckTrend = recent.map(m => m.stuckJobs?.total || 0);
        if (stuckTrend[2] > stuckTrend[0]) trends.push('📈 Stuck jobs increasing');
        else if (stuckTrend[2] < stuckTrend[0]) trends.push('📉 Stuck jobs decreasing');
        
        // Memory trend
        const memTrend = recent.map(m => m.resources?.heapAllocMB || 0);
        if (memTrend[2] > memTrend[0] + 50) trends.push('📈 Memory usage rising');
        else if (memTrend[2] < memTrend[0] - 50) trends.push('📉 Memory usage falling');
        
        // Goroutine trend
        const gorTrend = recent.map(m => m.resources?.goroutines || 0);
        if (gorTrend[2] > gorTrend[0] + 100) trends.push('📈 Goroutines increasing');
        
        return trends.length > 0 ? `\n   Trends: ${trends.join(', ')}` : '';
    }

    async start() {
        console.log('🚀 Starting Job Queue Monitor');
        console.log(`📊 Monitoring: ${this.baseUrl}`);
        console.log(`⏱️  Interval: ${this.interval}s`);
        console.log(`🚨 Alerts: ${this.alertThreshold} stuck jobs, ${this.memoryAlertMB}MB memory, ${this.goroutineAlert} goroutines`);
        console.log('');
        
        this.running = true;
        
        while (this.running) {
            const metrics = await this.collectMetrics();
            const output = this.formatMetrics(metrics);
            const trends = this.generateTrendAnalysis();
            
            console.log(output + trends);
            
            // Alert on critical issues
            if (metrics.stuckJobs?.total >= this.alertThreshold) {
                console.log(`🚨 ALERT: ${metrics.stuckJobs.total} stuck jobs detected (threshold: ${this.alertThreshold})`);
            }
            
            if (metrics.resources?.heapAllocMB >= this.memoryAlertMB) {
                console.log(`🚨 ALERT: High memory usage ${metrics.resources.heapAllocMB}MB (threshold: ${this.memoryAlertMB}MB)`);
            }
            
            if (metrics.resources?.goroutines >= this.goroutineAlert) {
                console.log(`🚨 ALERT: High goroutine count ${metrics.resources.goroutines} (threshold: ${this.goroutineAlert})`);
            }
            
            await new Promise(resolve => setTimeout(resolve, this.interval * 1000));
        }
    }

    stop() {
        this.running = false;
        console.log('\n🛑 Monitor stopped');
    }
}

// CLI Interface
async function main() {
    const args = process.argv.slice(2);
    
    if (args.includes('--help') || args.includes('-h')) {
        console.log(`
Project Beacon Job Queue Monitor

Usage:
  node job-queue-monitor.js [options]

Options:
  --help, -h                    Show this help message
  --interval <seconds>          Monitoring interval in seconds (default: 30)
  --alert-threshold <jobs>      Alert when stuck jobs exceed this number (default: 5)
  --memory-alert <mb>           Alert when memory usage exceeds this MB (default: 512)
  --goroutine-alert <count>     Alert when goroutines exceed this count (default: 1000)
  --base-url <url>             Runner API base URL (default: http://localhost:8080)

Environment Variables:
  RUNNER_BASE_URL              Base URL for runner API
  ADMIN_TOKEN                  Admin authentication token (required)

Examples:
  # Basic monitoring every 30 seconds
  node job-queue-monitor.js

  # High-frequency monitoring with custom thresholds
  node job-queue-monitor.js --interval 10 --alert-threshold 3 --memory-alert 256

  # Production monitoring
  RUNNER_BASE_URL=https://runner.fly.dev node job-queue-monitor.js --interval 60
        `);
        return;
    }

    const options = {
        interval: parseInt(args[args.indexOf('--interval') + 1]) || 30,
        alertThreshold: parseInt(args[args.indexOf('--alert-threshold') + 1]) || 5,
        memoryAlertMB: parseInt(args[args.indexOf('--memory-alert') + 1]) || 512,
        goroutineAlert: parseInt(args[args.indexOf('--goroutine-alert') + 1]) || 1000,
        baseUrl: args[args.indexOf('--base-url') + 1] || process.env.RUNNER_BASE_URL,
        adminToken: process.env.ADMIN_TOKEN
    };

    try {
        const monitor = new JobQueueMonitor(options);
        
        // Handle graceful shutdown
        process.on('SIGINT', () => {
            monitor.stop();
            process.exit(0);
        });
        
        process.on('SIGTERM', () => {
            monitor.stop();
            process.exit(0);
        });
        
        await monitor.start();
    } catch (error) {
        console.error('Fatal error:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main().catch(console.error);
}

module.exports = { JobQueueMonitor };
