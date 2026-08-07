package processing

import (
	"KNIRVGRAPH/internal/types"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	emailPattern    = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	urlPattern      = regexp.MustCompile(`https?://[^\s)]+`)
	ipPattern       = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	uuidPattern     = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	datePattern     = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
	numberPattern   = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
)

type Extractor struct {
	config types.ExtractionConfig
}

func NewExtractor(config types.ExtractionConfig) *Extractor {
	if len(config.EntityTypes) == 0 {
		config.EntityTypes = []string{"PERSON", "ORG", "GPE", "PRODUCT", "EVENT", "DATE", "EMAIL", "URL", "IP", "UUID"}
	}
	if config.MinConfidence <= 0 {
		config.MinConfidence = 0.5
	}
	return &Extractor{config: config}
}

func (e *Extractor) Extract(documentID, text string) ([]types.ExtractedEntity, []types.ExtractedRelationship, error) {
	var entities []types.ExtractedEntity
	var relationships []types.ExtractedRelationship

	if e.config.EnableEntities {
		ents, err := e.extractEntities(documentID, text)
		if err != nil {
			return nil, nil, fmt.Errorf("entity extraction failed: %w", err)
		}
		entities = ents
	}

	if e.config.EnableRelationships {
		rels, err := e.extractRelationships(documentID, text, entities)
		if err != nil {
			return nil, nil, fmt.Errorf("relationship extraction failed: %w", err)
		}
		relationships = rels
	}

	return entities, relationships, nil
}

func (e *Extractor) extractEntities(documentID, text string) ([]types.ExtractedEntity, error) {
	var entities []types.ExtractedEntity

	// Regex-based entity extraction
	entities = append(entities, e.extractByPattern(documentID, text, "EMAIL", emailPattern, 0.9)...)
	entities = append(entities, e.extractByPattern(documentID, text, "URL", urlPattern, 0.85)...)
	entities = append(entities, e.extractByPattern(documentID, text, "IP", ipPattern, 0.9)...)
	entities = append(entities, e.extractByPattern(documentID, text, "UUID", uuidPattern, 0.95)...)
	entities = append(entities, e.extractByPattern(documentID, text, "DATE", datePattern, 0.8)...)

	// Try LLM-based extraction if configured
	if e.config.LLMEndpoint != "" {
		llmEntities, err := e.extractWithLLM(documentID, text)
		if err == nil {
			entities = append(entities, llmEntities...)
		}
	}

	// Deduplicate and filter by confidence
	seen := make(map[string]bool)
	var filtered []types.ExtractedEntity
	for _, ent := range entities {
		if ent.Confidence < e.config.MinConfidence {
			continue
		}
		key := ent.Type + ":" + ent.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		filtered = append(filtered, ent)
	}

	for i := range filtered {
		filtered[i].ID = fmt.Sprintf("entity_%s_%d", documentID, i)
		filtered[i].DocumentID = documentID
		filtered[i].CreatedAt = time.Now()
	}

	return filtered, nil
}

func (e *Extractor) extractByPattern(documentID, text, entityType string, pattern *regexp.Regexp, baseConfidence float64) []types.ExtractedEntity {
	var entities []types.ExtractedEntity
	matches := pattern.FindAllString(text, -1)
	for _, match := range matches {
		entities = append(entities, types.ExtractedEntity{
			Type:       entityType,
			Name:       match,
			Confidence: baseConfidence,
		})
	}
	return entities
}

func (e *Extractor) extractWithLLM(documentID, text string) ([]types.ExtractedEntity, error) {
	if e.config.LLMEndpoint == "" {
		return nil, nil
	}

	prompt := fmt.Sprintf(`Extract named entities from the following text. Return JSON array with objects having fields: type, name, confidence. Text: %s`, truncate(text, 4000))

	reqBody := map[string]interface{}{
		"model":  e.config.LLMModel,
		"prompt": prompt,
		"stream": false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", e.config.LLMEndpoint+"/api/generate", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM endpoint returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return parseLLMEntities(documentID, result.Response)
}

func parseLLMEntities(documentID, response string) ([]types.ExtractedEntity, error) {
	var entities []types.ExtractedEntity
	scanner := bufio.NewScanner(strings.NewReader(response))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ent struct {
			Type       string  `json:"type"`
			Name       string  `json:"name"`
			Confidence float64 `json:"confidence"`
		}
		if err := json.Unmarshal([]byte(line), &ent); err != nil {
			continue
		}
		if ent.Type != "" && ent.Name != "" {
			entities = append(entities, types.ExtractedEntity{
				Type:       ent.Type,
				Name:       ent.Name,
				Confidence: ent.Confidence,
			})
		}
	}
	return entities, scanner.Err()
}

