#!/bin/sh
set -e

echo "===================="
echo "Sentry Alt Starting"
echo "===================="

# Set default database path if not provided
export DB_PATH="${DB_PATH:-/root/data/sentry.db}"

# Run database migrations
if [ -f /root/scripts/run-migrations.sh ]; then
    echo "Running database migrations..."
    /root/scripts/run-migrations.sh
else
    echo "Migration script not found, skipping migrations"
fi

echo ""
echo "Starting application..."
echo "===================="

# Execute the main application
exec "$@"
