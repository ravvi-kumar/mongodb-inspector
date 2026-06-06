package scorer

import (
	"testing"
)

func TestScorer_Composite(t *testing.T) {
	s := NewScorer()

	params := ScoreParams{
		Matched:               45,
		Sampled:               50,
		SourceField:           "userId",
		SourceType:            "objectId",
		TargetCollection:      "users",
		TargetField:           "_id",
		TargetType:            "objectId",
		CandidateReason:       "field name ends with Id/_id",
		TargetUniquenessRatio: 1.0,
	}
	result := s.Score(params)

	if result.Confidence <= 0.5 {
		t.Errorf("expected confidence > 0.5 for strong signals, got %f", result.Confidence)
	}
	if result.Confidence > 1.0 {
		t.Errorf("confidence should not exceed 1.0, got %f", result.Confidence)
	}
	if len(result.Signals) != 5 {
		t.Errorf("expected 5 signals, got %d", len(result.Signals))
	}
	if result.Explanation == "" {
		t.Error("expected non-empty explanation")
	}
}

func TestScorer_WeakSignals(t *testing.T) {
	s := NewScorer()

	params := ScoreParams{
		Matched:               1,
		Sampled:               100,
		SourceField:           "data",
		SourceType:            "string",
		TargetCollection:      "unrelated",
		TargetField:           "_id",
		TargetType:            "objectId",
		CandidateReason:       "common reference field name",
		TargetUniquenessRatio: 1.0,
	}
	result := s.Score(params)

	if result.Confidence >= 0.5 {
		t.Errorf("expected confidence < 0.5 for weak signals, got %f", result.Confidence)
	}
}

func TestScorer_NonIDTarget(t *testing.T) {
	s := NewScorer()

	params := ScoreParams{
		Matched:               45,
		Sampled:               50,
		SourceField:           "email",
		SourceType:            "string",
		TargetCollection:      "users",
		TargetField:           "email",
		TargetType:            "string",
		CandidateReason:       "",
		TargetUniquenessRatio: 0.95,
	}
	result := s.Score(params)

	if result.Confidence <= 0.5 {
		t.Errorf("expected confidence > 0.5 for good match, got %f", result.Confidence)
	}

	hasUniqueness := false
	for _, sig := range result.Signals {
		if sig.Reason != "" && len(sig.Reason) > 0 {
			t.Logf("signal: %s (score: %f)", sig.Reason, sig.Score)
		}
		if sig.Score > 0 && sig.Reason != "" {
			_ = hasUniqueness
		}
	}
	_ = hasUniqueness
}

func TestFormatExplanation(t *testing.T) {
	s := NewScorer()
	result := s.Score(ScoreParams{
		Matched:               10,
		Sampled:               10,
		SourceField:           "userId",
		SourceType:            "objectId",
		TargetCollection:      "users",
		TargetField:           "_id",
		TargetType:            "objectId",
		CandidateReason:       "field name ends with Id/_id",
		TargetUniquenessRatio: 1.0,
	})

	explanation := s.FormatExplanation(result)
	if explanation == "" {
		t.Error("expected non-empty formatted explanation")
	}
}

func TestScorer_WeightsSum(t *testing.T) {
	s := NewScorer()
	sum := 0.0
	for _, w := range s.weights {
		sum += w
	}
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("weights should sum to approximately 1.0, got %f", sum)
	}
}
