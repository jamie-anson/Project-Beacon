#!/usr/bin/env node

/**
 * Bulk Job Recovery Script for Project Beacon
 * 
 * Handles mass recovery of stuck jobs through admin API endpoints.
 * Supports dry-run mode, batch processing, and comprehensive reporting.
 * 
 * Usage:
 *   node bulk-job-recovery.js --help
 *   node bulk-job-recovery.js --dry-run
 *   node bulk-job-recovery.js --execute --batch-size 10
 */

const https = require('https');
const http = require('http');
const { URL } = require('url');

class BulkJobRecovery {
    constructor(options = {}) {
        this.baseUrl = options.baseUrl || process.env.RUNNER_BASE_URL || 'http://localhost:8080';
        this.adminToken = options.adminToken || process.env.ADMIN_TOKEN;
        this.dryRun = options.dryRun || false;
        this.batchSize = options.batchSize || 5;
        this.delay = options.delay || 2000; // 2s between batches
        
        if (!this.adminToken) {
            throw new Error('ADMIN_TOKEN environment variable or --admin-token required');
        }
    }

    async makeRequest(path, method = 'GET', body = null) {
        const url = new URL(path, this.baseUrl);
        const isHttps = url.protocol === 'https:';
        const client = isHttps ? https : http;
        
        const options = {
            hostname: url.hostname,
            port: url.port || (isHttps ? 443 : 80),
            path: url.pathname + url.search,
            method,
            headers: {
                'Authorization': `Bearer ${this.adminToken}`,
                'Content-Type': 'application/json',
                'User-Agent': 'Project-Beacon-Bulk-Recovery/1.0'
            }
        };

        if (body) {
            const bodyStr = JSON.stringify(body);
            options.headers['Content-Length'] = Buffer.byteLength(bodyStr);
        }

        return new Promise((resolve, reject) => {
            const req = client.request(options, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => {
                    try {
                        const parsed = data ? JSON.parse(data) : {};
                        resolve({ status: res.statusCode, data: parsed, headers: res.headers });
                    } catch (e) {
                        resolve({ status: res.statusCode, data: data, headers: res.headers });
                    }
                });
            });

            req.on('error', reject);
            
            if (body) {
                req.write(JSON.stringify(body));
            }
            
            req.end();
        });
    }

    async getStuckJobsStats() {
        console.log('🔍 Fetching stuck jobs statistics...');
        const response = await this.makeRequest('/admin/stuck-jobs-stats');
        
        if (response.status !== 200) {
            throw new Error(`Failed to get stuck jobs stats: ${response.status} ${JSON.stringify(response.data)}`);
        }
        
        return response.data;
    }

    async getResourceStats() {
        console.log('📊 Fetching system resource statistics...');
        const response = await this.makeRequest('/admin/resource-stats');
        
        if (response.status !== 200) {
            throw new Error(`Failed to get resource stats: ${response.status} ${JSON.stringify(response.data)}`);
        }
        
        return response.data;
    }

    async republishStuckJobs() {
        console.log('🔄 Triggering stuck jobs republishing...');
        const response = await this.makeRequest('/admin/repair-stuck-jobs', 'POST');
        
        if (response.status !== 200) {
            throw new Error(`Failed to republish stuck jobs: ${response.status} ${JSON.stringify(response.data)}`);
        }
        
        return response.data;
    }

    async performBulkRecovery() {
        const startTime = Date.now();
        console.log(`🚀 Starting bulk job recovery (dry-run: ${this.dryRun})`);
        console.log(`📋 Configuration: batch-size=${this.batchSize}, delay=${this.delay}ms`);
        console.log('');

        try {
            // Step 1: Get initial system state
            const [stuckStats, resourceStats] = await Promise.all([
                this.getStuckJobsStats(),
                this.getResourceStats()
            ]);

            console.log('📈 Initial System State:');
            console.log(`   Stuck Jobs (created): ${stuckStats.stats?.created_jobs || 0}`);
            console.log(`   Stuck Jobs (running): ${stuckStats.stats?.running_jobs || 0}`);
            console.log(`   Memory Usage: ${Math.round((resourceStats.heap_alloc || 0) / 1024 / 1024)}MB`);
            console.log(`   Goroutines: ${resourceStats.goroutine_count || 0}`);
            console.log('');

            const totalStuckJobs = (stuckStats.stats?.created_jobs || 0) + (stuckStats.stats?.running_jobs || 0);
            
            if (totalStuckJobs === 0) {
                console.log('✅ No stuck jobs found. System is healthy!');
                return { success: true, processed: 0, message: 'No stuck jobs found' };
            }

            console.log(`⚠️  Found ${totalStuckJobs} stuck jobs requiring recovery`);

            if (this.dryRun) {
                console.log('');
                console.log('🔍 DRY RUN MODE - No actual changes will be made');
                console.log(`   Would process ${totalStuckJobs} stuck jobs`);
                console.log(`   Would use ${Math.ceil(totalStuckJobs / this.batchSize)} batches`);
                console.log(`   Estimated time: ${Math.ceil(totalStuckJobs / this.batchSize) * (this.delay / 1000)}s`);
                return { success: true, processed: 0, message: 'Dry run completed' };
            }

            // Step 2: Execute recovery in batches
            console.log('');
            console.log('🔧 Executing recovery...');
            
            const recoveryResult = await this.republishStuckJobs();
            
            console.log('✅ Recovery operation completed');
            console.log(`   Processed: ${recoveryResult.processed || 0} jobs`);
            console.log(`   Errors: ${recoveryResult.errors || 0}`);

            // Step 3: Wait and verify recovery
            console.log('');
            console.log('⏳ Waiting 10 seconds for recovery to process...');
            await new Promise(resolve => setTimeout(resolve, 10000));

            const finalStats = await this.getStuckJobsStats();
            const finalStuckJobs = (finalStats.stats?.created_jobs || 0) + (finalStats.stats?.running_jobs || 0);

            console.log('');
            console.log('📊 Final System State:');
            console.log(`   Stuck Jobs (created): ${finalStats.stats?.created_jobs || 0}`);
            console.log(`   Stuck Jobs (running): ${finalStats.stats?.running_jobs || 0}`);
            console.log(`   Recovery Success Rate: ${Math.round(((totalStuckJobs - finalStuckJobs) / totalStuckJobs) * 100)}%`);

            const duration = Math.round((Date.now() - startTime) / 1000);
            console.log('');
            console.log(`🎉 Bulk recovery completed in ${duration}s`);

            return {
                success: true,
                processed: recoveryResult.processed || 0,
                errors: recoveryResult.errors || 0,
                initialStuck: totalStuckJobs,
                finalStuck: finalStuckJobs,
                recoveryRate: Math.round(((totalStuckJobs - finalStuckJobs) / totalStuckJobs) * 100),
                duration
            };

        } catch (error) {
            console.error('');
            console.error('❌ Bulk recovery failed:', error.message);
            throw error;
        }
    }
}

