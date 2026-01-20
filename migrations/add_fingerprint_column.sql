-- Migration: Add fingerprint and trace_id columns for error grouping
-- This migration adds columns needed for error grouping functionality

-- Add fingerprint column for error grouping
ALTER TABLE errors ADD COLUMN fingerprint TEXT;

-- Add trace_id column for distributed tracing
ALTER TABLE errors ADD COLUMN trace_id TEXT;

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_errors_fingerprint ON errors(project_id, fingerprint, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_errors_project_fingerprint_status ON errors(project_id, fingerprint, status);
CREATE INDEX IF NOT EXISTS idx_errors_trace_id ON errors(trace_id);

-- Note: Existing errors will have NULL fingerprints
-- They will be automatically assigned fingerprints when queried by the application
