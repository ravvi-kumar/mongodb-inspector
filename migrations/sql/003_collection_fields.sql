-- +goose Up
CREATE TABLE collection_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    collection_name TEXT NOT NULL,
    field_name TEXT NOT NULL,
    field_type TEXT NOT NULL,
    sample_values JSONB NOT NULL DEFAULT '[]',
    is_candidate BOOLEAN NOT NULL DEFAULT false,
    candidate_reason TEXT,
    document_count INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_collection_fields_scan_id ON collection_fields(scan_id);
CREATE INDEX idx_collection_fields_candidate ON collection_fields(scan_id, is_candidate) WHERE is_candidate = true;

-- +goose Down
DROP TABLE collection_fields;
