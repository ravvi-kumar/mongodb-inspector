package service

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestEnsureBSONValue(t *testing.T) {
	oidHex := "507f1f77bcf86cd799439011"
	oid, _ := primitive.ObjectIDFromHex(oidHex)

	result := ensureBSONValue(oidHex)
	if o, ok := result.(primitive.ObjectID); !ok || o != oid {
		t.Errorf("ensureBSONValue(24-char hex string) should return ObjectID")
	}

	result = ensureBSONValue("short")
	if result != "short" {
		t.Errorf("ensureBSONValue(short string) = %v, want short", result)
	}

	result = ensureBSONValue("zzzzzzzzzzzzzzzzzzzzzzzz")
	if result != "zzzzzzzzzzzzzzzzzzzzzzzz" {
		t.Errorf("ensureBSONValue(non-hex 24-char) should pass through")
	}

	result = ensureBSONValue(oid)
	if o, ok := result.(primitive.ObjectID); !ok || o != oid {
		t.Errorf("ensureBSONValue(ObjectID) should return same ObjectID")
	}

	result = ensureBSONValue(42)
	if result != 42 {
		t.Errorf("ensureBSONValue(42) = %v, want 42", result)
	}
}

func TestIDValToString(t *testing.T) {
	oid := primitive.NewObjectID()

	tests := []struct {
		input any
		want  string
	}{
		{nil, ""},
		{"abc123", "abc123"},
		{oid, oid.Hex()},
		{42, "42"},
	}

	for _, tt := range tests {
		got := idValToString(tt.input)
		if got != tt.want {
			t.Errorf("idValToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
