package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type FieldInfo struct {
	Name       string
	Type       string
	Values     []any
	Percentage float64
}

type CollectionSample struct {
	CollectionName string
	DocumentCount  int
	Fields         []FieldInfo
}

func SampleCollection(ctx context.Context, db *mongo.Database, collectionName string, sampleSize int) (*CollectionSample, error) {
	coll := db.Collection(collectionName)

	count, err := coll.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("count documents: %w", err)
	}

	limit := int64(sampleSize)
	if count < limit {
		limit = count
	}

	if limit == 0 {
		return &CollectionSample{
			CollectionName: collectionName,
			DocumentCount:  0,
			Fields:         []FieldInfo{},
		}, nil
	}

	cursor, err := coll.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode documents: %w", err)
	}

	if len(docs) == 0 {
		return &CollectionSample{
			CollectionName: collectionName,
			DocumentCount:  int(count),
			Fields:         []FieldInfo{},
		}, nil
	}

	fieldStats := extractFields(docs)

	return &CollectionSample{
		CollectionName: collectionName,
		DocumentCount:  int(count),
		Fields:         fieldStats,
	}, nil
}

type fieldAccum struct {
	name        string
	occurrences int
	types       map[string]int
	values      []any
}

func extractFields(docs []bson.M) []FieldInfo {
	fieldMap := make(map[string]*fieldAccum)
	for _, doc := range docs {
		for key, val := range doc {
			if _, ok := fieldMap[key]; !ok {
				fieldMap[key] = &fieldAccum{
					name:   key,
					types:  make(map[string]int),
					values: make([]any, 0, 10),
				}
			}
			f := fieldMap[key]
			f.occurrences++
			typeName := bsonTypeName(val)
			f.types[typeName]++
			if len(f.values) < 10 {
				f.values = append(f.values, val)
			}
		}
	}

	fields := make([]FieldInfo, 0, len(fieldMap))
	total := len(docs)
	for _, f := range fieldMap {
		dominantType := "mixed"
		maxCount := 0
		for t, c := range f.types {
			if c > maxCount {
				maxCount = c
				dominantType = t
			}
		}
		fields = append(fields, FieldInfo{
			Name:       f.name,
			Type:       dominantType,
			Values:     f.values,
			Percentage: float64(f.occurrences) / float64(total) * 100,
		})
	}
	return fields
}

func bsonTypeName(v any) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return "string"
	case int32:
		return "int"
	case int64:
		return "long"
	case float64:
		return "double"
	case bool:
		return "boolean"
	case primitive.ObjectID:
		return "objectId"
	case primitive.DateTime:
		return "date"
	case bson.M:
		return "object"
	case primitive.A:
		return "array"
	default:
		_ = val
		return fmt.Sprintf("%T", v)
	}
}
