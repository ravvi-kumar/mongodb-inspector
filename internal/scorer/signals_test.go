package scorer

import (
	"testing"
)

func TestScoreValueOverlap(t *testing.T) {
	tests := []struct {
		name     string
		matched  int
		sampled  int
		wantMin  float64
		wantMax  float64
	}{
		{"perfect", 10, 10, 1.0, 1.0},
		{"half", 5, 10, 0.5, 0.5},
		{"none", 0, 10, 0.0, 0.0},
		{"zero sampled", 0, 0, 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scoreValueOverlap(tt.matched, tt.sampled)
			if result.Score < tt.wantMin || result.Score > tt.wantMax {
				t.Errorf("scoreValueOverlap(%d, %d) = %f, want [%f, %f]", tt.matched, tt.sampled, result.Score, tt.wantMin, tt.wantMax)
			}
			if result.Score > 0 && result.Reason == "" {
				t.Error("expected non-empty reason for non-zero score")
			}
		})
	}
}

func TestScoreNameSimilarity(t *testing.T) {
	tests := []struct {
		sourceField      string
		targetCollection string
		wantMin          float64
		wantContains     string
	}{
		{"userId", "users", 0.7, "plural"},
		{"customer", "customers", 0.7, "plural"},
		{"orderId", "orders", 0.7, "plural"},
		{"category", "categories", 0.7, "plural"},
		{"createdBy", "users", 0.0, ""},
		{"random", "unrelated", 0.0, ""},
		{"author", "authors", 0.7, "plural"},
		{"reviewer", "reviewers", 0.7, "plural"},
		{"owner", "owners", 0.7, "plural"},
		{"productId", "products", 0.7, "plural"},
		{"companyId", "companies", 0.7, "plural"},
		{"contactId", "contacts", 0.7, "plural"},
	}

	for _, tt := range tests {
		t.Run(tt.sourceField+"->"+tt.targetCollection, func(t *testing.T) {
			result := scoreNameSimilarity(tt.sourceField, tt.targetCollection)
			if result.Score < tt.wantMin {
				t.Errorf("scoreNameSimilarity(%q, %q) = %f, want >= %f", tt.sourceField, tt.targetCollection, result.Score, tt.wantMin)
			}
			if tt.wantContains != "" {
				if result.Reason == "" {
					t.Errorf("expected reason containing %q, got empty", tt.wantContains)
				}
			}
		})
	}
}

func TestScoreNameSimilarity_ExactMatch(t *testing.T) {
	result := scoreNameSimilarity("user", "user")
	if result.Score != 1.0 {
		t.Errorf("exact match should score 1.0, got %f", result.Score)
	}
}

func TestScoreNameSimilarity_SingularMatch(t *testing.T) {
	result := scoreNameSimilarity("order", "orders")
	if result.Score != 0.9 {
		t.Errorf("singular match should score 0.9, got %f", result.Score)
	}
}

func TestScoreNameSimilarity_Partial(t *testing.T) {
	result := scoreNameSimilarity("userSettings", "users")
	if result.Score < 0.5 {
		t.Errorf("partial match should score >= 0.5, got %f", result.Score)
	}
}

func TestScoreTypeCompatibility(t *testing.T) {
	tests := []struct {
		source, target string
		want           float64
	}{
		{"objectId", "objectId", 1.0},
		{"string", "string", 0.7},
		{"string", "objectId", 0.4},
		{"objectId", "string", 0.4},
		{"int", "objectId", 0.0},
		{"objectId", "int", 0.0},
	}

	for _, tt := range tests {
		result := scoreTypeCompatibility(tt.source, tt.target)
		if result.Score != tt.want {
			t.Errorf("scoreTypeCompatibility(%q, %q) = %f, want %f", tt.source, tt.target, result.Score, tt.want)
		}
	}
}

func TestScoreNamingConvention(t *testing.T) {
	tests := []struct {
		reason string
		want   float64
	}{
		{"field name ends with Id/_id", 0.9},
		{"field name ends with Ref/Reference", 0.8},
		{"field name ends with By (e.g. createdBy)", 0.6},
		{"common reference field name", 0.5},
		{"field type is ObjectId", 0.4},
		{"string values look like ObjectIDs", 0.3},
		{"no reason", 0.1},
	}

	for _, tt := range tests {
		result := scoreNamingConvention(tt.reason)
		if result.Score != tt.want {
			t.Errorf("scoreNamingConvention(%q) = %f, want %f", tt.reason, result.Score, tt.want)
		}
		if result.Reason == "" {
			t.Errorf("scoreNamingConvention(%q) returned empty reason", tt.reason)
		}
	}
}

func TestScoreUniqueness(t *testing.T) {
	tests := []struct {
		ratio float64
		want  float64
	}{
		{1.0, 1.0},
		{0.5, 0.5},
		{0.0, 0.0},
		{-1.0, 0.0},
	}

	for _, tt := range tests {
		result := scoreUniqueness(tt.ratio)
		if result.Score != tt.want {
			t.Errorf("scoreUniqueness(%f) = %f, want %f", tt.ratio, result.Score, tt.want)
		}
	}
}

func TestStemFieldName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"userId", "user"},
		{"orderId", "order"},
		{"customerRef", "customer"},
		{"productReference", "product"},
		{"createdBy", "created"},
		{"author", "author"},
		{"plain", "plain"},
		{"Id", "Id"},
	}

	for _, tt := range tests {
		got := stemFieldName(tt.input)
		if got != tt.want {
			t.Errorf("stemFieldName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToSingular(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "user"},
		{"categories", "category"},
		{"boxes", "box"},
		{"wolves", "wolf"},
		{"pages", "page"},
		{"churches", "church"},
		{"dishes", "dish"},
		{"orders", "order"},
		{"analytics", "analytic"},
	}

	for _, tt := range tests {
		got := toSingular(tt.input)
		if got != tt.want {
			t.Errorf("toSingular(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToPlural(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "users"},
		{"box", "boxes"},
		{"church", "churches"},
		{"dish", "dishes"},
		{"order", "orders"},
	}

	for _, tt := range tests {
		got := toPlural(tt.input)
		if got != tt.want {
			t.Errorf("toPlural(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
