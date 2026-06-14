package pg

import (
	"context"
	"database/sql"
	"errors"
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
	orphan, _, err := s.ListByConnectionPaginated(ctx, connectionID, 0, 0)
	return orphan, err
}

func (s *OrphanStore) ListByConnectionPaginated(ctx context.Context, connectionID string, offset, limit int) ([]domain.Orphan, int64, error) {
	var total int64
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orphans o
		 JOIN relationships r ON r.id = o.relationship_id
		 WHERE r.connection_id = $1`, connectionID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count orphans: %w", err)
	}

	query := `SELECT o.id, o.relationship_id, o.source_collection, o.source_field, o.missing_value, o.created_at
			  FROM orphans o
			  JOIN relationships r ON r.id = o.relationship_id
			  WHERE r.connection_id = $1
			  ORDER BY o.created_at DESC`
	args := []any{connectionID}

	if limit > 0 {
		query += ` LIMIT $2 OFFSET $3`
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orphans: %w", err)
	}
	defer rows.Close()

	var orphans []domain.Orphan
	for rows.Next() {
		var o domain.Orphan
		if err := rows.Scan(&o.ID, &o.RelationshipID, &o.SourceCollection, &o.SourceField, &o.MissingValue, &o.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan orphan: %w", err)
		}
		orphans = append(orphans, o)
	}
	return orphans, total, rows.Err()
}

func (s *OrphanStore) DeleteByConnection(ctx context.Context, connectionID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM orphans o USING relationships r
		 WHERE r.id = o.relationship_id AND r.connection_id = $1`, connectionID)
	return err
}

func (s *OrphanStore) Get(ctx context.Context, id string) (*domain.Orphan, error) {
	var o domain.Orphan
	err := s.db.QueryRowContext(ctx,
		`SELECT id, relationship_id, source_collection, source_field, missing_value, created_at
		 FROM orphans WHERE id = $1`, id,
	).Scan(&o.ID, &o.RelationshipID, &o.SourceCollection, &o.SourceField, &o.MissingValue, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("orphan not found: %s", id)
		}
		return nil, fmt.Errorf("get orphan: %w", err)
	}
	return &o, nil
}
