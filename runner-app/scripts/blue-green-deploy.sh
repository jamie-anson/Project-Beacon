#!/bin/bash

# Project Beacon - Blue-Green Deployment Script
# 
# Implements zero-downtime deployment strategy with health checks,
# rollback capabilities, and deployment validation.

set -euo pipefail

# Configuration
BLUE_URL="${BLUE_URL:-http://localhost:8080}"
GREEN_URL="${GREEN_URL:-http://localhost:8081}"
HEALTH_ENDPOINT="/health"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"
VALIDATION_TIMEOUT=300  # 5 minutes
ROLLBACK_TIMEOUT=60     # 1 minute

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if URL is healthy
check_health() {
    local url=$1
    local timeout=${2:-10}
    
    if curl -s -f --max-time "$timeout" "$url$HEALTH_ENDPOINT" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Wait for service to become healthy
wait_for_health() {
    local url=$1
    local timeout=${2:-60}
    local interval=5
    local elapsed=0
    
    log "Waiting for $url to become healthy..."
    
    while [ $elapsed -lt $timeout ]; do
        if check_health "$url" 5; then
            success "Service at $url is healthy"
            return 0
        fi
        
        sleep $interval
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    
    echo
    error "Service at $url failed to become healthy within ${timeout}s"
    return 1
}

# Run deployment validation
validate_deployment() {
    local url=$1
    
    log "Running deployment validation on $url..."
    
    if [ -f "scripts/deployment-validation.js" ]; then
        if node scripts/deployment-validation.js "$url"; then
            success "Deployment validation passed"
            return 0
        else
            error "Deployment validation failed"
            return 1
        fi
    else
        warning "Deployment validation script not found, skipping..."
        return 0
    fi
}

# Run performance test
run_performance_test() {
    local url=$1
    
    log "Running performance test on $url..."
    
    if [ -f "scripts/performance-test.js" ]; then
        if node scripts/performance-test.js "$url" 30000 5; then
            success "Performance test passed"
            return 0
        else
            warning "Performance test failed, but continuing deployment"
            return 0  # Don't fail deployment on performance issues
        fi
    else
        warning "Performance test script not found, skipping..."
        return 0
    fi
}

# Deploy to target environment
deploy_to_target() {
    local target=$1
    local environment=${2:-"production"}
    
    log "Deploying to $target environment..."
    
    # This would typically involve:
    # - Building the application
    # - Pushing to container registry
    # - Updating deployment configuration
    # - Rolling out to target environment
    
    case $environment in
        "fly")
            log "Deploying to Fly.io..."
            if command -v flyctl >/dev/null 2>&1; then
                flyctl deploy --app "beacon-runner-$target"
            else
                error "flyctl not found, cannot deploy to Fly.io"
                return 1
            fi
            ;;
        "docker")
            log "Deploying Docker container..."
            docker build -t "beacon-runner:$target" .
            docker stop "beacon-runner-$target" 2>/dev/null || true
            docker rm "beacon-runner-$target" 2>/dev/null || true
            docker run -d --name "beacon-runner-$target" \
                -p "${target_port}:8080" \
                -e DATABASE_URL="$DATABASE_URL" \
                -e REDIS_URL="$REDIS_URL" \
                -e ADMIN_TOKEN="$ADMIN_TOKEN" \
                "beacon-runner:$target"
            ;;
        *)
            log "Simulating deployment to $target..."
            sleep 2
            ;;
    esac
    
    success "Deployment to $target completed"
}

# Switch traffic from blue to green
switch_traffic() {
    local from=$1
    local to=$2
    
    log "Switching traffic from $from to $to..."
    
    # This would typically involve updating load balancer configuration
    # For this example, we'll simulate the process
    
    case "${LOAD_BALANCER:-simulate}" in
        "nginx")
            log "Updating nginx configuration..."
            # Update nginx upstream configuration
            ;;
        "haproxy")
            log "Updating HAProxy configuration..."
            # Update HAProxy backend configuration
            ;;
        "cloudflare")
            log "Updating Cloudflare DNS..."
            # Update DNS records via Cloudflare API
            ;;
        *)
            log "Simulating traffic switch..."
            sleep 1
            ;;
    esac
    
    success "Traffic switched from $from to $to"
}

