-- +goose Up
CREATE UNIQUE INDEX idx_relationships_unique_combination 
ON relationships(connection_id, source_collection, source_field, target_collection, target_field);

-- +goose Down
DROP INDEX idx_relationships_unique_combination;