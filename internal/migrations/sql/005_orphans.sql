-- +goose Up
CREATE TABLE orphans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    relationship_id UUID NOT NULL REFERENCES relationships(id) ON DELETE CASCADE,
    source_collection TEXT NOT NULL,
    source_field TEXT NOT NULL,
    missing_value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orphans_relationship_id ON orphans(relationship_id);
CREATE INDEX idx_orphans_source ON orphans(source_collection, source_field);

-- +goose Down
DROP TABLE orphans;
