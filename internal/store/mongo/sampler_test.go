package mongo

import (
	"math"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestBsonTypeName(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{nil, "null"},
		{"hello", "string"},
		{int32(42), "int"},
		{int64(42), "long"},
		{float64(3.14), "double"},
		{true, "boolean"},
		{primitive.NewObjectID(), "objectId"},
		{primitive.DateTime(1234567890), "date"},
		{bson.M{"key": "val"}, "object"},
		{primitive.A{"a", "b"}, "array"},
	}

	for _, tt := range tests {
		got := bsonTypeName(tt.input)
		if got != tt.want {
			t.Errorf("bsonTypeName(%T) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractFields(t *testing.T) {
	oid := primitive.NewObjectID()
	docs := []bson.M{
		{"_id": oid, "name": "Alice", "age": int32(30), "active": true},
		{"_id": primitive.NewObjectID(), "name": "Bob", "age": int32(25)},
		{"_id": primitive.NewObjectID(), "name": "Carol"},
	}

	fields := extractFields(docs)

	fieldMap := make(map[string]FieldInfo)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	if len(fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(fields))
	}

	if f, ok := fieldMap["name"]; !ok {
		t.Error("missing 'name' field")
	} else {
		if f.Type != "string" {
			t.Errorf("name type = %q, want string", f.Type)
		}
		if f.Percentage != 100.0 {
			t.Errorf("name percentage = %f, want 100", f.Percentage)
		}
		if len(f.Values) != 3 {
			t.Errorf("name values count = %d, want 3", len(f.Values))
		}
	}

	if f, ok := fieldMap["age"]; !ok {
		t.Error("missing 'age' field")
	} else {
		if f.Type != "int" {
			t.Errorf("age type = %q, want int", f.Type)
		}
		if !approxEqual(f.Percentage, 100.0*2/3, 0.01) {
			t.Errorf("age percentage = %f, want %f", f.Percentage, 100.0*2/3)
		}
	}

	if f, ok := fieldMap["active"]; !ok {
		t.Error("missing 'active' field")
	} else {
		if !approxEqual(f.Percentage, 100.0/3, 0.01) {
			t.Errorf("active percentage = %f, want %f", f.Percentage, 100.0/3)
		}
	}
}

func TestExtractFields_SparseFields(t *testing.T) {
	docs := []bson.M{
		{"_id": "1", "a": "val"},
		{"_id": "2", "b": "val"},
		{"_id": "3", "a": "val", "b": "val"},
	}

	fields := extractFields(docs)
	fieldMap := make(map[string]FieldInfo)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	if !approxEqual(fieldMap["a"].Percentage, 100.0*2/3, 0.01) {
		t.Errorf("a percentage = %f", fieldMap["a"].Percentage)
	}
	if !approxEqual(fieldMap["b"].Percentage, 100.0*2/3, 0.01) {
		t.Errorf("b percentage = %f", fieldMap["b"].Percentage)
	}
}

func TestExtractFields_EmptyDocs(t *testing.T) {
	fields := extractFields([]bson.M{})
	if len(fields) != 0 {
		t.Errorf("expected 0 fields for empty docs, got %d", len(fields))
	}
}

func TestExtractFields_MixedTypes(t *testing.T) {
	docs := []bson.M{
		{"_id": "1", "field": "string"},
		{"_id": "2", "field": int32(42)},
		{"_id": "3", "field": "another"},
	}

	fields := extractFields(docs)
	for _, f := range fields {
		if f.Name == "field" {
			if f.Type != "string" {
				t.Errorf("dominant type should be string, got %q", f.Type)
			}
		}
	}
}

func TestExtractFields_ValueCap(t *testing.T) {
	docs := make([]bson.M, 300)
	for i := range docs {
		docs[i] = bson.M{"_id": i, "val": i}
	}

	fields := extractFields(docs)
	for _, f := range fields {
		if f.Name == "val" {
			if len(f.Values) > maxSampleValuesPerField {
				t.Errorf("values should be capped at %d, got %d", maxSampleValuesPerField, len(f.Values))
			}
		}
	}
}
