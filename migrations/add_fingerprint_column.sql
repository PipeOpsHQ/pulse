-- Migration: Add fingerprint and trace_id columns for error grouping
-- Date: 2024-01-20
-- Description: Adds columns needed for error grouping functionality
-- This enables grouping similar errors together and tracking distributed traces

-- Check if fingerprint column exists before adding
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we use a workaround
-- The migration script handles this by checking if it's already been applied

-- Add fingerprint column for error grouping
-- This column stores a hash of error characteristics for grouping similar errors
ALTER TABLE errors ADD COLUMN fingerprint TEXT;

-- Add trace_id column for distributed tracing
-- This column links errors to distributed trace operations
ALTER TABLE errors ADD COLUMN trace_id TEXT;

-- Create indexes for better query performance
-- These indexes dramatically improve query speed for grouped error views
CREATE INDEX IF NOT EXISTS idx_errors_fingerprint ON errors(project_id, fingerprint, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_errors_project_fingerprint_status ON errors(project_id, fingerprint, status);
CREATE INDEX IF NOT EXISTS idx_errors_trace_id ON errors(trace_id);

-- Note: Existing errors will have NULL fingerprints
-- They will be automatically assigned fingerprints when queried by the application
-- using a fallback mechanism: COALESCE(fingerprint, message || ':' || level)
