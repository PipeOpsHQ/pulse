# Database Migrations

This directory contains database migration files for the Sentry Alt application.

## How It Works

1. **Automatic Execution**: Migrations run automatically when the Docker container starts
2. **Idempotent**: Each migration runs only once, tracked in the `schema_migrations` table
3. **Sequential**: Migrations are applied in alphabetical order by filename

## Migration Naming Convention

Use the format: `YYYY-MM-DD_description.sql`

Example:
- `2024-01-15_add_fingerprint_column.sql`
- `2024-02-20_add_user_settings.sql`

## Creating a New Migration

1. Create a new `.sql` file in this directory
2. Write idempotent SQL (use `IF NOT EXISTS`, `IF EXISTS`, etc.)
3. Test locally before deploying

Example migration file:

```sql
-- Migration: Add new feature
-- Description: Adds support for X

-- Add new column
ALTER TABLE tablename ADD COLUMN IF NOT EXISTS new_column TEXT;

-- Create index
CREATE INDEX IF NOT EXISTS idx_name ON tablename(column);
```

## Best Practices

- ✅ Use `IF NOT EXISTS` and `IF EXISTS` for safety
- ✅ Include comments describing what the migration does
- ✅ Test migrations on a copy of production data first
- ✅ Keep migrations small and focused
- ❌ Don't modify existing migration files after they've been deployed
- ❌ Don't use destructive operations (DROP, DELETE) without careful consideration

## Manual Migration Execution

If you need to run migrations manually:

```bash
# Using Docker
docker exec -it <container_name> /root/scripts/run-migrations.sh

# Or directly with sqlite3
sqlite3 /path/to/sentry.db < migrations/your_migration.sql
```

## Checking Migration Status

```bash
# Connect to database
sqlite3 /path/to/sentry.db

# List applied migrations
SELECT * FROM schema_migrations ORDER BY applied_at;
```

## Troubleshooting

### Migration Fails on Startup

1. Check container logs: `docker logs <container_name>`
2. The application won't start if a migration fails
3. Fix the migration SQL and redeploy

### Migration Already Exists Error

If you need to retry a failed migration:

```sql
-- Remove the migration record
DELETE FROM schema_migrations WHERE migration_name = 'your_migration.sql';
```

Then restart the container to retry.
