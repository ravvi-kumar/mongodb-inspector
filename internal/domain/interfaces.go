package domain

import "context"

type ConnectionReader interface {
	Get(ctx context.Context, id string) (*Connection, error)
}

type ScanReader interface {
	Get(ctx context.Context, id string) (*Scan, error)
	GetCandidateFields(ctx context.Context, scanID string) ([]CollectionField, error)
	GetFieldsByScan(ctx context.Context, scanID string) ([]CollectionField, error)
}

type RelationshipReaderWriter interface {
	Create(ctx context.Context, r *Relationship) error
	GetApproved(ctx context.Context, connectionID string) ([]Relationship, error)
	GetByScanConnection(ctx context.Context, scanID string) (string, error)
}

type OrphanReaderWriter interface {
	DeleteByConnection(ctx context.Context, connectionID string) error
	CreateBatch(ctx context.Context, orphans []Orphan) error
	ListByConnection(ctx context.Context, connectionID string) ([]Orphan, error)
}
