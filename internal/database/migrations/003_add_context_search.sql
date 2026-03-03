-- Context Search: Add basic full-text search support
-- Note: If FTS5 is not available, full-text search will be limited

-- Try to create FTS5 table (may fail on systems without FTS5)
-- Application should handle this gracefully
