-- +goose Up
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id UUID NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    source_collection TEXT NOT NULL,
    source_field TEXT NOT NULL,
    target_collection TEXT NOT NULL,
    target_field TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    matched_values INT NOT NULL DEFAULT 0,
    sampled_values INT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'suggested' CHECK (status IN ('suggested', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_relationships_connection_id ON relationships(connection_id);
CREATE INDEX idx_relationships_status ON relationships(connection_id, status);
CREATE INDEX idx_relationships_source ON relationships(source_collection, source_field);
CREATE INDEX idx_relationships_target ON relationships(target_collection, target_field);

-- +goose Down
DROP TABLE relationships;
