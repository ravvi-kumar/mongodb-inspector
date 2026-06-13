package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
)

type RelationshipStore struct {
	db DBTX
}

func NewRelationshipStore(db DBTX) *RelationshipStore {
	return &RelationshipStore{db: db}
}

func (s *RelationshipStore) Create(ctx context.Context, r *domain.Relationship) error {
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO relationships (connection_id, source_collection, source_field, target_collection, target_field, confidence, matched_values, sampled_values, status, explanation)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, created_at, updated_at`,
		r.ConnectionID, r.SourceCollection, r.SourceField, r.TargetCollection, r.TargetField,
		r.Confidence, r.MatchedValues, r.SampledValues, string(r.Status), r.Explanation,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create relationship: %w", err)
	}
	return nil
}

func (s *RelationshipStore) CreateOrSkip(ctx context.Context, r *domain.Relationship) (bool, error) {
	now := time.Now()
	r.CreatedAt = now
	r.UpdatedAt = now

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO relationships (connection_id, source_collection, source_field, target_collection, target_field, confidence, matched_values, sampled_values, status, explanation)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (connection_id, source_collection, source_field, target_collection, target_field) DO NOTHING
		 RETURNING id, created_at, updated_at`,
		r.ConnectionID, r.SourceCollection, r.SourceField, r.TargetCollection, r.TargetField,
		r.Confidence, r.MatchedValues, r.SampledValues, string(r.Status), r.Explanation,
	).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("create or skip relationship: %w", err)
	}
	return true, nil
}

func (s *RelationshipStore) Get(ctx context.Context, id string) (*domain.Relationship, error) {
	var r domain.Relationship
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, connection_id, source_collection, source_field, target_collection, target_field, confidence, matched_values, sampled_values, status, explanation, created_at, updated_at
		 FROM relationships WHERE id = $1`, id,
	).Scan(&r.ID, &r.ConnectionID, &r.SourceCollection, &r.SourceField, &r.TargetCollection, &r.TargetField, &r.Confidence, &r.MatchedValues, &r.SampledValues, &status, &r.Explanation, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get relationship: %w", err)
	}
	r.Status = domain.RelationshipStatus(status)
	return &r, nil
}

func (s *RelationshipStore) List(ctx context.Context, connectionID string, statusFilter *string) ([]domain.Relationship, error) {
	rels, _, err := s.ListPaginated(ctx, connectionID, statusFilter, 0, 0)
	return rels, err
}

func (s *RelationshipStore) ListPaginated(ctx context.Context, connectionID string, statusFilter *string, offset, limit int) ([]domain.Relationship, int64, error) {
	query := `SELECT COUNT(*) FROM relationships WHERE connection_id = $1`
	args := []any{connectionID}

	if statusFilter != nil {
		query += ` AND status = $2`
		args = append(args, *statusFilter)
	}

	var total int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count relationships: %w", err)
	}

	query = `SELECT id, connection_id, source_collection, source_field, target_collection, target_field, confidence, matched_values, sampled_values, status, explanation, created_at, updated_at
			  FROM relationships WHERE connection_id = $1`
	args = []any{connectionID}

	if statusFilter != nil {
		query += ` AND status = $2`
		args = append(args, *statusFilter)
	}

	query += ` ORDER BY confidence DESC`

	if limit > 0 {
		query += ` LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list relationships: %w", err)
	}
	defer rows.Close()

	var rels []domain.Relationship
	for rows.Next() {
		var r domain.Relationship
		var status string
		if err := rows.Scan(&r.ID, &r.ConnectionID, &r.SourceCollection, &r.SourceField, &r.TargetCollection, &r.TargetField, &r.Confidence, &r.MatchedValues, &r.SampledValues, &status, &r.Explanation, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan relationship: %w", err)
		}
		r.Status = domain.RelationshipStatus(status)
		rels = append(rels, r)
	}
	return rels, total, rows.Err()
}

func (s *RelationshipStore) UpdateStatus(ctx context.Context, id string, status domain.RelationshipStatus) (*domain.Relationship, error) {
	var r domain.Relationship
	var s2 string
	err := s.db.QueryRowContext(ctx,
		`UPDATE relationships SET status = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING id, connection_id, source_collection, source_field, target_collection, target_field, confidence, matched_values, sampled_values, status, explanation, created_at, updated_at`,
		id, string(status),
	).Scan(&r.ID, &r.ConnectionID, &r.SourceCollection, &r.SourceField, &r.TargetCollection, &r.TargetField, &r.Confidence, &r.MatchedValues, &r.SampledValues, &s2, &r.Explanation, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update relationship status: %w", err)
	}
	r.Status = domain.RelationshipStatus(s2)
	return &r, nil
}

func (s *RelationshipStore) GetApproved(ctx context.Context, connectionID string) ([]domain.Relationship, error) {
	status := string(domain.RelationshipStatusApproved)
	rels, _, err := s.ListPaginated(ctx, connectionID, &status, 0, 0)
	return rels, err
}

func (s *RelationshipStore) GetByScanConnection(ctx context.Context, scanID string) (string, error) {
	var connectionID string
	err := s.db.QueryRowContext(ctx,
		`SELECT connection_id FROM scans WHERE id = $1`, scanID,
	).Scan(&connectionID)
	if err != nil {
		return "", fmt.Errorf("get scan connection: %w", err)
	}
	return connectionID, nil
}

func (s *RelationshipStore) DeleteByConnection(ctx context.Context, connectionID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM relationships WHERE connection_id = $1`, connectionID)
	return err
}
