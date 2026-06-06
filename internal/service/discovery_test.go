package service

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUniqueNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		input  []any
		expect int
	}{
		{"nil input", nil, 0},
		{"empty input", []any{}, 0},
		{"all empty strings", []any{"", "", ""}, 0},
		{"all nil", []any{nil, nil}, 0},
		{"deduplicates", []any{"a", "a", "b", "b"}, 2},
		{"mixed with empty", []any{"a", "", "b", nil}, 2},
		{"objectids", []any{
			map[string]any{"$oid": "507f1f77bcf86cd799439011"},
			map[string]any{"$oid": "507f1f77bcf86cd799439011"},
			map[string]any{"$oid": "507f1f77bcf86cd799439012"},
		}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uniqueNonEmpty(tt.input)
			if len(result) != tt.expect {
				t.Errorf("uniqueNonEmpty(%v) = %d results, want %d", tt.input, len(result), tt.expect)
			}
		})
	}
}

func TestToBSONValues(t *testing.T) {
	oidHex := "507f1f77bcf86cd799439011"
	oid, _ := primitive.ObjectIDFromHex(oidHex)

	tests := []struct {
		name  string
		input []any
		check func([]any) bool
	}{
		{
			"converts $oid map to ObjectID",
			[]any{map[string]any{"$oid": oidHex}},
			func(result []any) bool {
				if len(result) != 1 {
					return false
				}
				o, ok := result[0].(primitive.ObjectID)
				return ok && o == oid
			},
		},
		{
			"converts 24-char hex string to ObjectID",
			[]any{oidHex},
			func(result []any) bool {
				if len(result) != 1 {
					return false
				}
				o, ok := result[0].(primitive.ObjectID)
				return ok && o == oid
			},
		},
		{
			"leaves short strings as-is",
			[]any{"short"},
			func(result []any) bool {
				return len(result) == 1 && result[0] == "short"
			},
		},
		{
			"leaves non-hex 24-char strings as-is",
			[]any{"zzzzzzzzzzzzzzzzzzzzzzzz"},
			func(result []any) bool {
				return len(result) == 1 && result[0] == "zzzzzzzzzzzzzzzzzzzzzzzz"
			},
		},
		{
			"passes through non-map non-string values",
			[]any{42, true},
			func(result []any) bool {
				return len(result) == 2 && result[0] == 42 && result[1] == true
			},
		},
		{
			"map without $oid key passes through",
			[]any{map[string]any{"foo": "bar"}},
			func(result []any) bool {
				if len(result) != 1 {
					return false
				}
				m, ok := result[0].(map[string]any)
				return ok && m["foo"] == "bar"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toBSONValues(tt.input)
			if !tt.check(result) {
				t.Errorf("toBSONValues(%v) check failed, got %v", tt.input, result)
			}
		})
	}
}

func TestValueToString(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{nil, ""},
		{"hello", "hello"},
		{42, "42"},
		{map[string]any{"$oid": "abc123"}, "abc123"},
		{map[string]any{"$date": "2024-01-01"}, "2024-01-01"},
		{map[string]any{"unknown": "val"}, ""},
	}

	for _, tt := range tests {
		got := valueToString(tt.input)
		if got != tt.want {
			t.Errorf("valueToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDocID(t *testing.T) {
	oid := primitive.NewObjectID()

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"bson.M with ObjectID", bson.M{"_id": oid}, oid.Hex()},
		{"map with string", map[string]any{"_id": "abc123"}, "abc123"},
		{"bson.M without _id", bson.M{}, ""},
		{"non-map", "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docID(tt.input)
			if got != tt.want {
				t.Errorf("docID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFieldValue(t *testing.T) {
	doc := bson.M{"name": "Alice", "age": 30}

	if v := fieldValue(doc, "name"); v != "Alice" {
		t.Errorf("fieldValue(name) = %v, want Alice", v)
	}
	if v := fieldValue(doc, "missing"); v != nil {
		t.Errorf("fieldValue(missing) = %v, want nil", v)
	}
	if v := fieldValue("not a map", "name"); v != nil {
		t.Errorf("fieldValue on non-map should return nil")
	}
}