# Rollback deployment
rollback() {
    local from=$1
    local to=$2
    
    error "Rolling back deployment from $from to $to..."
    
    # Switch traffic back
    switch_traffic "$from" "$to"
    
    # Wait for rollback to complete
    if wait_for_health "$to" $ROLLBACK_TIMEOUT; then
        success "Rollback completed successfully"
        return 0
    else
        error "Rollback failed - manual intervention required"
        return 1
    fi
}

# Main blue-green deployment function
blue_green_deploy() {
    local current_env="blue"
    local target_env="green"
    local current_url="$BLUE_URL"
    local target_url="$GREEN_URL"
    
    # Determine current active environment
    if check_health "$GREEN_URL" 5 && ! check_health "$BLUE_URL" 5; then
        current_env="green"
        target_env="blue"
        current_url="$GREEN_URL"
        target_url="$BLUE_URL"
    fi
    
    log "Starting blue-green deployment"
    log "Current environment: $current_env ($current_url)"
    log "Target environment: $target_env ($target_url)"
    
    # Step 1: Deploy to target environment
    if ! deploy_to_target "$target_env" "${DEPLOYMENT_METHOD:-simulate}"; then
        error "Deployment to $target_env failed"
        exit 1
    fi
    
    # Step 2: Wait for target environment to become healthy
    if ! wait_for_health "$target_url" $VALIDATION_TIMEOUT; then
        error "Target environment failed health check"
        exit 1
    fi
    
    # Step 3: Run deployment validation
    if ! validate_deployment "$target_url"; then
        error "Deployment validation failed"
        exit 1
    fi
    
    # Step 4: Run performance test (optional)
    run_performance_test "$target_url"
    
    # Step 5: Switch traffic to target environment
    switch_traffic "$current_env" "$target_env"
    
    # Step 6: Verify traffic switch was successful
    sleep 10  # Allow time for traffic to switch
    
    if ! check_health "$target_url" 10; then
        error "Target environment unhealthy after traffic switch"
        rollback "$target_env" "$current_env"
        exit 1
    fi
    
    # Step 7: Monitor for a short period
    log "Monitoring new deployment for 30 seconds..."
    for i in {1..6}; do
        if ! check_health "$target_url" 5; then
            error "Health check failed during monitoring period"
            rollback "$target_env" "$current_env"
            exit 1
        fi
        sleep 5
        echo -n "."
    done
    echo
    
    success "Blue-green deployment completed successfully!"
    success "Active environment: $target_env ($target_url)"
    
    # Optional: Stop old environment
    if [ "${STOP_OLD_ENV:-false}" = "true" ]; then
        log "Stopping old environment: $current_env"
        # Implementation depends on deployment method
    fi
}

# Cleanup function
cleanup() {
    log "Cleaning up..."
    # Add any cleanup logic here
}

# Trap cleanup on exit
trap cleanup EXIT

# Usage information
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  --blue-url URL          Blue environment URL (default: $BLUE_URL)"
    echo "  --green-url URL         Green environment URL (default: $GREEN_URL)"
    echo "  --deployment-method M   Deployment method: fly, docker, simulate (default: simulate)"
    echo "  --stop-old-env          Stop old environment after successful deployment"
    echo ""
    echo "Environment Variables:"
    echo "  BLUE_URL               Blue environment URL"
    echo "  GREEN_URL              Green environment URL"
    echo "  ADMIN_TOKEN            Admin API token for validation"
    echo "  DATABASE_URL           Database connection string"
    echo "  REDIS_URL              Redis connection string"
    echo "  DEPLOYMENT_METHOD      Deployment method"
    echo "  LOAD_BALANCER          Load balancer type: nginx, haproxy, cloudflare, simulate"
    echo ""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        --blue-url)
            BLUE_URL="$2"
            shift 2
            ;;
        --green-url)
            GREEN_URL="$2"
            shift 2
            ;;
        --deployment-method)
            DEPLOYMENT_METHOD="$2"
            shift 2
            ;;
        --stop-old-env)
            STOP_OLD_ENV="true"
            shift
            ;;
        *)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate required tools
if ! command -v curl >/dev/null 2>&1; then
    error "curl is required but not installed"
    exit 1
fi

if ! command -v node >/dev/null 2>&1; then
    warning "node is not installed, deployment validation will be skipped"
fi

# Run the deployment
blue_green_deploy
