# Project Beacon Admin Scripts

This directory contains administrative tools for managing and monitoring the Project Beacon runner application.

## Scripts Overview

### 🔧 bulk-job-recovery.js
Comprehensive bulk job recovery tool for handling stuck jobs at scale.

**Features:**
- Dry-run mode for safe testing
- Batch processing with configurable delays
- Real-time progress reporting
- Recovery success rate analysis
- System resource monitoring during recovery

**Usage:**
```bash
# Dry run to see what would be recovered
node bulk-job-recovery.js --dry-run

# Execute recovery with custom batch size
ADMIN_TOKEN=your-token node bulk-job-recovery.js --execute --batch-size 10

# Production recovery
RUNNER_BASE_URL=https://runner.fly.dev ADMIN_TOKEN=token node bulk-job-recovery.js --execute
```

### 📊 job-queue-monitor.js
Real-time job queue health monitoring dashboard.

**Features:**
- Continuous monitoring with configurable intervals
- Alert thresholds for stuck jobs, memory usage, and goroutines
- Trend analysis and historical tracking
- Health status indicators
- Graceful shutdown handling

**Usage:**
```bash
# Basic monitoring every 30 seconds
ADMIN_TOKEN=your-token node job-queue-monitor.js

# High-frequency monitoring with custom alerts
node job-queue-monitor.js --interval 10 --alert-threshold 3 --memory-alert 256

# Production monitoring
RUNNER_BASE_URL=https://runner.fly.dev ADMIN_TOKEN=token node job-queue-monitor.js --interval 60
```

## CLI Admin Tool

### 🛠️ cmd/admin/main.go
Native Go CLI tool for direct database operations and job management.

**Build:**
```bash
go build -o admin ./cmd/admin
```

**Commands:**
```bash
# List stuck jobs
./admin stuck-jobs

# Dry-run recovery check
./admin recovery

# Execute recovery for jobs stale >15 minutes
./admin recovery --execute --threshold 15

# Show comprehensive statistics
./admin stats

# Requeue specific job
./admin requeue job-12345
```

## Environment Variables

All tools require these environment variables:

```bash
# Required
ADMIN_TOKEN=your-admin-bearer-token
DATABASE_URL=postgresql://user:pass@host:port/db

# Optional
RUNNER_BASE_URL=http://localhost:8080  # Default for scripts
```

## Security Notes

- **Admin Token**: All tools require a valid admin bearer token
- **Rate Limiting**: Admin endpoints have built-in rate limiting
- **Audit Logging**: All admin operations are logged for security
- **Network Access**: Ensure proper firewall rules for admin endpoints

## Monitoring Alerts

The monitoring tools will alert on:

- **Stuck Jobs**: >5 jobs stuck in created/running status
- **Memory Usage**: >512MB heap allocation
- **Goroutine Leaks**: >1000 active goroutines
- **API Failures**: Admin endpoint unavailability

## Recovery Procedures

### For Stuck Jobs (created/running status):
1. Run `bulk-job-recovery.js --dry-run` to assess scope
2. Execute `bulk-job-recovery.js --execute` for recovery
3. Monitor with `job-queue-monitor.js` to verify recovery

### For Stale Jobs (processing status):
1. Use `admin recovery` to check stale jobs
2. Execute `admin recovery --execute` to reset stale jobs
3. Jobs will be automatically reprocessed by outbox publisher

### For Individual Jobs:
1. Use `admin requeue <job-id>` for specific job recovery
2. Check job status with `admin stuck-jobs`

## Production Integration

### Deployment
1. Copy scripts to production server
2. Set environment variables:
   ```bash
   export RUNNER_BASE_URL="https://your-runner.fly.dev"
   export ADMIN_TOKEN="your-admin-token"
   export DATABASE_URL="your-database-url"
   ```
3. Run initial recovery:
   ```bash
   node bulk-job-recovery.js --dry-run
   node bulk-job-recovery.js --execute
   ```

