#!/bin/bash

# Phase 2: Market Research & Monitoring Script
# Collects demand patterns, pricing trends, and competitive intelligence

LOG_DIR="/tmp/golem-market-research"
mkdir -p "$LOG_DIR"

echo "🔍 Starting Golem Market Research & Monitoring"
echo "📊 Data collection directory: $LOG_DIR"
echo "⏰ Started at: $(date)"

# Function to log with timestamp
log_with_timestamp() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# Monitor provider logs for market events
monitor_provider_logs() {
    log_with_timestamp "📈 Monitoring provider logs for market events..."
    docker compose -f /Users/Jammie/Desktop/Project\ Beacon/Website/golem-provider/docker-compose.yml \
        logs -f golem-provider 2>&1 | \
        grep -E "(market|demand|offer|agreement|negotiation|payment)" | \
        tee -a "$LOG_DIR/market-events-$(date +%Y%m%d).log"
}

# Check payment status periodically
monitor_payments() {
    while true; do
        log_with_timestamp "💰 Checking payment status..."
        docker compose -f /Users/Jammie/Desktop/Project\ Beacon/Website/golem-provider/docker-compose.yml \
            exec -T golem-provider yagna payment status --network holesky 2>/dev/null | \
            tee -a "$LOG_DIR/payment-status-$(date +%Y%m%d).log"
        sleep 300  # Check every 5 minutes
    done
}

# Monitor resource utilization
monitor_resources() {
    while true; do
        log_with_timestamp "🖥️  Monitoring resource utilization..."
        {
            echo "=== $(date) ==="
            docker stats beacon-golem-provider --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
        } >> "$LOG_DIR/resource-usage-$(date +%Y%m%d).log"
        sleep 60  # Check every minute
    done
}

# Main monitoring loop
main() {
    log_with_timestamp "🚀 Starting Phase 2 market monitoring..."
    
    # Start background monitoring processes
    monitor_payments &
    PAYMENT_PID=$!
    
    monitor_resources &
    RESOURCE_PID=$!
    
    # Monitor provider logs in foreground
    monitor_provider_logs
    
    # Cleanup on exit
    trap "kill $PAYMENT_PID $RESOURCE_PID 2>/dev/null" EXIT
}

# Run main function
main
