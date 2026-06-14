package service

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongodriver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

type OrphanService struct {
	connStore     domain.ConnectionReader
	relStore      domain.RelationshipReaderWriter
	orphanStore   domain.OrphanReaderWriter
	investigation *InvestigationService
}

func NewOrphanService(connStore *pg.ConnectionStore, relStore *pg.RelationshipStore, orphanStore *pg.OrphanStore, investigation *InvestigationService) *OrphanService {
	return &OrphanService{connStore: connStore, relStore: relStore, orphanStore: orphanStore, investigation: investigation}
}

func (s *OrphanService) DetectOrphans(ctx context.Context, connectionID string) ([]domain.Orphan, error) {
	conn, err := s.connStore.Get(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}

	rels, _, err := s.relStore.ListPaginated(ctx, connectionID, nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get approved relationships: %w", err)
	}

	filteredRels := make([]domain.Relationship, 0)
	for _, rel := range rels {
		if rel.Status == domain.RelationshipStatusApproved {
			filteredRels = append(filteredRels, rel)
		}
	}
	rels = filteredRels

	if len(rels) == 0 {
		return nil, nil
	}

	mongoConn, err := mongostore.NewConnector(ctx, conn.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("connect to mongo: %w", err)
	}
	defer mongoConn.Close(ctx)

	db := mongoConn.Database(conn.Database)

	if err := s.orphanStore.DeleteByConnection(ctx, connectionID); err != nil {
		return nil, fmt.Errorf("clear old orphans: %w", err)
	}

	var allOrphans []domain.Orphan

	for _, rel := range rels {
		orphans, err := s.detectForRelationship(ctx, db, rel)
		if err != nil {
			log.Printf("warning: orphan detection failed for %s.%s → %s.%s: %v",
				rel.SourceCollection, rel.SourceField, rel.TargetCollection, rel.TargetField, err)
			continue
		}
		allOrphans = append(allOrphans, orphans...)
	}

	if len(allOrphans) > 0 {
		if err := s.orphanStore.CreateBatch(ctx, allOrphans); err != nil {
			return nil, fmt.Errorf("store orphans: %w", err)
		}
	}

	log.Printf("orphan detection complete: %d orphans found for connection %s", len(allOrphans), connectionID)

	return allOrphans, nil
}

func (s *OrphanService) detectForRelationship(ctx context.Context, db *mongodriver.Database, rel domain.Relationship) ([]domain.Orphan, error) {
	pipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":   "$" + rel.SourceField,
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := db.Collection(rel.SourceCollection).Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate source values: %w", err)
	}
	defer cursor.Close(ctx)

	var sourceValues []struct {
		ID    any `bson:"_id"`
		Count int `bson:"count"`
	}
	if err := cursor.All(ctx, &sourceValues); err != nil {
		return nil, fmt.Errorf("decode source values: %w", err)
	}

	var candidateValues []any
	for _, sv := range sourceValues {
		if sv.ID == nil {
			continue
		}
		candidateValues = append(candidateValues, sv.ID)
	}

	if len(candidateValues) == 0 {
		return nil, nil
	}

	bsonValues := make([]any, len(candidateValues))
	for i, v := range candidateValues {
		bsonValues[i] = ensureBSONValue(v)
	}

	cursor2, err := db.Collection(rel.TargetCollection).Find(ctx, bson.M{
		rel.TargetField: bson.M{"$in": bsonValues},
	}, options.Find().SetProjection(bson.M{rel.TargetField: 1}))
	if err != nil {
		return nil, fmt.Errorf("find existing targets: %w", err)
	}
	defer cursor2.Close(ctx)

	var existing []bson.M
	if err := cursor2.All(ctx, &existing); err != nil {
		return nil, fmt.Errorf("decode existing: %w", err)
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, doc := range existing {
		key := idValToString(doc[rel.TargetField])
		if key != "" {
			existingSet[key] = struct{}{}
		}
	}

	var orphans []domain.Orphan
	for _, sv := range sourceValues {
		if sv.ID == nil {
			continue
		}
		key := idValToString(sv.ID)
		if key == "" {
			continue
		}
		if _, found := existingSet[key]; !found {
			orphans = append(orphans, domain.Orphan{
				RelationshipID:   rel.ID,
				SourceCollection: rel.SourceCollection,
				SourceField:      rel.SourceField,
				MissingValue:     key,
			})
		}
	}

	log.Printf("orphan check %s.%s → %s.%s: %d unique values, %d existing, %d orphans",
		rel.SourceCollection, rel.SourceField, rel.TargetCollection, rel.TargetField,
		len(candidateValues), len(existingSet), len(orphans))

	return orphans, nil
}

func ensureBSONValue(v any) any {
	switch val := v.(type) {
	case string:
		if len(val) == 24 {
			if objID, err := primitive.ObjectIDFromHex(val); err == nil {
				return objID
			}
		}
		return val
	case primitive.ObjectID:
		return val
	default:
		return v
	}
}

func idValToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case primitive.ObjectID:
		return val.Hex()
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (s *OrphanService) ListOrphans(ctx context.Context, connectionID string) ([]domain.Orphan, error) {
	orphans, _, err := s.orphanStore.ListByConnectionPaginated(ctx, connectionID, 0, 0)
	return orphans, err
}

func (s *OrphanService) ListOrphansPaginated(ctx context.Context, connectionID string, offset, limit int) ([]domain.Orphan, int64, error) {
	return s.orphanStore.ListByConnectionPaginated(ctx, connectionID, offset, limit)
}

func (s *OrphanService) InvestigateOrphan(ctx context.Context, orphanID string) (*domain.InvestigateResult, error) {
	orphan, err := s.orphanStore.Get(ctx, orphanID)
	if err != nil {
		return nil, fmt.Errorf("get orphan: %w", err)
	}

	rel, err := s.relStore.Get(ctx, orphan.RelationshipID)
	if err != nil {
		return nil, fmt.Errorf("get relationship: %w", err)
	}

	return s.investigation.Investigate(ctx, rel.ConnectionID, orphan.MissingValue)
}
