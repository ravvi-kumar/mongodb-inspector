package scorer

import (
	"fmt"
	"strings"
)

type SignalResult struct {
	Score  float64
	Reason string
}

type CompositeResult struct {
	Confidence  float64
	Explanation string
	Signals     []SignalResult
}

type ScoreParams struct {
	Matched               int
	Sampled               int
	SourceField           string
	SourceType            string
	TargetCollection      string
	TargetField           string
	TargetType            string
	CandidateReason       string
	TargetUniquenessRatio float64
}

type Scorer struct {
	weights map[string]float64
}

func NewScorer() *Scorer {
	return &Scorer{
		weights: map[string]float64{
			"value_overlap":      0.50,
			"name_similarity":    0.20,
			"type_compatibility": 0.15,
			"naming_convention":  0.05,
			"uniqueness":         0.10,
		},
	}
}

func (s *Scorer) Score(params ScoreParams) CompositeResult {
	var signals []SignalResult

	valSignal := scoreValueOverlap(params.Matched, params.Sampled)
	signals = append(signals, valSignal)

	nameSignal := scoreNameSimilarity(params.SourceField, params.TargetCollection)
	signals = append(signals, nameSignal)

	typeSignal := scoreTypeCompatibility(params.SourceType, params.TargetType)
	signals = append(signals, typeSignal)

	convSignal := scoreNamingConvention(params.CandidateReason)
	signals = append(signals, convSignal)

	uniqSignal := SignalResult{Score: 0}
	if params.TargetField != "_id" {
		uniqSignal = scoreUniqueness(params.TargetUniquenessRatio)
	}
	signals = append(signals, uniqSignal)

	confidence := 0.0
	confidence += valSignal.Score * s.weights["value_overlap"]
	confidence += nameSignal.Score * s.weights["name_similarity"]
	confidence += typeSignal.Score * s.weights["type_compatibility"]
	confidence += convSignal.Score * s.weights["naming_convention"]
	if params.TargetField != "_id" {
		confidence += uniqSignal.Score * s.weights["uniqueness"]
	} else {
		confidence += 1.0 * s.weights["uniqueness"]
	}

	var parts []string
	for _, sig := range signals {
		if sig.Reason != "" {
			parts = append(parts, sig.Reason+".")
		}
	}
	explanation := strings.Join(parts, " ")

	return CompositeResult{
		Confidence:  confidence,
		Explanation: explanation,
		Signals:     signals,
	}
}

func (s *Scorer) FormatExplanation(result CompositeResult) string {
	if result.Explanation != "" {
		return result.Explanation
	}
	return fmt.Sprintf("Confidence: %.1f%%.", result.Confidence*100)
}
