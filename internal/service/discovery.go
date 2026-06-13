package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
	"github.com/ravikumar/mongodb-inspector/internal/scorer"
	mongostore "github.com/ravikumar/mongodb-inspector/internal/store/mongo"
	"github.com/ravikumar/mongodb-inspector/internal/store/pg"
)

const confidenceThreshold = 0.2
const autoApproveThreshold = 0.7
const uniquenessThreshold = 0.8

type DiscoveryService struct {
	scanStore         domain.ScanReader
	relStore          domain.RelationshipReaderWriter
	connStore         domain.ConnectionReader
	scorer            *scorer.Scorer
	batchSize         int
	batchDelay        time.Duration
}

func NewDiscoveryService(scanStore *pg.ScanStore, relStore *pg.RelationshipStore, connStore *pg.ConnectionStore) *DiscoveryService {
	return &DiscoveryService{
		scanStore:  scanStore,
		relStore:   relStore,
		connStore:  connStore,
		scorer:     scorer.NewScorer(),
		batchSize:  50,
		batchDelay: 100 * time.Millisecond,
	}
}

func (s *DiscoveryService) SetRateLimit(batchSize int, delayMs int) {
	if batchSize > 0 {
		s.batchSize = batchSize
	}
	if delayMs >= 0 {
		s.batchDelay = time.Duration(delayMs) * time.Millisecond
	}
}

type uniqueField struct {
	Collection    string
	FieldName     string
	FieldType     string
	UniquenessRatio float64
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

	if len(candidates) == 0 {
		return nil
	}

	mongoConn, err := mongostore.NewConnector(ctx, conn.ConnectionString)
	if err != nil {
		return fmt.Errorf("connect to mongo: %w", err)
	}
	defer mongoConn.Close(ctx)

	db := mongoConn.Database(conn.Database)

	uniqueTargets, err := uniqueFields(ctx, db, allFields)
	if err != nil {
		return fmt.Errorf("identify unique fields: %w", err)
	}

	log.Printf("discovery: %d candidates, %d unique target fields across collections", len(candidates), len(uniqueTargets))

	queryCount := 0
	for _, candidate := range candidates {
		uniqueVals := uniqueNonEmpty(candidate.SampleValues)
		if len(uniqueVals) == 0 {
			continue
		}

		candidateReason := ""
		if candidate.CandidateReason != nil {
			candidateReason = *candidate.CandidateReason
		}

		for _, target := range uniqueTargets {
			if target.Collection == candidate.CollectionName && target.FieldName == candidate.FieldName {
				continue
			}

			matched, sampled := queryMatchCount(ctx, db, target.Collection, target.FieldName, uniqueVals)
			if sampled == 0 {
				continue
			}

			queryCount++
			if s.batchSize > 0 && queryCount%s.batchSize == 0 {
				time.Sleep(s.batchDelay)
			}

			params := scorer.ScoreParams{
				Matched:               matched,
				Sampled:               sampled,
				SourceField:           candidate.FieldName,
				SourceType:            candidate.FieldType,
				TargetCollection:      target.Collection,
				TargetField:           target.FieldName,
				TargetType:            target.FieldType,
				CandidateReason:       candidateReason,
				TargetUniquenessRatio: target.UniquenessRatio,
			}

			result := s.scorer.Score(params)

			log.Printf("match %s.%s → %s.%s: %d/%d map=%.0f%% comp=%.1f%%",
				candidate.CollectionName, candidate.FieldName, target.Collection, target.FieldName,
				matched, sampled, float64(matched)/float64(sampled)*100, result.Confidence*100)

			if result.Confidence < confidenceThreshold {
				continue
			}

			status := domain.RelationshipStatusSuggested
			if result.Confidence >= autoApproveThreshold {
				status = domain.RelationshipStatusApproved
			}

			rel := &domain.Relationship{
				ConnectionID:     scan.ConnectionID,
				SourceCollection: candidate.CollectionName,
				SourceField:      candidate.FieldName,
				TargetCollection: target.Collection,
				TargetField:      target.FieldName,
				Confidence:       math.Round(result.Confidence*1000) / 1000,
				MatchedValues:    matched,
				SampledValues:    sampled,
				Status:           status,
				Explanation:      s.scorer.FormatExplanation(result),
			}

			created, err := s.relStore.CreateOrSkip(ctx, rel)
			if err != nil {
				log.Printf("warning: failed to create relationship %s.%s → %s.%s: %v",
					candidate.CollectionName, candidate.FieldName, target.Collection, target.FieldName, err)
			} else if created {
				log.Printf("discovered: %s.%s → %s.%s (%.1f%%, %d/%d) [%s]",
					candidate.CollectionName, candidate.FieldName, target.Collection, target.FieldName,
					result.Confidence*100, matched, sampled, status)
			} else {
				log.Printf("skipped duplicate: %s.%s → %s.%s",
					candidate.CollectionName, candidate.FieldName, target.Collection, target.FieldName)
			}
		}
	}

	return nil
}

