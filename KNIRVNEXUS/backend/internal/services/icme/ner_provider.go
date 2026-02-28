package icme

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

type NERProvider struct {
	logger *zap.Logger
}

type NEREntity struct {
	Text  string
	Label string
	Start int
	End   int
}

func NewNERProvider(logger *zap.Logger) (*NERProvider, error) {
	return &NERProvider{logger: logger}, nil
}

func (n *NERProvider) ExtractEntitiesAndRelations(text string) ([]ExtractedEntity, []ExtractedRelation, error) {
	entities := make([]ExtractedEntity, 0)

	patterns := map[string]*regexp.Regexp{
		"PERSON":   regexp.MustCompile(`\b([A-Z][a-z]+(?:\s+[A-Z][a-z]+)*)\b`),
		"ORG":      regexp.MustCompile(`\b([A-Z][A-Za-z0-9]*(?:\s+[A-Z][A-Za-z0-9]+)*)\b`),
		"ERROR":    regexp.MustCompile(`(?i)(error|exception|bug|fault|issue|failure)`),
		"SOLUTION": regexp.MustCompile(`(?i)(fix|solution|resolution|patch|workaround)`),
		"CONFIG":   regexp.MustCompile(`(?i)(config|configuration|setting|option|parameter)`),
		"EVENT":    regexp.MustCompile(`(?i)(event|incident|occurrence|trigger)`),
	}

	entityID := 0
	for label, pattern := range patterns {
		matches := pattern.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			if len(match) >= 4 {
				start, end := match[2], match[3]
				entityText := text[start:end]
				if len(entityText) > 2 {
					entities = append(entities, ExtractedEntity{
						ID:    fmt.Sprintf("ent_%d", entityID),
						Text:  entityText,
						Label: label,
						Score: 0.85,
						Start: start,
						End:   end,
					})
					entityID++
				}
			}
		}
	}

	relations := make([]ExtractedRelation, 0)
	entList := entities
	for i, e1 := range entList {
		for _, e2 := range entList[i+1:] {
			rel := inferRelation(e1.Label, e2.Label)
			if rel != "" {
				relations = append(relations, ExtractedRelation{
					FromEntityID: e1.ID,
					ToEntityID:   e2.ID,
					RelationType: rel,
					Confidence:   0.75,
				})
			}
		}
	}

	return entities, relations, nil
}

func inferRelation(label1, label2 string) string {
	if label1 == "ERROR" && label2 == "SOLUTION" {
		return "RESOLVED_BY"
	}
	if label1 == "CONFIG" && label2 == "EVENT" {
		return "TRIGGERS"
	}
	if label1 == "PERSON" && label2 == "ORG" {
		return "WORKS_AT"
	}
	if label1 == "ORG" && label2 == "EVENT" {
		return "ORGANIZES"
	}
	return ""
}

func (n *NERProvider) ExtractDomain(text string) string {
	lower := strings.ToLower(text)
	domainKeywords := map[string][]string{
		"technical":  {"api", "code", "sdk", "implementation", "function", "class", "module"},
		"scientific": {"research", "study", "experiment", "data", "analysis", "hypothesis"},
		"historical": {"history", "event", "timeline", "past", "century", "era"},
		"financial":  {"finance", "money", "market", "investment", "transaction", "bank"},
		"medical":    {"health", "medical", "clinical", "patient", "diagnosis", "treatment"},
	}

	for domain, keywords := range domainKeywords {
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return domain
			}
		}
	}
	return "general"
}
