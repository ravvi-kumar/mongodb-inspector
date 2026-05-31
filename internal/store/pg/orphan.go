package pg

import (
	"context"
	"fmt"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
)

type OrphanStore struct {
	db DBTX
}

func NewOrphanStore(db DBTX) *OrphanStore {
	return &OrphanStore{db: db}
}

func (s *OrphanStore) Create(ctx context.Context, o *domain.Orphan) error {
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO orphans (relationship_id, source_collection, source_field, missing_value)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		o.RelationshipID, o.SourceCollection, o.SourceField, o.MissingValue,
	).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return fmt.Errorf("create orphan: %w", err)
	}
	return nil
}

func (s *OrphanStore) CreateBatch(ctx context.Context, orphans []domain.Orphan) error {
	if len(orphans) == 0 {
		return nil
	}

	for i := range orphans {
		o := &orphans[i]
		err := s.db.QueryRowContext(ctx,
			`INSERT INTO orphans (relationship_id, source_collection, source_field, missing_value)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id, created_at`,
			o.RelationshipID, o.SourceCollection, o.SourceField, o.MissingValue,
		).Scan(&o.ID, &o.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert orphan: %w", err)
		}
	}

	return nil
}

func (s *OrphanStore) ListByConnection(ctx context.Context, connectionID string) ([]domain.Orphan, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT o.id, o.relationship_id, o.source_collection, o.source_field, o.missing_value, o.created_at
		 FROM orphans o
		 JOIN relationships r ON r.id = o.relationship_id
		 WHERE r.connection_id = $1
		 ORDER BY o.created_at DESC`, connectionID)
	if err != nil {
		return nil, fmt.Errorf("list orphans: %w", err)
	}
	defer rows.Close()

	var orphans []domain.Orphan
	for rows.Next() {
		var o domain.Orphan
		if err := rows.Scan(&o.ID, &o.RelationshipID, &o.SourceCollection, &o.SourceField, &o.MissingValue, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan orphan: %w", err)
		}
		orphans = append(orphans, o)
	}
	return orphans, rows.Err()
}

func (s *OrphanStore) DeleteByConnection(ctx context.Context, connectionID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM orphans o USING relationships r
		 WHERE r.id = o.relationship_id AND r.connection_id = $1`, connectionID)
	return err
}
