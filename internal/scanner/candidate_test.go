package scanner

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestIsCandidateField_ExcludesId(t *testing.T) {
	r := IsCandidateField("_id", "objectId", nil)
	if r.IsCandidate {
		t.Error("_id should never be a candidate")
	}
}

func TestIsCandidateField_CamelCaseIdSuffix(t *testing.T) {
	for _, name := range []string{"userId", "orderId", "paymentId", "commentId"} {
		r := IsCandidateField(name, "string", nil)
		if !r.IsCandidate {
			t.Errorf("%s should be candidate (camelCase Id suffix)", name)
		}
		if r.Reason != "field name ends with Id/_id" {
			t.Errorf("unexpected reason for %s: %s", name, r.Reason)
		}
	}
}

func TestIsCandidateField_SnakeCaseIdSuffix(t *testing.T) {
	for _, name := range []string{"user_id", "order_id", "payment_id"} {
		r := IsCandidateField(name, "string", nil)
		if !r.IsCandidate {
			t.Errorf("%s should be candidate (snake_case _id suffix)", name)
		}
	}
}

func TestIsCandidateField_RefSuffix(t *testing.T) {
	for _, name := range []string{"userRef", "orderReference", "paymentRef"} {
		r := IsCandidateField(name, "string", nil)
		if !r.IsCandidate {
			t.Errorf("%s should be candidate (Ref suffix)", name)
		}
	}
}

func TestIsCandidateField_BySuffix(t *testing.T) {
	for _, name := range []string{"createdBy", "updatedBy", "deletedBy"} {
		r := IsCandidateField(name, "string", nil)
		if !r.IsCandidate {
			t.Errorf("%s should be candidate (By suffix)", name)
		}
	}
}

func TestIsCandidateField_BySuffix_ExcludesSortBy(t *testing.T) {
	r := IsCandidateField("sortby", "string", nil)
	if r.IsCandidate {
		t.Error("sortby should not be a candidate")
	}
}

func TestIsCandidateField_CommonRefNames(t *testing.T) {
	for _, name := range []string{"customer", "owner", "author", "parent", "user", "creator", "assignee", "approver", "reviewer", "manager", "admin", "member", "reporter", "subscriber"} {
		r := IsCandidateField(name, "string", nil)
		if !r.IsCandidate {
			t.Errorf("%q should be candidate (common ref name)", name)
		}
		if r.Reason != "common reference field name" {
			t.Errorf("unexpected reason for %s: %s", name, r.Reason)
		}
	}
}

func TestIsCandidateField_ObjectIdType(t *testing.T) {
	r := IsCandidateField("someField", "objectId", nil)
	if !r.IsCandidate {
		t.Error("objectId typed field should be candidate")
	}
	if r.Reason != "field type is ObjectId" {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestIsCandidateField_HexStringValues(t *testing.T) {
	values := []any{
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
		"507f1f77bcf86cd799439013",
	}
	r := IsCandidateField("someField", "string", values)
	if !r.IsCandidate {
		t.Error("string field with hex ObjectID values should be candidate")
	}
}

func TestIsCandidateField_HexStringValues_BelowThreshold(t *testing.T) {
	values := []any{
		"not-a-hex-id",
		"also-not-hex",
		"507f1f77bcf86cd799439011",
	}
	r := IsCandidateField("someField", "string", values)
	if r.IsCandidate {
		t.Error("field with mostly non-hex values should not be candidate")
	}
}

func TestIsCandidateField_NonCandidateFields(t *testing.T) {
	for _, name := range []string{"name", "email", "title", "description", "price", "quantity", "status", "createdAt", "updatedAt"} {
		r := IsCandidateField(name, "string", nil)
		if r.IsCandidate {
			t.Errorf("%q should not be candidate", name)
		}
	}
}

func TestIsCandidateField_EmptyValues(t *testing.T) {
	r := IsCandidateField("randomField", "string", []any{})
	if r.IsCandidate {
		t.Error("non-matching field with empty values should not be candidate")
	}
}

func TestIsCandidateField_NilValues(t *testing.T) {
	r := IsCandidateField("randomField", "string", nil)
	if r.IsCandidate {
		t.Error("non-matching field with nil values should not be candidate")
	}
}

func TestIsCandidateField_HexStringValues_EmptyStringsIgnored(t *testing.T) {
	values := []any{
		"",
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
	}
	r := IsCandidateField("someField", "string", values)
	if !r.IsCandidate {
		t.Error("should be candidate: empty strings are ignored, remaining are hex")
	}
}

func TestIsCandidateField_NonStringValuesIgnored(t *testing.T) {
	values := []any{
		42,
		true,
		"507f1f77bcf86cd799439011",
	}
	r := IsCandidateField("someField", "string", values)
	if !r.IsCandidate {
		t.Error("should be candidate: non-string values are ignored, remaining is hex")
	}
}

func TestIsCandidateField_ArrayOfObjectIds(t *testing.T) {
	values := []any{
		primitive.NewObjectID(),
		primitive.NewObjectID(),
	}
	r := IsCandidateField("tagIds", "array", values)
	if !r.IsCandidate {
		t.Error("array of ObjectIds should be candidate")
	}
	if r.Reason != "array of ObjectIds" {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestIsCandidateField_ArrayOfHexStrings(t *testing.T) {
	values := []any{
		"507f1f77bcf86cd799439011",
		"507f1f77bcf86cd799439012",
		"507f1f77bcf86cd799439013",
	}
	r := IsCandidateField("tagIds", "array", values)
	if !r.IsCandidate {
		t.Error("array of hex strings should be candidate")
	}
	if r.Reason != "array of hex strings (look like ObjectIDs)" {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestIsCandidateField_ArrayOfHexStrings_Threshold(t *testing.T) {
	values := []any{
		"not-hex",
		"also-not-hex",
		"507f1f77bcf86cd799439011",
	}
	r := IsCandidateField("tagIds", "array", values)
	if r.IsCandidate {
		t.Error("array with mostly non-hex strings should not be candidate")
	}
}

func TestIsCandidateField_ArrayOfNonCandidate(t *testing.T) {
	values := []any{
		"hello", "world",
	}
	r := IsCandidateField("tags", "array", values)
	if r.IsCandidate {
		t.Error("array of plain strings should not be candidate")
	}
}

func TestIsCandidateField_NestedCandidateViaFieldName(t *testing.T) {
	r := IsCandidateField("customer.id", "string", nil)
	if !r.IsCandidate {
		t.Error("dotted path with 'id' suffix should be candidate")
	}
	if r.Reason != "field name ends with Id/_id" {
		t.Errorf("unexpected reason: %s", r.Reason)
	}
}

func TestIsCandidateField_NestedCandidateViaBySuffix(t *testing.T) {
	r := IsCandidateField("metadata.createdBy", "string", nil)
	if !r.IsCandidate {
		t.Error("dotted path with By suffix should be candidate")
	}
}

func TestIsCandidateField_NestedExcludesId(t *testing.T) {
	r := IsCandidateField("_id", "objectId", nil)
	if r.IsCandidate {
		t.Error("_id should not be candidate even as exact match")
	}
}
