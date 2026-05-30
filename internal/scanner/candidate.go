package scanner

import (
	"regexp"
	"strings"
)

var (
	camelCaseIDRe    = regexp.MustCompile(`[Ii]d$`)
	snakeCaseIDRe     = regexp.MustCompile(`_id$`)
	refRe             = regexp.MustCompile(`(?i)(ref|reference)$`)
	endsWithByRe      = regexp.MustCompile(`(?i)(by|By)$`)
	hexObjectIDRe     = regexp.MustCompile(`^[a-f0-9]{24}$`)
)

var commonRefNames = map[string]bool{
	"customer":  true,
	"owner":     true,
	"author":    true,
	"parent":    true,
	"user":      true,
	"creator":   true,
	"assignee":  true,
	"approver":  true,
	"reviewer":  true,
	"manager":   true,
	"admin":     true,
	"member":    true,
	"reporter":  true,
	"subscriber": true,
}

type CandidateResult struct {
	IsCandidate bool
	Reason      string
}

func IsCandidateField(fieldName string, fieldType string, sampleValues []any) CandidateResult {
	if fieldName == "_id" {
		return CandidateResult{IsCandidate: false}
	}

	if camelCaseIDRe.MatchString(fieldName) || snakeCaseIDRe.MatchString(fieldName) {
		return CandidateResult{IsCandidate: true, Reason: "field name ends with Id/_id"}
	}

	if refRe.MatchString(fieldName) {
		return CandidateResult{IsCandidate: true, Reason: "field name ends with Ref/Reference"}
	}

	if endsWithByRe.MatchString(fieldName) && fieldName != "sortby" {
		return CandidateResult{IsCandidate: true, Reason: "field name ends with By (e.g. createdBy)"}
	}

	lower := strings.ToLower(fieldName)
	if commonRefNames[lower] {
		return CandidateResult{IsCandidate: true, Reason: "common reference field name"}
	}

	if fieldType == "objectId" {
		return CandidateResult{IsCandidate: true, Reason: "field type is ObjectId"}
	}

	if fieldType == "string" {
		matchCount := 0
		checked := 0
		for _, v := range sampleValues {
			s, ok := v.(string)
			if !ok || s == "" {
				continue
			}
			checked++
			if hexObjectIDRe.MatchString(s) {
				matchCount++
			}
		}
		if checked > 0 && matchCount > checked/2 {
			return CandidateResult{IsCandidate: true, Reason: "string values look like ObjectIDs"}
		}
	}

	return CandidateResult{IsCandidate: false}
}
