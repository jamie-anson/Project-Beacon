#!/bin/bash
# Quick script to view Sentry logs for beacon-runner-production

ORG="jamie-anson"
PROJECT="beacon-runner-production"

echo "📊 Sentry Logs for $PROJECT"
echo "================================"
echo ""

echo "Recent Issues:"
sentry-cli issues list --project $PROJECT --org $ORG

echo ""
echo "Recent Events (last 20):"
sentry-cli events list --project $PROJECT --org $ORG --max-rows 20

echo ""
echo "💡 Tip: Run 'flyctl ext sentry dashboard -a beacon-runner-production' to open web UI"
