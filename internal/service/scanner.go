package service

import (
	"context"
	"fmt"
	"log"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/scanner"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

type ScannerService struct {
	scanStore    *pg.ScanStore
	connStore    *pg.ConnectionStore
}

func NewScannerService(scanStore *pg.ScanStore, connStore *pg.ConnectionStore) *ScannerService {
	return &ScannerService{scanStore: scanStore, connStore: connStore}
}

func (s *ScannerService) StartScan(ctx context.Context, connectionID string, sampleSize int) (*domain.Scan, error) {
	conn, err := s.connStore.Get(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	if conn.Database == "" {
		return nil, fmt.Errorf("no database selected for this connection")
	}

	scan := &domain.Scan{
		ConnectionID: connectionID,
		Status:       domain.ScanStatusPending,
		SampleSize:   sampleSize,
	}

	if err := s.scanStore.Create(ctx, scan); err != nil {
		return nil, fmt.Errorf("create scan: %w", err)
	}

	return scan, nil
}

func (s *ScannerService) RunScan(ctx context.Context, scanID string) error {
	scan, err := s.scanStore.Get(ctx, scanID)
	if err != nil {
		return fmt.Errorf("get scan: %w", err)
	}

	conn, err := s.connStore.Get(ctx, scan.ConnectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	if err := s.scanStore.UpdateStatus(ctx, scanID, domain.ScanStatusRunning, nil); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	mongoConn, err := mongostore.NewConnector(ctx, conn.ConnectionString)
	if err != nil {
		errMsg := err.Error()
		_ = s.scanStore.UpdateStatus(ctx, scanID, domain.ScanStatusFailed, &errMsg)
		return fmt.Errorf("connect to mongo: %w", err)
	}
	defer mongoConn.Close(ctx)

	collections, err := mongoConn.ListCollections(ctx, conn.Database)
	if err != nil {
		errMsg := err.Error()
		_ = s.scanStore.UpdateStatus(ctx, scanID, domain.ScanStatusFailed, &errMsg)
		return fmt.Errorf("list collections: %w", err)
	}

	db := mongoConn.Database(conn.Database)

	for _, collName := range collections {
		log.Printf("scanning collection: %s", collName)

		sample, err := mongostore.SampleCollection(ctx, db, collName, scan.SampleSize)
		if err != nil {
			log.Printf("warning: failed to sample %s: %v", collName, err)
			continue
		}

		for _, fi := range sample.Fields {
			candidate := scanner.IsCandidateField(fi.Name, fi.Type, fi.Values)

			f := &domain.CollectionField{
				ScanID:         scanID,
				CollectionName: collName,
				FieldName:      fi.Name,
				FieldType:      fi.Type,
				SampleValues:   fi.Values,
				IsCandidate:    candidate.IsCandidate,
				DocumentCount:  sample.DocumentCount,
			}
			if candidate.IsCandidate {
				f.CandidateReason = &candidate.Reason
			}

			if err := s.scanStore.InsertField(ctx, f); err != nil {
				log.Printf("warning: failed to insert field %s.%s: %v", collName, fi.Name, err)
			}
		}
	}

	if err := s.scanStore.UpdateStatus(ctx, scanID, domain.ScanStatusCompleted, nil); err != nil {
		return fmt.Errorf("update status completed: %w", err)
	}

	log.Printf("scan %s completed", scanID)
	return nil
}

func (s *ScannerService) GetScan(ctx context.Context, scanID string) (*domain.Scan, error) {
	return s.scanStore.Get(ctx, scanID)
}

func (s *ScannerService) ListScans(ctx context.Context, connectionID string) ([]domain.Scan, error) {
	scans, _, err := s.scanStore.ListByConnectionPaginated(ctx, connectionID, 0, 0)
	return scans, err
}

func (s *ScannerService) ListScansPaginated(ctx context.Context, connectionID string, offset, limit int) ([]domain.Scan, int64, error) {
	return s.scanStore.ListByConnectionPaginated(ctx, connectionID, offset, limit)
}

func (s *ScannerService) GetScanFields(ctx context.Context, scanID string) ([]domain.CollectionField, error) {
	fields, _, err := s.scanStore.GetFieldsByScanPaginated(ctx, scanID, 0, 0)
	return fields, err
}

func (s *ScannerService) GetScanFieldsPaginated(ctx context.Context, scanID string, offset, limit int) ([]domain.CollectionField, int64, error) {
	return s.scanStore.GetFieldsByScanPaginated(ctx, scanID, offset, limit)
}

func (s *ScannerService) GetCandidateFields(ctx context.Context, scanID string) ([]domain.CollectionField, error) {
	return s.scanStore.GetCandidateFields(ctx, scanID)
}
