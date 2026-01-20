#!/bin/sh
set -e

# Database file location
DB_FILE="${DB_PATH:-/root/data/sentry.db}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-/root/migrations}"

echo "Starting database migrations..."
echo "Database: $DB_FILE"
echo "Migrations directory: $MIGRATIONS_DIR"

# Check if database file exists
if [ ! -f "$DB_FILE" ]; then
    echo "Database file does not exist yet. Will be created by application."
    exit 0
fi

# Check if migrations directory exists
if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "Migrations directory not found: $MIGRATIONS_DIR"
    echo "Skipping migrations."
    exit 0
fi

# Create migrations tracking table if it doesn't exist
sqlite3 "$DB_FILE" <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    migration_name TEXT NOT NULL UNIQUE,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
EOF

echo "Checking for pending migrations..."

# Process each .sql file in migrations directory
for migration_file in "$MIGRATIONS_DIR"/*.sql; do
    # Skip if no .sql files found
    [ -e "$migration_file" ] || continue
    
    # Get just the filename
    migration_name=$(basename "$migration_file")
    
    # Check if migration has already been applied
    applied=$(sqlite3 "$DB_FILE" "SELECT COUNT(*) FROM schema_migrations WHERE migration_name = '$migration_name';")
    
    if [ "$applied" -eq 0 ]; then
        echo "Applying migration: $migration_name"
        
        # Apply the migration (capture output for better error handling)
        migration_output=$(sqlite3 "$DB_FILE" < "$migration_file" 2>&1) || migration_exit_code=$?
        
        # Check if migration succeeded or if it's a benign error (like duplicate column)
        if [ -z "$migration_exit_code" ] || [ "$migration_exit_code" -eq 0 ]; then
            # Record successful migration
            sqlite3 "$DB_FILE" "INSERT INTO schema_migrations (migration_name) VALUES ('$migration_name');"
            echo "✓ Migration applied successfully: $migration_name"
        elif echo "$migration_output" | grep -qi "duplicate column"; then
            # Column already exists - this is OK, mark as applied
            echo "⚠ Migration partially applied (columns already exist): $migration_name"
            sqlite3 "$DB_FILE" "INSERT INTO schema_migrations (migration_name) VALUES ('$migration_name');" || true
            echo "✓ Marked as applied: $migration_name"
        else
            # Real error
            echo "✗ Migration failed: $migration_name"
            echo "Error output: $migration_output"
            exit 1
        fi
        migration_exit_code=0  # Reset for next iteration
    else
        echo "⊘ Migration already applied: $migration_name"
    fi
done

echo "All migrations completed successfully!"
