#!/bin/bash
# Clear all Redis queues and stop runner jobs
# This will flush job queues without touching the database

set -e

echo "🧹 Clearing Redis Queues and Runner Jobs"
echo "=========================================="

# Check if redis-cli is available
if ! command -v redis-cli &> /dev/null; then
    echo "❌ redis-cli not found. Please install redis-cli or run commands manually."
    exit 1
fi

# Get Redis connection info (adjust if needed)
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_URL="${REDIS_URL:-}"

# Use REDIS_URL if set, otherwise use host:port
if [ -n "$REDIS_URL" ]; then
    REDIS_CMD="redis-cli -u $REDIS_URL"
else
    REDIS_CMD="redis-cli -h $REDIS_HOST -p $REDIS_PORT"
fi

echo ""
echo "📊 Current Queue Status:"
echo "------------------------"

# Check main jobs queue
JOBS_QUEUE_LEN=$($REDIS_CMD LLEN jobs 2>/dev/null || echo "0")
echo "Main queue (jobs): $JOBS_QUEUE_LEN messages"

# Check dead-letter queue
DEAD_QUEUE_LEN=$($REDIS_CMD LLEN jobs:dead 2>/dev/null || echo "0")
echo "Dead-letter queue (jobs:dead): $DEAD_QUEUE_LEN messages"

# Check processing set (if using advanced queue)
PROCESSING_LEN=$($REDIS_CMD ZCARD jobs:processing 2>/dev/null || echo "0")
echo "Processing set (jobs:processing): $PROCESSING_LEN jobs"

echo ""
echo "🗑️  Clearing Queues..."
echo "------------------------"

# Clear main jobs queue
if [ "$JOBS_QUEUE_LEN" -gt 0 ]; then
    $REDIS_CMD DEL jobs
    echo "✅ Cleared main queue (jobs): $JOBS_QUEUE_LEN messages deleted"
else
    echo "✅ Main queue already empty"
fi

# Clear dead-letter queue
if [ "$DEAD_QUEUE_LEN" -gt 0 ]; then
    $REDIS_CMD DEL jobs:dead
    echo "✅ Cleared dead-letter queue (jobs:dead): $DEAD_QUEUE_LEN messages deleted"
else
    echo "✅ Dead-letter queue already empty"
fi

# Clear processing set
if [ "$PROCESSING_LEN" -gt 0 ]; then
    $REDIS_CMD DEL jobs:processing
    echo "✅ Cleared processing set (jobs:processing): $PROCESSING_LEN jobs deleted"
else
    echo "✅ Processing set already empty"
fi

# Clear any retry queues
$REDIS_CMD DEL jobs:retry 2>/dev/null || true
echo "✅ Cleared retry queue (if exists)"

echo ""
echo "📊 Final Queue Status:"
echo "------------------------"
JOBS_QUEUE_LEN=$($REDIS_CMD LLEN jobs 2>/dev/null || echo "0")
DEAD_QUEUE_LEN=$($REDIS_CMD LLEN jobs:dead 2>/dev/null || echo "0")
PROCESSING_LEN=$($REDIS_CMD ZCARD jobs:processing 2>/dev/null || echo "0")

echo "Main queue (jobs): $JOBS_QUEUE_LEN"
echo "Dead-letter queue (jobs:dead): $DEAD_QUEUE_LEN"
echo "Processing set (jobs:processing): $PROCESSING_LEN"

echo ""
echo "✅ Queue cleanup complete!"
echo ""
echo "⚠️  Note: This only cleared the queues. To stop running jobs in the runner:"
echo "   1. Restart the runner service (Fly.io: fly apps restart beacon-runner-production)"
echo "   2. Or use the admin API to cancel individual jobs"
