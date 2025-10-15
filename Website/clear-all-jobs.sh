#!/bin/bash
# Master script to clear ALL jobs from queues and runner
# Runs both queue cleanup and runner restart

set -e

echo "🧹 COMPLETE JOB CLEANUP"
echo "======================="
echo "This will:"
echo "  1. Clear all Redis queues (jobs, jobs:dead, jobs:processing)"
echo "  2. Restart the runner to stop in-flight jobs"
echo ""
read -p "Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Cancelled"
    exit 1
fi

echo ""
echo "Step 1: Clearing Redis Queues"
echo "=============================="
bash "$(dirname "$0")/clear-queues.sh"

echo ""
echo "Step 2: Restarting Runner"
echo "========================="
bash "$(dirname "$0")/stop-runner-jobs.sh"

echo ""
echo "✅ COMPLETE CLEANUP FINISHED!"
echo "=============================="
echo ""
echo "Summary:"
echo "  ✅ Redis queues cleared"
echo "  ✅ Runner restarted"
echo "  ✅ All in-flight jobs stopped"
echo ""
echo "Next steps:"
echo "  - Check portal to verify no jobs are running"
echo "  - Submit a new test job to verify system is working"
