package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
)

type ScanStore struct {
	db DBTX
}

func NewScanStore(db DBTX) *ScanStore {
	return &ScanStore{db: db}
}

func (s *ScanStore) Create(ctx context.Context, scan *domain.Scan) error {
	now := time.Now()
	scan.CreatedAt = now

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO scans (connection_id, status, sample_size)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		scan.ConnectionID, string(scan.Status), scan.SampleSize,
	).Scan(&scan.ID, &scan.CreatedAt)
	if err != nil {
		return fmt.Errorf("create scan: %w", err)
	}
	return nil
}

func (s *ScanStore) Get(ctx context.Context, id string) (*domain.Scan, error) {
	var scan domain.Scan
	var status string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, connection_id, status, sample_size, error, started_at, completed_at, created_at
		 FROM scans WHERE id = $1`, id,
	).Scan(&scan.ID, &scan.ConnectionID, &status, &scan.SampleSize, &scan.Error, &scan.StartedAt, &scan.CompletedAt, &scan.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get scan: %w", err)
	}
	scan.Status = domain.ScanStatus(status)
	return &scan, nil
}

func (s *ScanStore) UpdateStatus(ctx context.Context, id string, status domain.ScanStatus, errMsg *string) error {
	if status == domain.ScanStatusRunning {
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = $2, started_at = now() WHERE id = $1`,
			id, string(status))
		return err
	}
	if status == domain.ScanStatusCompleted {
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = $2, completed_at = now() WHERE id = $1`,
			id, string(status))
		return err
	}
	if status == domain.ScanStatusFailed {
		_, err := s.db.ExecContext(ctx,
			`UPDATE scans SET status = $2, error = $3, completed_at = now() WHERE id = $1`,
			id, string(status), errMsg)
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE scans SET status = $2 WHERE id = $1`,
		id, string(status))
	return err
}

func (s *ScanStore) ListByConnection(ctx context.Context, connectionID string) ([]domain.Scan, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, connection_id, status, sample_size, error, started_at, completed_at, created_at
		 FROM scans WHERE connection_id = $1 ORDER BY created_at DESC`, connectionID)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()

	var scans []domain.Scan
	for rows.Next() {
		var scan domain.Scan
		var status string
		if err := rows.Scan(&scan.ID, &scan.ConnectionID, &status, &scan.SampleSize, &scan.Error, &scan.StartedAt, &scan.CompletedAt, &scan.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		scan.Status = domain.ScanStatus(status)
		scans = append(scans, scan)
	}
	return scans, rows.Err()
}

func (s *ScanStore) InsertField(ctx context.Context, f *domain.CollectionField) error {
	values, err := json.Marshal(f.SampleValues)
	if err != nil {
		return fmt.Errorf("marshal sample values: %w", err)
	}

	err = s.db.QueryRowContext(ctx,
		`INSERT INTO collection_fields (scan_id, collection_name, field_name, field_type, sample_values, is_candidate, candidate_reason, document_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		f.ScanID, f.CollectionName, f.FieldName, f.FieldType, values, f.IsCandidate, f.CandidateReason, f.DocumentCount,
	).Scan(&f.ID)
	if err != nil {
		return fmt.Errorf("insert field: %w", err)
	}
	return nil
}

func (s *ScanStore) GetFieldsByScan(ctx context.Context, scanID string) ([]domain.CollectionField, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scan_id, collection_name, field_name, field_type, sample_values, is_candidate, candidate_reason, document_count
		 FROM collection_fields WHERE scan_id = $1
		 ORDER BY collection_name, field_name`, scanID)
	if err != nil {
		return nil, fmt.Errorf("get fields: %w", err)
	}
	defer rows.Close()

	var fields []domain.CollectionField
	for rows.Next() {
		var f domain.CollectionField
		var values []byte
		if err := rows.Scan(&f.ID, &f.ScanID, &f.CollectionName, &f.FieldName, &f.FieldType, &values, &f.IsCandidate, &f.CandidateReason, &f.DocumentCount); err != nil {
			return nil, fmt.Errorf("scan field row: %w", err)
		}
		if err := json.Unmarshal(values, &f.SampleValues); err != nil {
			return nil, fmt.Errorf("unmarshal sample values: %w", err)
		}
		fields = append(fields, f)
	}
	return fields, rows.Err()
}

func (s *ScanStore) GetCandidateFields(ctx context.Context, scanID string) ([]domain.CollectionField, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scan_id, collection_name, field_name, field_type, sample_values, is_candidate, candidate_reason, document_count
		 FROM collection_fields WHERE scan_id = $1 AND is_candidate = true
		 ORDER BY collection_name, field_name`, scanID)
	if err != nil {
		return nil, fmt.Errorf("get candidate fields: %w", err)
	}
	defer rows.Close()

	var fields []domain.CollectionField
	for rows.Next() {
		var f domain.CollectionField
		var values []byte
		if err := rows.Scan(&f.ID, &f.ScanID, &f.CollectionName, &f.FieldName, &f.FieldType, &values, &f.IsCandidate, &f.CandidateReason, &f.DocumentCount); err != nil {
			return nil, fmt.Errorf("scan field row: %w", err)
		}
		if err := json.Unmarshal(values, &f.SampleValues); err != nil {
			return nil, fmt.Errorf("unmarshal sample values: %w", err)
		}
		fields = append(fields, f)
	}
	return fields, rows.Err()
}
