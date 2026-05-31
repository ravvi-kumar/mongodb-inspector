package service

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

const confidenceThreshold = 0.2
const autoApproveThreshold = 0.7

type DiscoveryService struct {
	scanStore *pg.ScanStore
	relStore  *pg.RelationshipStore
	connStore *pg.ConnectionStore
}

func NewDiscoveryService(scanStore *pg.ScanStore, relStore *pg.RelationshipStore, connStore *pg.ConnectionStore) *DiscoveryService {
	return &DiscoveryService{scanStore: scanStore, relStore: relStore, connStore: connStore}
}

func (s *DiscoveryService) DiscoverRelationships(ctx context.Context, scanID string) error {
	scan, err := s.scanStore.Get(ctx, scanID)
	if err != nil {
		return fmt.Errorf("get scan: %w", err)
	}

	conn, err := s.connStore.Get(ctx, scan.ConnectionID)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}

	candidates, err := s.scanStore.GetCandidateFields(ctx, scanID)
	if err != nil {
		return fmt.Errorf("get candidates: %w", err)
	}

	allFields, err := s.scanStore.GetFieldsByScan(ctx, scanID)
	if err != nil {
		return fmt.Errorf("get all fields: %w", err)
	}

	collectionsWithID := collectionsWithIDFields(allFields)

	log.Printf("discovery: %d candidates, %d collections to check against", len(candidates), len(collectionsWithID))

	if len(candidates) == 0 {
		return nil
	}

	mongoConn, err := mongostore.NewConnector(ctx, conn.ConnectionString)
	if err != nil {
		return fmt.Errorf("connect to mongo: %w", err)
	}
	defer mongoConn.Close(ctx)

	db := mongoConn.Database(conn.Database)

	for _, candidate := range candidates {
		uniqueVals := uniqueNonEmpty(candidate.SampleValues)
		if len(uniqueVals) == 0 {
			continue
		}

		for _, targetColl := range collectionsWithID {
			if targetColl == candidate.CollectionName {
				continue
			}

			matched, sampled := queryMatchCount(ctx, db, targetColl, "_id", uniqueVals)
			if sampled == 0 {
				continue
			}

			confidence := float64(matched) / float64(sampled)

			log.Printf("match %s.%s → %s._id: %d/%d (%.1f%%)",
				candidate.CollectionName, candidate.FieldName, targetColl,
				matched, sampled, confidence*100)

			if confidence < confidenceThreshold {
				continue
			}

			status := domain.RelationshipStatusSuggested
			if confidence >= autoApproveThreshold {
				status = domain.RelationshipStatusApproved
			}

			rel := &domain.Relationship{
				ConnectionID:     scan.ConnectionID,
				SourceCollection: candidate.CollectionName,
				SourceField:      candidate.FieldName,
				TargetCollection: targetColl,
				TargetField:      "_id",
				Confidence:       confidence,
				MatchedValues:    matched,
				SampledValues:    sampled,
				Status:           status,
			}

			if err := s.relStore.Create(ctx, rel); err != nil {
				log.Printf("warning: failed to create relationship %s.%s → %s._id: %v",
					candidate.CollectionName, candidate.FieldName, targetColl, err)
			} else {
				log.Printf("discovered: %s.%s → %s._id (%.1f%%, %d/%d) [%s]",
					candidate.CollectionName, candidate.FieldName, targetColl,
					confidence*100, matched, sampled, status)
			}
		}
	}

	return nil
}

func queryMatchCount(ctx context.Context, db *mongo.Database, collection string, field string, values []any) (matched int, sampled int) {
	if len(values) == 0 {
		return 0, 0
	}

	bsonValues := toBSONValues(values)

	filter := bson.M{field: bson.M{"$in": bsonValues}}

	count, err := db.Collection(collection).CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("warning: count query failed on %s: %v", collection, err)
		return 0, len(values)
	}

	return int(count), len(values)
}

func toBSONValues(values []any) []any {
	result := make([]any, 0, len(values))
	for _, v := range values {
		switch val := v.(type) {
		case map[string]any:
			if oid, ok := val["$oid"]; ok {
				hexStr := fmt.Sprintf("%v", oid)
				objID, err := parseObjectID(hexStr)
				if err == nil {
					result = append(result, objID)
					continue
				}
			}
			result = append(result, v)
		case string:
			if len(val) == 24 {
				objID, err := parseObjectID(val)
				if err == nil {
					result = append(result, objID)
				} else {
					result = append(result, val)
				}
			} else {
				result = append(result, val)
			}
		default:
			result = append(result, v)
		}
	}
	return result
}

func parseObjectID(hex string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(hex)
}

func uniqueNonEmpty(values []any) []any {
	seen := make(map[string]struct{})
	result := make([]any, 0)
	for _, v := range values {
		key := valueToString(v)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, v)
	}
	return result
}

func collectionsWithIDFields(fields []domain.CollectionField) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, f := range fields {
		if f.FieldName == "_id" {
			if _, ok := seen[f.CollectionName]; !ok {
				seen[f.CollectionName] = struct{}{}
				result = append(result, f.CollectionName)
			}
		}
	}
	return result
}

func valueToString(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if oid, ok := val["$oid"]; ok {
			return fmt.Sprintf("%v", oid)
		}
		if date, ok := val["$date"]; ok {
			return fmt.Sprintf("%v", date)
		}
		return ""
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