### Monitoring Setup
1. Add to crontab for regular monitoring:
   ```bash
   # Check job queue health every 5 minutes
   */5 * * * * cd /path/to/runner && node scripts/job-queue-monitor.js
   ```

2. Set up alerting based on script output
3. Configure log aggregation for script outputs

### Maintenance
- Run `bulk-job-recovery.js` during maintenance windows
- Monitor `job-queue-monitor.js` output for trends
- Use admin CLI for quick diagnostics

## Phase 6: Deployment Validation & Performance Testing

### Deployment Validation Script
`deployment-validation.js` - Comprehensive deployment health validation

**Features:**
- Health check validation (basic, database, Redis, admin API)
- Performance testing (latency, concurrent requests)
- Reliability testing (error handling, rate limiting, authentication)
- Resource monitoring validation

**Usage:**
```bash
# Basic validation
node scripts/deployment-validation.js https://your-runner.fly.dev

# With admin token for full validation
ADMIN_TOKEN=your-token node scripts/deployment-validation.js https://your-runner.fly.dev
```

### Performance Testing Script
`performance-test.js` - Load testing and performance benchmarking

**Features:**
- Multi-threaded load testing with worker threads
- Configurable duration, concurrency, and ramp-up
- Real-time resource monitoring during tests
- Comprehensive performance metrics and scoring

**Usage:**
```bash
# Basic performance test (60s, 10 concurrent)
node scripts/performance-test.js https://your-runner.fly.dev

# Custom test (30s, 20 concurrent)
node scripts/performance-test.js https://your-runner.fly.dev 30000 20

# With resource monitoring
ADMIN_TOKEN=your-token node scripts/performance-test.js https://your-runner.fly.dev
```

### Blue-Green Deployment Script
`blue-green-deploy.sh` - Zero-downtime deployment with rollback

**Features:**
- Automated blue-green deployment strategy
- Health check validation and traffic switching
- Automatic rollback on failure
- Support for multiple deployment methods (Fly.io, Docker, etc.)
- Integrated deployment validation and performance testing

**Usage:**
```bash
# Basic blue-green deployment
./scripts/blue-green-deploy.sh

# With custom URLs
./scripts/blue-green-deploy.sh --blue-url http://blue.example.com --green-url http://green.example.com

# Deploy to Fly.io with old environment cleanup
DEPLOYMENT_METHOD=fly ./scripts/blue-green-deploy.sh --stop-old-env
```

**Environment Variables:**
- `BLUE_URL` - Blue environment URL
- `GREEN_URL` - Green environment URL  
- `DEPLOYMENT_METHOD` - fly, docker, or simulate
- `LOAD_BALANCER` - nginx, haproxy, cloudflare, or simulate
- `ADMIN_TOKEN` - Admin API token for validation

### Production Deployment Workflow

1. **Pre-deployment Validation:**
   ```bash
   # Validate current production
   node scripts/deployment-validation.js $PRODUCTION_URL
   ```

2. **Deploy with Blue-Green:**
   ```bash
   # Zero-downtime deployment
   ./scripts/blue-green-deploy.sh
   ```

3. **Post-deployment Testing:**
   ```bash
   # Validate new deployment
   node scripts/deployment-validation.js $NEW_PRODUCTION_URL
   
   # Run performance test
   node scripts/performance-test.js $NEW_PRODUCTION_URL
   ```

4. **Monitoring:**
   ```bash
   # Continuous monitoring
   node scripts/job-queue-monitor.js
   ```

## Troubleshooting

### Common Issues:

1. **Authentication Errors**: Verify ADMIN_TOKEN is set correctly
2. **Connection Timeouts**: Check RUNNER_BASE_URL and network connectivity
3. **Permission Denied**: Ensure admin token has proper permissions
4. **Database Errors**: Verify DATABASE_URL for CLI tools

### Debug Mode:
Add `--verbose` flag to scripts for detailed logging (if implemented).

### Support:
For issues or feature requests, check the main Project Beacon documentation or create an issue in the repository.