func (e *Extractor) extractRelationships(documentID, text string, entities []types.ExtractedEntity) ([]types.ExtractedRelationship, error) {
	var relationships []types.ExtractedRelationship

	entityNames := make(map[string]bool)
	for _, ent := range entities {
		entityNames[ent.Name] = true
	}

	// Co-occurrence based relationship extraction
	sentences := sentenceEnders.Split(text, -1)
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		words := strings.Fields(sentence)
		found := make(map[string]bool)
		for _, word := range words {
			clean := strings.Trim(word, ".,!?;:\"'()")
			if entityNames[clean] && !found[clean] {
				found[clean] = true
			}
		}
		pairs := make([]string, 0, len(found))
		for name := range found {
			pairs = append(pairs, name)
		}
		for i := 0; i < len(pairs); i++ {
			for j := i + 1; j < len(pairs); j++ {
				relationships = append(relationships, types.ExtractedRelationship{
					Source:     pairs[i],
					Target:     pairs[j],
					Type:       "co_occurs",
					Weight:     0.5,
					Evidence:   sentence,
					Confidence: 0.6,
				})
			}
		}
	}

	// Pattern-based relationship extraction
	relationships = append(relationships, e.extractPatternRelationships(documentID, text)...)

	// Try LLM-based relationship extraction
	if e.config.LLMEndpoint != "" {
		llmRels, err := e.extractRelationshipsWithLLM(documentID, text)
		if err == nil {
			relationships = append(relationships, llmRels...)
		}
	}

	for i := range relationships {
		if relationships[i].ID == "" {
			relationships[i].ID = fmt.Sprintf("rel_%s_%d", documentID, i)
		}
		relationships[i].DocumentID = documentID
		relationships[i].CreatedAt = time.Now()
	}

	return relationships, nil
}

func (e *Extractor) extractPatternRelationships(documentID, text string) []types.ExtractedRelationship {
	var relationships []types.ExtractedRelationship

	patterns := []struct {
		pattern     *regexp.Regexp
		relType     string
		confidence  float64
	}{
		{regexp.MustCompile(`(?i)(\w+)\s+(?:is\s+)?(?:a|an|the)\s+(?:type\s+of\s+)?(\w+)`), "is_a", 0.7},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:owns|owned\s+by)\s+(\w+)`), "owns", 0.8},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:located\s+in|in)\s+(\w+)`), "located_in", 0.8},
		{regexp.MustCompile(`(?i)(\w+)\s+(?:works\s+for|at)\s+(\w+)`), "works_for", 0.75},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				relationships = append(relationships, types.ExtractedRelationship{
					Source:     match[1],
					Target:     match[2],
					Type:       p.relType,
					Weight:     0.7,
					Evidence:   match[0],
					Confidence: p.confidence,
				})
			}
		}
	}

	return relationships
}

func (e *Extractor) extractRelationshipsWithLLM(documentID, text string) ([]types.ExtractedRelationship, error) {
	if e.config.LLMEndpoint == "" {
		return nil, nil
	}

	prompt := fmt.Sprintf(`Extract relationships from the following text. Return JSON array with objects having fields: source, target, type, weight, evidence, confidence. Text: %s`, truncate(text, 4000))

	reqBody := map[string]interface{}{
		"model":  e.config.LLMModel,
		"prompt": prompt,
		"stream": false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", e.config.LLMEndpoint+"/api/generate", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM endpoint returned status %d", resp.StatusCode)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var rels []types.ExtractedRelationship
	scanner := bufio.NewScanner(strings.NewReader(result.Response))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rel struct {
			Source     string  `json:"source"`
			Target     string  `json:"target"`
			Type       string  `json:"type"`
			Weight     float64 `json:"weight"`
			Evidence   string  `json:"evidence"`
			Confidence float64 `json:"confidence"`
		}
		if err := json.Unmarshal([]byte(line), &rel); err != nil {
			continue
		}
		if rel.Source != "" && rel.Target != "" {
			rels = append(rels, types.ExtractedRelationship{
				Source:     rel.Source,
				Target:     rel.Target,
				Type:       rel.Type,
				Weight:     rel.Weight,
				Evidence:   rel.Evidence,
				Confidence: rel.Confidence,
			})
		}
	}
	return rels, scanner.Err()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
