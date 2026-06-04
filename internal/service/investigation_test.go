package service

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
)

func TestFlattenTree(t *testing.T) {
	tree := &domain.InvestigationNode{
		Collection: "payments",
		ID:         "pay_1",
		Document:   bson.M{"_id": "pay_1"},
		Children: []*domain.InvestigationNode{
			{
				Collection: "orders",
				ID:         "ord_1",
				Document:   bson.M{"_id": "ord_1"},
				Children: []*domain.InvestigationNode{
					{
						Collection: "users",
						ID:         "usr_1",
						Document:   bson.M{"_id": "usr_1"},
					},
				},
			},
		},
	}

	var flat []domain.FlatDocument
	flattenTree(tree, &flat)

	if len(flat) != 3 {
		t.Fatalf("expected 3 flat docs, got %d", len(flat))
	}

	expected := []struct {
		coll string
		id   string
	}{
		{"payments", "pay_1"},
		{"orders", "ord_1"},
		{"users", "usr_1"},
	}

	for i, e := range expected {
		if flat[i].Collection != e.coll || flat[i].ID != e.id {
			t.Errorf("flat[%d] = %s/%s, want %s/%s", i, flat[i].Collection, flat[i].ID, e.coll, e.id)
		}
	}
}

func TestFlattenTree_Single(t *testing.T) {
	tree := &domain.InvestigationNode{
		Collection: "users",
		ID:         "usr_1",
		Document:   bson.M{"_id": "usr_1"},
	}

	var flat []domain.FlatDocument
	flattenTree(tree, &flat)

	if len(flat) != 1 {
		t.Errorf("expected 1 doc, got %d", len(flat))
	}
}

func TestFlattenTree_Empty(t *testing.T) {
	tree := &domain.InvestigationNode{
		Collection: "users",
		ID:         "usr_1",
		Document:   bson.M{"_id": "usr_1"},
		Children:   []*domain.InvestigationNode{},
	}

	var flat []domain.FlatDocument
	flattenTree(tree, &flat)

	if len(flat) != 1 {
		t.Errorf("expected 1 doc, got %d", len(flat))
	}
}

func TestIDToString(t *testing.T) {
	oid := primitive.NewObjectID()

	tests := []struct {
		input any
		want  string
	}{
		{nil, ""},
		{"abc", "abc"},
		{oid, oid.Hex()},
		{42, "42"},
	}

	for _, tt := range tests {
		got := idToString(tt.input)
		if got != tt.want {
			t.Errorf("idToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToBSONValue(t *testing.T) {
	oidHex := "507f1f77bcf86cd799439011"
	oid, _ := primitive.ObjectIDFromHex(oidHex)

	result := toBSONValue(oidHex)
	if o, ok := result.(primitive.ObjectID); !ok || o != oid {
		t.Errorf("toBSONValue(24-char hex) should convert to ObjectID")
	}

	result = toBSONValue("short")
	if result != "short" {
		t.Errorf("toBSONValue(short string) = %v, want short", result)
	}

	result = toBSONValue(42)
	if result != 42 {
		t.Errorf("toBSONValue(42) = %v, want 42", result)
	}

	result = toBSONValue(map[string]any{"$oid": oidHex})
	if o, ok := result.(primitive.ObjectID); !ok || o != oid {
		t.Errorf("toBSONValue($oid map) should convert to ObjectID")
	}

	result = toBSONValue(map[string]any{"foo": "bar"})
	m, ok := result.(map[string]any)
	if !ok || m["foo"] != "bar" {
		t.Errorf("toBSONValue(map without $oid) should pass through")
	}
}

func TestVisitKey(t *testing.T) {
	v1 := visitKey{collection: "users", id: "123"}
	v2 := visitKey{collection: "users", id: "123"}
	v3 := visitKey{collection: "orders", id: "123"}

	if v1 != v2 {
		t.Error("identical visitKeys should be equal")
	}
	if v1 == v3 {
		t.Error("different visitKeys should not be equal")
	}
}