// CLI Interface
async function main() {
    const args = process.argv.slice(2);
    
    if (args.includes('--help') || args.includes('-h')) {
        console.log(`
Project Beacon Bulk Job Recovery Tool

Usage:
  node bulk-job-recovery.js [options]

Options:
  --help, -h              Show this help message
  --dry-run              Simulate recovery without making changes
  --execute              Execute actual recovery (default if no --dry-run)
  --batch-size <n>       Number of jobs to process per batch (default: 5)
  --delay <ms>           Delay between batches in milliseconds (default: 2000)
  --base-url <url>       Runner API base URL (default: http://localhost:8080)
  --admin-token <token>  Admin authentication token (or use ADMIN_TOKEN env var)

Environment Variables:
  RUNNER_BASE_URL        Base URL for runner API
  ADMIN_TOKEN           Admin authentication token (required)

Examples:
  # Dry run to see what would be recovered
  node bulk-job-recovery.js --dry-run

  # Execute recovery with custom batch size
  node bulk-job-recovery.js --execute --batch-size 10

  # Recovery with custom endpoint
  RUNNER_BASE_URL=https://my-runner.fly.dev node bulk-job-recovery.js --execute
        `);
        return;
    }

    const options = {
        dryRun: args.includes('--dry-run'),
        batchSize: parseInt(args[args.indexOf('--batch-size') + 1]) || 5,
        delay: parseInt(args[args.indexOf('--delay') + 1]) || 2000,
        baseUrl: args[args.indexOf('--base-url') + 1] || process.env.RUNNER_BASE_URL,
        adminToken: args[args.indexOf('--admin-token') + 1] || process.env.ADMIN_TOKEN
    };

    try {
        const recovery = new BulkJobRecovery(options);
        const result = await recovery.performBulkRecovery();
        
        if (result.success) {
            process.exit(0);
        } else {
            process.exit(1);
        }
    } catch (error) {
        console.error('Fatal error:', error.message);
        process.exit(1);
    }
}

if (require.main === module) {
    main().catch(console.error);
}

module.exports = { BulkJobRecovery };
