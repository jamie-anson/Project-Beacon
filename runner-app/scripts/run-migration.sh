#!/bin/bash
# Run database migration for per-question execution feature

set -e

echo "Running database migration: 007_add_question_id_to_executions.sql"
echo ""

# Get DATABASE_URL from Fly.io
echo "Getting DATABASE_URL from Fly.io secrets..."
DATABASE_URL=$(flyctl secrets list --app beacon-runner-change-me --json | jq -r '.[] | select(.Name == "DATABASE_URL") | .Value')

if [ -z "$DATABASE_URL" ]; then
    echo "Error: Could not get DATABASE_URL from Fly.io"
    echo ""
    echo "Please run manually:"
    echo "  flyctl ssh console --app beacon-runner-change-me"
    echo "  Then inside the container:"
    echo "  echo \$DATABASE_URL"
    echo "  Copy the URL and run:"
    echo "  psql <DATABASE_URL> -f migrations/007_add_question_id_to_executions.sql"
    exit 1
fi

echo "DATABASE_URL found!"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "Error: psql is not installed"
    echo ""
    echo "Please install PostgreSQL client:"
    echo "  brew install postgresql"
    echo ""
    echo "Or run the migration manually with the DATABASE_URL above"
    exit 1
fi

echo "Running migration..."
psql "$DATABASE_URL" -f migrations/007_add_question_id_to_executions.sql

echo ""
echo "Migration completed successfully!"
echo ""
echo "Verifying changes..."
psql "$DATABASE_URL" -c "\d executions" | grep question_id

echo ""
echo "Done! You can now deploy the updated runner app."
