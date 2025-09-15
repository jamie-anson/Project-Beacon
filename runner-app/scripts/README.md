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

## Integration with Production

### Fly.io Deployment:
```bash
# Set admin token in Fly secrets
fly secrets set ADMIN_TOKEN=your-secure-token

# Run recovery from local machine
RUNNER_BASE_URL=https://your-app.fly.dev node bulk-job-recovery.js --execute
```

### Monitoring Setup:
```bash
# Continuous monitoring (run in tmux/screen)
RUNNER_BASE_URL=https://your-app.fly.dev node job-queue-monitor.js --interval 60
```

### Automated Recovery:
Consider setting up cron jobs for regular health checks:
```bash
# Check for stuck jobs every hour
0 * * * * cd /path/to/scripts && node bulk-job-recovery.js --dry-run
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
