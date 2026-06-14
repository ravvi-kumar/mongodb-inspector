-- +goose Up
-- Adding explanation field to relationships for Sprint 9 Discovery V2
-- This column stores a human-readable explanation of why the relationship was detected,
-- including matched/sampled counts, field name signals, type compatibility, etc.
ALTER TABLE relationships ADD COLUMN explanation TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE relationships DROP COLUMN explanation;
