#!/bin/bash

set -e

# Load environment variables
if [ -f .env ]; then
    source .env
fi

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL is not set"
    exit 1
fi

# Run all migrations in order
echo "Running migrations..."

for migration in migrations/*.sql; do
    if [ -f "$migration" ]; then
        echo "Applying: $(basename $migration)"
        psql "$DATABASE_URL" -f "$migration"
    fi
done

echo "Migrations completed successfully"
