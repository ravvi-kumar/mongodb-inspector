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
	CreateOrSkip(ctx context.Context, r *Relationship) (bool, error)
	Get(ctx context.Context, id string) (*Relationship, error)
	GetApproved(ctx context.Context, connectionID string) ([]Relationship, error)
	GetByScanConnection(ctx context.Context, scanID string) (string, error)
	List(ctx context.Context, connectionID string, statusFilter *string) ([]Relationship, error)
	ListPaginated(ctx context.Context, connectionID string, statusFilter *string, offset, limit int) ([]Relationship, int64, error)
	SearchByCollection(ctx context.Context, connectionID string, collectionName string) ([]Relationship, error)
}

type OrphanReaderWriter interface {
	DeleteByConnection(ctx context.Context, connectionID string) error
	CreateBatch(ctx context.Context, orphans []Orphan) error
	ListByConnection(ctx context.Context, connectionID string) ([]Orphan, error)
	ListByConnectionPaginated(ctx context.Context, connectionID string, offset, limit int) ([]Orphan, int64, error)
	Get(ctx context.Context, id string) (*Orphan, error)
}

type StatsReader interface {
	GetConnectionStats(ctx context.Context, connectionID string) (*ConnectionStats, error)
}
