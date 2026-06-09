package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.M{"size": limit}}},
	}
	cursor, err := coll.Aggregate(ctx, pipeline)
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

const maxSampleValuesPerField = 200

const maxNestingDepth = 3

type fieldAccum struct {
	name        string
	occurrences int
	types       map[string]int
	values      []any
}

func extractFields(docs []bson.M) []FieldInfo {
	fieldMap := make(map[string]*fieldAccum)
	for _, doc := range docs {
		extractDoc("", doc, fieldMap, 0)
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

func extractDoc(prefix string, doc bson.M, fieldMap map[string]*fieldAccum, depth int) {
	if depth >= maxNestingDepth {
		return
	}
	for key, val := range doc {
		fullName := key
		if prefix != "" {
			fullName = prefix + "." + key
		}

		switch v := val.(type) {
		case bson.M:
			typeName := bsonTypeName(v)
			recordField(fieldMap, fullName, v, typeName)
			extractDoc(fullName, v, fieldMap, depth+1)
		case primitive.A:
			leafType := arrayLeafType(v)
			if leafType == "object" {
				recordField(fieldMap, fullName, v, "array")
				extractArrayObjects(fullName, v, fieldMap, depth+1)
			} else {
				sampleArrayScalars(fieldMap, fullName, v, leafType)
			}
		default:
			typeName := bsonTypeName(v)
			recordField(fieldMap, fullName, v, typeName)
		}
	}
}

func extractArrayObjects(prefix string, arr primitive.A, fieldMap map[string]*fieldAccum, depth int) {
	if depth >= maxNestingDepth {
		return
	}
	for _, elem := range arr {
		if obj, ok := elem.(bson.M); ok {
			extractDoc(prefix, obj, fieldMap, depth)
		}
	}
}

func sampleArrayScalars(fieldMap map[string]*fieldAccum, name string, arr primitive.A, leafType string) {
	if _, ok := fieldMap[name]; !ok {
		fieldMap[name] = &fieldAccum{
			name:   name,
			types:  make(map[string]int),
			values: make([]any, 0, maxSampleValuesPerField),
		}
	}
	f := fieldMap[name]
	f.occurrences++
	f.types["array"]++
	count := 0
	for _, elem := range arr {
		if count >= maxSampleValuesPerField {
			break
		}
		f.values = append(f.values, elem)
		count++
	}
}

func recordField(fieldMap map[string]*fieldAccum, name string, val any, typeName string) {
	if _, ok := fieldMap[name]; !ok {
		fieldMap[name] = &fieldAccum{
			name:   name,
			types:  make(map[string]int),
			values: make([]any, 0, maxSampleValuesPerField),
		}
	}
	f := fieldMap[name]
	f.occurrences++
	f.types[typeName]++
	if len(f.values) < maxSampleValuesPerField {
		f.values = append(f.values, val)
	}
}

func arrayLeafType(arr primitive.A) string {
	seen := map[string]int{}
	for _, elem := range arr {
		seen[bsonTypeName(elem)]++
	}
	maxType := ""
	maxCount := 0
	for t, c := range seen {
		if c > maxCount {
			maxCount = c
			maxType = t
		}
	}
	return maxType
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