func uniqueFields(ctx context.Context, db *mongo.Database, allFields []domain.CollectionField) ([]uniqueField, error) {
	collectionFields := make(map[string][]domain.CollectionField)
	for _, f := range allFields {
		collectionFields[f.CollectionName] = append(collectionFields[f.CollectionName], f)
	}

	var targets []uniqueField

	for collName, fields := range collectionFields {
		estCount, err := db.Collection(collName).EstimatedDocumentCount(ctx)
		if err != nil {
			log.Printf("warning: could not estimate doc count for %s: %v", collName, err)
			continue
		}

		for _, f := range fields {
			if f.FieldName == "_id" {
				targets = append(targets, uniqueField{
					Collection:      collName,
					FieldName:       "_id",
					FieldType:       f.FieldType,
					UniquenessRatio: 1.0,
				})
				continue
			}

			ratio := estimateUniqueness(ctx, db, collName, f.FieldName, estCount)
			if ratio >= uniquenessThreshold {
				targets = append(targets, uniqueField{
					Collection:      collName,
					FieldName:       f.FieldName,
					FieldType:       f.FieldType,
					UniquenessRatio: ratio,
				})
			}
		}
	}

	return targets, nil
}

func estimateUniqueness(ctx context.Context, db *mongo.Database, collName, fieldName string, estCount int64) float64 {
	if estCount <= 0 {
		return 0
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Cheaper path: use sampled aggregation instead of full Distinct for large collections
	if estCount > 100000 {
		return estimateUniquenessSampled(ctx, db, collName, fieldName, estCount)
	}

	// For small collections, Distinct is safe
	if fieldHasIndex(ctx, db, collName, fieldName) {
		distinct, err := db.Collection(collName).Distinct(ctx, fieldName, bson.M{})
		if err != nil {
			return 0
		}
		distinctCount := int64(len(distinct))
		if distinctCount == 0 {
			return 0
		}
		ratio := float64(distinctCount) / float64(estCount)
		if ratio > 1.0 {
			ratio = 1.0
		}
		return ratio
	}

	return estimateUniquenessSampled(ctx, db, collName, fieldName, estCount)
}

func estimateUniquenessSampled(ctx context.Context, db *mongo.Database, collName, fieldName string, estCount int64) float64 {
	sampleSize := int64(10000)
	if estCount < sampleSize {
		sampleSize = estCount
	}

	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.M{"size": sampleSize}}},
		{{Key: "$group", Value: bson.M{
			"_id": "$" + fieldName,
		}}},
		{{Key: "$count", Value: "distinct_count"}},
	}

	cursor, err := db.Collection(collName).Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("warning: sampled uniqueness aggregation failed on %s.%s: %v", collName, fieldName, err)
		return 0
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return 0
	}

	var result struct {
		DistinctCount int64 `bson:"distinct_count"`
	}
	if err := cursor.Decode(&result); err != nil {
		return 0
	}

	if result.DistinctCount == 0 {
		return 0
	}

	ratio := float64(result.DistinctCount) / float64(sampleSize)
	if ratio > 1.0 {
		ratio = 1.0
	}
	return ratio
}

func fieldHasIndex(ctx context.Context, db *mongo.Database, collName, fieldName string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cursor, err := db.Collection(collName).Indexes().List(ctx)
	if err != nil {
		return false
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var idx struct {
			Key bson.M `bson:"key"`
		}
		if err := cursor.Decode(&idx); err != nil {
			continue
		}
		if _, ok := idx.Key[fieldName]; ok {
			return true
		}
	}
	return false
}

func queryMatchCount(ctx context.Context, db *mongo.Database, collection string, field string, values []any) (matched int, sampled int) {
	if len(values) == 0 {
		return 0, 0
	}

	bsonValues := toBSONValues(values)

	filter := bson.M{field: bson.M{"$in": bsonValues}}

	count, err := db.Collection(collection).CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("warning: count query failed on %s.%s: %v", collection, field, err)
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
