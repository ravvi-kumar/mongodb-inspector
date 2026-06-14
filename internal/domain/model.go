package domain

import (
	"time"
)

type Connection struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	ConnectionString string    `json:"connection_string"`
	Database         string    `json:"database"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type ScanStatus string

const (
	ScanStatusPending   ScanStatus = "pending"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
)

type Scan struct {
	ID           string     `json:"id"`
	ConnectionID string     `json:"connection_id"`
	Status       ScanStatus `json:"status"`
	SampleSize   int        `json:"sample_size"`
	Error        *string    `json:"error,omitempty"`
	RetryCount   int        `json:"retry_count"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CollectionField struct {
	ID              string  `json:"id"`
	ScanID          string  `json:"scan_id"`
	CollectionName  string  `json:"collection_name"`
	FieldName       string  `json:"field_name"`
	FieldType       string  `json:"field_type"`
	SampleValues    []any   `json:"sample_values"`
	IsCandidate     bool    `json:"is_candidate"`
	CandidateReason *string `json:"candidate_reason,omitempty"`
	DocumentCount   int     `json:"document_count"`
}

type RelationshipStatus string

const (
	RelationshipStatusSuggested RelationshipStatus = "suggested"
	RelationshipStatusApproved  RelationshipStatus = "approved"
	RelationshipStatusRejected  RelationshipStatus = "rejected"
)

type Relationship struct {
	ID               string             `json:"id"`
	ConnectionID     string             `json:"connection_id"`
	SourceCollection string             `json:"source_collection"`
	SourceField      string             `json:"source_field"`
	TargetCollection string             `json:"target_collection"`
	TargetField      string             `json:"target_field"`
	Confidence       float64            `json:"confidence"`
	MatchedValues    int                `json:"matched_values"`
	SampledValues    int                `json:"sampled_values"`
	Status           RelationshipStatus `json:"status"`
	Explanation      string             `json:"explanation"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

type Orphan struct {
	ID               string    `json:"id"`
	RelationshipID   string    `json:"relationship_id"`
	SourceCollection string    `json:"source_collection"`
	SourceField      string    `json:"source_field"`
	MissingValue     string    `json:"missing_value"`
	CreatedAt        time.Time `json:"created_at"`
}

type InvestigateResult struct {
	Root      InvestigationNode   `json:"root"`
	Tree      *InvestigationNode  `json:"tree"`
	Documents []FlatDocument      `json:"documents"`
	Metadata  *CollectionMetadata `json:"metadata,omitempty"`
}

type CollectionMetadata struct {
	DocumentCount int `json:"document_count"`
	FieldCount    int `json:"field_count"`
}

type InvestigationNode struct {
	Collection   string               `json:"collection"`
	ID           string               `json:"id"`
	Document     any                  `json:"document"`
	Relationship string               `json:"relationship,omitempty"`
	Children     []*InvestigationNode `json:"children"`
	Metadata     *NodeMetadata        `json:"metadata,omitempty"`
}

type NodeMetadata struct {
	Depth             int    `json:"depth"`
	SiblingCount      int    `json:"sibling_count"`
	RelationshipLabel string `json:"relationship_label,omitempty"`
}

type FlatDocument struct {
	Collection string `json:"collection"`
	ID         string `json:"id"`
	Document   any    `json:"document"`
}

type PaginatedResponse struct {
	Data   any `json:"data"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

type ScanSummary struct {
	ScanID          string `json:"scan_id"`
	Status          string `json:"status"`
	TotalFields     int    `json:"total_fields"`
	CandidateFields int    `json:"candidate_fields"`
	Relationships   int    `json:"relationships"`
	Orphans         int    `json:"orphans"`
	Collections     int    `json:"collections"`
}

type ConnectionStats struct {
	ConnectionID      string `json:"connection_id"`
	CollectionCount   int    `json:"collection_count"`
	FieldCount        int    `json:"field_count"`
	RelationshipCount int    `json:"relationship_count"`
	OrphanCount       int    `json:"orphan_count"`
}

type SchemaMap struct {
	Nodes []SchemaNode `json:"nodes"`
	Edges []SchemaEdge `json:"edges"`
}

type SchemaNode struct {
	Collection string `json:"collection"`
	FieldCount int    `json:"field_count"`
}

type SchemaEdge struct {
	ID               string  `json:"id"`
	SourceCollection string  `json:"source_collection"`
	SourceField      string  `json:"source_field"`
	TargetCollection string  `json:"target_collection"`
	TargetField      string  `json:"target_field"`
	Confidence       float64 `json:"confidence"`
}

type BatchInvestigateRequest struct {
	ConnectionID string   `json:"connection_id"`
	DocumentIDs  []string `json:"document_ids"`
}

type BatchInvestigateResult struct {
	Results map[string]InvestigateResult `json:"results"`
}

type RelationshipTrace struct {
	RelationshipID   string         `json:"relationship_id"`
	SourceCollection string         `json:"source_collection"`
	SourceField      string         `json:"source_field"`
	TargetCollection string         `json:"target_collection"`
	TargetField      string         `json:"target_field"`
	ForwardDocs      []FlatDocument `json:"forward_docs"`
	ReverseDocs      []FlatDocument `json:"reverse_docs"`
}
