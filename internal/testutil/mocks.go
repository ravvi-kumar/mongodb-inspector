package testutil

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
)

type MockConnectionStore struct {
	mu   sync.RWMutex
	data map[string]*domain.Connection
}

func NewMockConnectionStore() *MockConnectionStore {
	return &MockConnectionStore{data: make(map[string]*domain.Connection)}
}

func (m *MockConnectionStore) Add(c *domain.Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[c.ID] = c
}

func (m *MockConnectionStore) Get(ctx context.Context, id string) (*domain.Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.data[id]
	if !ok {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	return c, nil
}

type MockScanStore struct {
	mu         sync.RWMutex
	scans      map[string]*domain.Scan
	candidates []domain.CollectionField
	allFields  []domain.CollectionField
}

func NewMockScanStore() *MockScanStore {
	return &MockScanStore{scans: make(map[string]*domain.Scan)}
}

func (m *MockScanStore) AddScan(s *domain.Scan) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scans[s.ID] = s
}

func (m *MockScanStore) SetCandidates(fields []domain.CollectionField) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.candidates = fields
}

func (m *MockScanStore) SetAllFields(fields []domain.CollectionField) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allFields = fields
}

func (m *MockScanStore) Get(ctx context.Context, id string) (*domain.Scan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.scans[id]
	if !ok {
		return nil, fmt.Errorf("scan not found: %s", id)
	}
	return s, nil
}

func (m *MockScanStore) GetCandidateFields(ctx context.Context, scanID string) ([]domain.CollectionField, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.candidates, nil
}

func (m *MockScanStore) GetFieldsByScan(ctx context.Context, scanID string) ([]domain.CollectionField, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.allFields, nil
}

type MockRelationshipStore struct {
	mu    sync.RWMutex
	rels  map[string]*domain.Relationship
	connIDByScan map[string]string
}

func NewMockRelationshipStore() *MockRelationshipStore {
	return &MockRelationshipStore{
		rels:         make(map[string]*domain.Relationship),
		connIDByScan: make(map[string]string),
	}
}

func (m *MockRelationshipStore) Add(r *domain.Relationship) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rels[r.ID] = r
}

func (m *MockRelationshipStore) SetScanConnection(scanID, connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connIDByScan[scanID] = connID
}

func (m *MockRelationshipStore) Create(ctx context.Context, r *domain.Relationship) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rels[r.ID] = r
	return nil
}

func (m *MockRelationshipStore) CreateOrSkip(ctx context.Context, r *domain.Relationship) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rel := range m.rels {
		if rel.ConnectionID == r.ConnectionID && rel.SourceCollection == r.SourceCollection && rel.SourceField == r.SourceField && rel.TargetCollection == r.TargetCollection && rel.TargetField == r.TargetField {
			return false, nil
		}
	}
	m.rels[r.ID] = r
	return true, nil
}

func (m *MockRelationshipStore) List(ctx context.Context, connectionID string, statusFilter *string) ([]domain.Relationship, error) {
	rels, _, err := m.ListPaginated(ctx, connectionID, statusFilter, 0, 0)
	return rels, err
}

func (m *MockRelationshipStore) ListPaginated(ctx context.Context, connectionID string, statusFilter *string, offset, limit int) ([]domain.Relationship, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.Relationship
	for _, r := range m.rels {
		if r.ConnectionID != connectionID {
			continue
		}
		if statusFilter != nil && string(r.Status) != *statusFilter {
			continue
		}
		result = append(result, *r)
	}
	total := int64(len(result))
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if offset >= len(result) {
		return []domain.Relationship{}, total, nil
	}
	if limit > 0 && offset+limit < len(result) {
		result = result[offset : offset+limit]
	} else {
		result = result[offset:]
	}
	return result, total, nil
}

func (m *MockRelationshipStore) GetApproved(ctx context.Context, connectionID string) ([]domain.Relationship, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []domain.Relationship
	for _, r := range m.rels {
		if r.ConnectionID == connectionID && r.Status == domain.RelationshipStatusApproved {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *MockRelationshipStore) GetByScanConnection(ctx context.Context, scanID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	connID, ok := m.connIDByScan[scanID]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return connID, nil
}

type MockOrphanStore struct {
	mu      sync.RWMutex
	orphans []domain.Orphan
}

func NewMockOrphanStore() *MockOrphanStore {
	return &MockOrphanStore{}
}

func (m *MockOrphanStore) DeleteByConnection(ctx context.Context, connectionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orphans = nil
	return nil
}

func (m *MockOrphanStore) CreateBatch(ctx context.Context, orphans []domain.Orphan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orphans = append(m.orphans, orphans...)
	return nil
}

func (m *MockOrphanStore) ListByConnection(ctx context.Context, connectionID string) ([]domain.Orphan, error) {
	orphans, _, err := m.ListByConnectionPaginated(ctx, connectionID, 0, 0)
	return orphans, err
}

func (m *MockOrphanStore) ListByConnectionPaginated(ctx context.Context, connectionID string, offset, limit int) ([]domain.Orphan, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	total := int64(len(m.orphans))
	if offset >= len(m.orphans) {
		return []domain.Orphan{}, total, nil
	}
	result := m.orphans
	if limit > 0 && offset+limit < len(result) {
		result = result[offset : offset+limit]
	} else {
		result = result[offset:]
	}
	return result, total, nil
}
