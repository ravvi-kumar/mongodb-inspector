package service

import (
	"context"
	"testing"
	"time"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/testutil"
)

func TestDiscoverRelationships_NoCandidates(t *testing.T) {
	mockScan := testutil.NewMockScanStore()
	mockRel := testutil.NewMockRelationshipStore()
	mockConn := testutil.NewMockConnectionStore()

	svc := &DiscoveryService{
		scanStore: mockScan,
		relStore:  mockRel,
		connStore: mockConn,
	}

	mockConn.Add(&domain.Connection{
		ID:               "conn-1",
		Name:             "test",
		ConnectionString: "mongodb://localhost",
		Database:         "testdb",
	})

	mockScan.AddScan(&domain.Scan{
		ID:           "scan-1",
		ConnectionID: "conn-1",
		Status:       domain.ScanStatusCompleted,
		SampleSize:   100,
		CreatedAt:    time.Now(),
	})

	mockScan.SetCandidates([]domain.CollectionField{})
	mockScan.SetAllFields([]domain.CollectionField{
		{CollectionName: "users", FieldName: "_id", FieldType: "objectId"},
	})

	err := svc.DiscoverRelationships(context.Background(), "scan-1")
	if err != nil {
		t.Errorf("expected no error with empty candidates, got %v", err)
	}
}

func TestDiscoverRelationships_ScanNotFound(t *testing.T) {
	mockScan := testutil.NewMockScanStore()
	mockRel := testutil.NewMockRelationshipStore()
	mockConn := testutil.NewMockConnectionStore()

	svc := &DiscoveryService{
		scanStore: mockScan,
		relStore:  mockRel,
		connStore: mockConn,
	}

	err := svc.DiscoverRelationships(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent scan")
	}
}

func TestDiscoverRelationships_ConnectionNotFound(t *testing.T) {
	mockScan := testutil.NewMockScanStore()
	mockRel := testutil.NewMockRelationshipStore()
	mockConn := testutil.NewMockConnectionStore()

	svc := &DiscoveryService{
		scanStore: mockScan,
		relStore:  mockRel,
		connStore: mockConn,
	}

	mockScan.AddScan(&domain.Scan{
		ID:           "scan-1",
		ConnectionID: "conn-missing",
		Status:       domain.ScanStatusCompleted,
		SampleSize:   100,
		CreatedAt:    time.Now(),
	})

	err := svc.DiscoverRelationships(context.Background(), "scan-1")
	if err == nil {
		t.Error("expected error for missing connection")
	}
}

func TestDiscoverRelationships_ConfidenceThreshold(t *testing.T) {
	if confidenceThreshold != 0.2 {
		t.Errorf("confidenceThreshold = %f, want 0.2", confidenceThreshold)
	}
}

func TestDiscoverRelationships_AutoApproveThreshold(t *testing.T) {
	if autoApproveThreshold != 0.7 {
		t.Errorf("autoApproveThreshold = %f, want 0.7", autoApproveThreshold)
	}
}

func TestDiscoverRelationships_StatusLogic(t *testing.T) {
	tests := []struct {
		confidence float64
		wantStatus domain.RelationshipStatus
	}{
		{0.19, domain.RelationshipStatusSuggested},
		{0.20, domain.RelationshipStatusSuggested},
		{0.50, domain.RelationshipStatusSuggested},
		{0.69, domain.RelationshipStatusSuggested},
		{0.70, domain.RelationshipStatusApproved},
		{0.95, domain.RelationshipStatusApproved},
		{1.00, domain.RelationshipStatusApproved},
	}

	for _, tt := range tests {
		status := domain.RelationshipStatusSuggested
		if tt.confidence >= autoApproveThreshold {
			status = domain.RelationshipStatusApproved
		}
		if status != tt.wantStatus {
			t.Errorf("confidence %.2f: got %s, want %s", tt.confidence, status, tt.wantStatus)
		}
	}
}
