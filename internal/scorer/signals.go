package scorer

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	idSuffixRe  = regexp.MustCompile(`(?i)([Ii]d|_id)$`)
	refSuffixRe = regexp.MustCompile(`(?i)(ref|reference)$`)
	bySuffixRe  = regexp.MustCompile(`(?i)(by|By)$`)
)

var pluralSuffixes = []struct {
	suffix string
	repl   string
}{
	{"ies", "y"},
	{"ches", "ch"},
	{"shes", "sh"},
	{"ses", "se"},
	{"xes", "x"},
	{"ves", "f"},
	{"ges", "ge"},
}

func scoreValueOverlap(matched, sampled int) SignalResult {
	if sampled == 0 {
		return SignalResult{Score: 0, Reason: "no values sampled"}
	}
	score := float64(matched) / float64(sampled)
	return SignalResult{
		Score:  score,
		Reason: fmt.Sprintf("%d/%d values matched (%.1f%%)", matched, sampled, score*100),
	}
}

func scoreNameSimilarity(sourceField, targetCollection string) SignalResult {
	stem := stemFieldName(sourceField)
	if stem == "" {
		return SignalResult{Score: 0}
	}

	stemLower := strings.ToLower(stem)
	targetLower := strings.ToLower(targetCollection)

	if stemLower == targetLower {
		return SignalResult{
			Score:  1.0,
			Reason: fmt.Sprintf("field name '%s' matches collection '%s' exactly", sourceField, targetCollection),
		}
	}

	singularTarget := toSingular(targetCollection)
	if stemLower == singularTarget {
		return SignalResult{
			Score:  0.9,
			Reason: fmt.Sprintf("field name '%s' matches collection '%s' (singular)", sourceField, targetCollection),
		}
	}

	pluralStem := toPlural(stem)
	if pluralStem == targetLower || pluralStem == singularTarget {
		return SignalResult{
			Score:  0.7,
			Reason: fmt.Sprintf("field name '%s' is plural/singular variant of '%s'", sourceField, targetCollection),
		}
	}

	if strings.Contains(targetLower, stemLower) || strings.Contains(stemLower, targetLower) {
		return SignalResult{
			Score:  0.5,
			Reason: fmt.Sprintf("field name '%s' partially matches collection '%s'", sourceField, targetCollection),
		}
	}

	return SignalResult{Score: 0}
}

func scoreTypeCompatibility(sourceType, targetType string) SignalResult {
	if sourceType == "objectId" && targetType == "objectId" {
		return SignalResult{
			Score:  1.0,
			Reason: fmt.Sprintf("type compatible: %s → %s", sourceType, targetType),
		}
	}
	if sourceType == "string" && targetType == "string" {
		return SignalResult{
			Score:  0.7,
			Reason: fmt.Sprintf("type compatible: %s → %s (both strings)", sourceType, targetType),
		}
	}
	if sourceType == "string" && targetType == "objectId" {
		return SignalResult{
			Score:  0.4,
			Reason: fmt.Sprintf("type potentially compatible: %s → %s (string may contain hex ObjectIDs)", sourceType, targetType),
		}
	}
	if sourceType == "objectId" && targetType == "string" {
		return SignalResult{
			Score:  0.4,
			Reason: fmt.Sprintf("type potentially compatible: %s → %s (ObjectID may match hex strings)", sourceType, targetType),
		}
	}
	return SignalResult{Score: 0}
}

func scoreNamingConvention(candidateReason string) SignalResult {
	switch {
	case strings.Contains(candidateReason, "Id/_id"):
		return SignalResult{
			Score:  0.9,
			Reason: fmt.Sprintf("strong naming convention: %s", candidateReason),
		}
	case strings.Contains(candidateReason, "Ref/Reference"):
		return SignalResult{
			Score:  0.8,
			Reason: fmt.Sprintf("good naming convention: %s", candidateReason),
		}
	case strings.Contains(candidateReason, "By"):
		return SignalResult{
			Score:  0.6,
			Reason: fmt.Sprintf("moderate naming convention: %s", candidateReason),
		}
	case strings.Contains(candidateReason, "common reference"):
		return SignalResult{
			Score:  0.5,
			Reason: fmt.Sprintf("common reference name: %s", candidateReason),
		}
	case strings.Contains(candidateReason, "ObjectId"):
		return SignalResult{
			Score:  0.4,
			Reason: fmt.Sprintf("type-based candidate: %s", candidateReason),
		}
	case strings.Contains(candidateReason, "look like ObjectIDs"):
		return SignalResult{
			Score:  0.3,
			Reason: fmt.Sprintf("content-based candidate: %s", candidateReason),
		}
	default:
		return SignalResult{Score: 0.1, Reason: "no naming convention detected"}
	}
}

func scoreUniqueness(ratio float64) SignalResult {
	if ratio <= 0 {
		return SignalResult{Score: 0}
	}
	return SignalResult{
		Score:  ratio,
		Reason: fmt.Sprintf("target field uniqueness: %.1f%%", ratio*100),
	}
}

func stemFieldName(fieldName string) string {
	s := fieldName

	s = idSuffixRe.ReplaceAllString(s, "")
	s = refSuffixRe.ReplaceAllString(s, "")
	s = bySuffixRe.ReplaceAllString(s, "")

	if s == "" {
		return fieldName
	}
	return s
}

func toSingular(word string) string {
	lower := strings.ToLower(word)
	for _, ps := range pluralSuffixes {
		if strings.HasSuffix(lower, ps.suffix) {
			return lower[:len(lower)-len(ps.suffix)] + ps.repl
		}
	}
	if strings.HasSuffix(lower, "s") && len(lower) > 2 {
		return lower[:len(lower)-1]
	}
	return lower
}

func toPlural(word string) string {
	lower := strings.ToLower(word)
	if lower == "" {
		return lower
	}
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "sh") {
		return lower + "es"
	}
	return lower + "s"
}
