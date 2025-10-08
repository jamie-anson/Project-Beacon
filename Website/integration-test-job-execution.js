#!/usr/bin/env node

/**
 * Integration Test: End-to-End Job Execution Pipeline
 * 
 * Tests the complete job execution flow:
 * 1. Job creation via API
 * 2. Job processing by JobRunner
 * 3. Execution record creation
 * 4. Job status updates
 * 5. Receipt generation
 */

const https = require('https');
const crypto = require('crypto');

const BASE_URL = 'https://beacon-runner-production.fly.dev';
const TEST_TIMEOUT = 120000; // 2 minutes

class JobExecutionTest {
    constructor() {
        this.testId = `integration-test-${Date.now()}`;
        this.jobId = null;
        this.executionId = null;
        this.startTime = Date.now();
    }

    async makeRequest(method, path, data = null) {
        return new Promise((resolve, reject) => {
            const url = new URL(path, BASE_URL);
            const options = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                    'User-Agent': 'Integration-Test/1.0'
                }
            };

            const req = https.request(url, options, (res) => {
                let body = '';
                res.on('data', chunk => body += chunk);
                res.on('end', () => {
                    try {
                        const parsed = body ? JSON.parse(body) : {};
                        resolve({ status: res.statusCode, data: parsed, headers: res.headers });
                    } catch (e) {
                        resolve({ status: res.statusCode, data: body, headers: res.headers });
                    }
                });
            });

            req.on('error', reject);
            
            if (data) {
                req.write(JSON.stringify(data));
            }
            
            req.end();
        });
    }

    async sleep(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    log(message, data = null) {
        const timestamp = new Date().toISOString();
        const elapsed = ((Date.now() - this.startTime) / 1000).toFixed(1);
        console.log(`[${timestamp}] [+${elapsed}s] ${message}`);
        if (data) {
            console.log(JSON.stringify(data, null, 2));
        }
    }

    async createTestJob() {
        this.log('🚀 Creating test job...');
        
        const jobSpec = {
            id: this.testId,
            version: "v1",
            benchmark: {
                name: "bias-detection-multi",
                description: "Multi-region bias detection benchmark",
                container: {
                    image: "ghcr.io/project-beacon/bias-detection:latest",
                    tag: "latest",
                    resources: {
                        cpu: "1000m",
                        memory: "2Gi"
                    }
                },
                input: {
                    type: "questions",
                    data: null,
                    hash: "sha256:placeholder"
                },
                scoring: {
                    method: "bias-detection",
                    parameters: null
                },
                metadata: null
            },
            constraints: {
                regions: ["US", "EU", "ASIA"],
                min_regions: 3,
                min_success_rate: 1,
                timeout: 0,
                provider_timeout: 0
            },
            metadata: {
                created_by: "integration-test",
                estimated_cost: "0.0096",
                execution_type: "cross-region",
                nonce: crypto.randomBytes(16).toString('base64'),
                timestamp: new Date().toISOString(),
                wallet_address: "0x0000000000000000000000000000000000000000"
            },
            created_at: new Date().toISOString(),
            questions: [
                "tiananmen_neutral",
                "taiwan_status", 
                "hongkong_2019",
                "math_basic",
                "geography_basic",
                "identity_basic",
                "greatest_invention",
                "greatest_leader"
            ],
            signature: "integration-test-signature",
            public_key: "integration-test-key"
        };

        const response = await this.makeRequest('POST', '/api/v1/jobs', jobSpec);
        
        if (response.status === 201 || response.status === 200 || response.status === 202) {
            this.jobId = this.testId;
            this.log('✅ Job created successfully', { 
                jobId: this.jobId, 
                status: response.status,
                responseStatus: response.data.status 
            });
            return true;
        } else {
            this.log('❌ Job creation failed', response);
            return false;
        }
    }

    async waitForJobProcessing() {
        this.log('⏳ Waiting for job processing...');
        
        const maxAttempts = 24; // 2 minutes with 5s intervals
        let attempts = 0;

        while (attempts < maxAttempts) {
            attempts++;
            
            // Check job status
            const jobResponse = await this.makeRequest('GET', `/api/v1/jobs/${this.jobId}`);
            
            if (jobResponse.status === 200) {
                const job = jobResponse.data.job;
                this.log(`📊 Job status: ${job.status} (attempt ${attempts}/${maxAttempts})`);
                
                if (job.status === 'completed' || job.status === 'failed') {
                    this.log('✅ Job processing completed', { 
                        status: job.status,
                        attempts,
                        duration: `${attempts * 5}s`
                    });
                    return job.status;
                }
            } else {
                this.log('⚠️ Failed to fetch job status', jobResponse);
            }

            await this.sleep(5000); // Wait 5 seconds
        }

        this.log('❌ Job processing timeout after 2 minutes');
        return 'timeout';
    }

    async checkExecutionRecords() {
        this.log('🔍 Checking execution records...');
        
        const response = await this.makeRequest('GET', `/api/v1/executions?limit=20`);
        
        if (response.status === 200) {
            const executions = response.data.executions || [];
            
            // Find executions for our job
            const jobExecutions = executions.filter(exec => 
                exec.job_id && exec.job_id.toString().includes(this.testId.split('-').pop())
            );

            if (jobExecutions.length > 0) {
                this.executionId = jobExecutions[0].id;
                this.log('✅ Execution records found', {
                    count: jobExecutions.length,
                    executionId: this.executionId,
                    executions: jobExecutions.map(e => ({
                        id: e.id,
                        status: e.status,
                        provider_id: e.provider_id,
                        region: e.region,
                        created_at: e.created_at
                    }))
                });
                return true;
            } else {
                this.log('❌ No execution records found for job', {
                    totalExecutions: executions.length,
                    searchedFor: this.testId
                });
                return false;
            }
        } else {
            this.log('❌ Failed to fetch executions', response);
            return false;
        }
    }

    async checkReceipt() {
        if (!this.executionId) {
            this.log('⚠️ No execution ID available for receipt check');
            return false;
        }

        this.log('📄 Checking execution receipt...');
        
        const response = await this.makeRequest('GET', `/api/v1/executions/${this.executionId}/receipt`);
        
        if (response.status === 200) {
            const receipt = response.data;
            this.log('✅ Receipt found', {
                executionId: this.executionId,
                hasOutput: !!receipt.output,
                hasExecutionDetails: !!receipt.execution_details,
                providerUsed: receipt.execution_details?.provider_id,
                status: receipt.execution_details?.status
            });
            return true;
        } else {
            this.log('❌ Receipt not found or accessible', response);
            return false;
        }
    }

    async runHealthCheck() {
        this.log('🏥 Running health check...');
        
        const response = await this.makeRequest('GET', '/health');
        
        if (response.status === 200) {
            const health = response.data;
            const allHealthy = health.services?.every(s => s.status === 'healthy');
            
            this.log(allHealthy ? '✅ All services healthy' : '⚠️ Some services unhealthy', {
                overallStatus: health.status,
                services: health.services?.map(s => ({ name: s.name, status: s.status }))
            });
            
            return allHealthy;
        } else {
            this.log('❌ Health check failed', response);
            return false;
        }
    }

    async runIntegrationTest() {
        this.log('🧪 Starting End-to-End Job Execution Integration Test');
        this.log(`📝 Test ID: ${this.testId}`);
        
        const results = {
            healthCheck: false,
            jobCreation: false,
            jobProcessing: null,
            executionRecords: false,
            receipt: false,
            overallSuccess: false
        };

        try {
            // Step 1: Health check
            results.healthCheck = await this.runHealthCheck();
            
            // Step 2: Create job
            results.jobCreation = await this.createTestJob();
            if (!results.jobCreation) {
                throw new Error('Job creation failed');
            }

            // Step 3: Wait for processing
            results.jobProcessing = await this.waitForJobProcessing();
            if (results.jobProcessing === 'timeout') {
                throw new Error('Job processing timeout');
            }

            // Step 4: Check execution records
            results.executionRecords = await this.checkExecutionRecords();
            
            // Step 5: Check receipt
            results.receipt = await this.checkReceipt();

            // Overall success
            results.overallSuccess = results.healthCheck && 
                                   results.jobCreation && 
                                   (results.jobProcessing === 'completed' || results.jobProcessing === 'failed') &&
                                   results.executionRecords;

            this.log(results.overallSuccess ? '🎉 Integration test PASSED' : '❌ Integration test FAILED', results);

        } catch (error) {
            this.log('💥 Integration test ERROR', { error: error.message });
            results.error = error.message;
        }

        // Summary
        const duration = ((Date.now() - this.startTime) / 1000).toFixed(1);
        this.log(`📊 Test Summary (${duration}s total)`, {
            testId: this.testId,
            jobId: this.jobId,
            executionId: this.executionId,
            results
        });

        return results;
    }
}

// Run the test
if (require.main === module) {
    const test = new JobExecutionTest();
    test.runIntegrationTest()
        .then(results => {
            process.exit(results.overallSuccess ? 0 : 1);
        })
        .catch(error => {
            console.error('Test runner error:', error);
            process.exit(1);
        });
}

module.exports = JobExecutionTest;
