package normalizer

import (
	"fmt"
	"strings"
)

type SecurityNormalizer struct {
	schema *SecuritySchema
}

type SecurityRecord struct {
	FileName     string    `arrow:"file_name"`
	ChunkID      int32     `arrow:"chunk_id"`
	Content      string    `arrow:"content"`
	Embedding    []float32 `arrow:"embedding"`
	Tokens       []string  `arrow:"tokens"`
	POSTags      []int     `arrow:"pos_tags"`
	DepHashes    []uint32  `arrow:"dep_hashes"`
	SecurityTags []string  `arrow:"security_tags"`
	DomainSig    uint16    `arrow:"domain_sig"`
	Slot4Raw     uint32    `arrow:"slot4_raw"`
}

var SECURITY_TAG_MAPPINGS = map[string]SecurityMapping{
	"guardrail_block": {
		Slot10: 0x2400, // Logic/Set domain
		Slot4:  0x07,   // PREP (constraint marker)
		Weight: -1.0,   // Negative reinforcement
	},
	"guardrail_warn": {
		Slot10: 0x2401, // Guardrail subdomain
		Slot4:  0x04,   // ADV
		Weight: -0.5,
	},
	"security_constraint": {
		Slot10: 0x2400,
		Slot4:  0x02, // VERB (action)
		Weight: 1.0,
	},
	"violation": {
		Slot10: 0x2402, // Violation subdomain
		Slot4:  0x01,   // NOUN (subject)
		Weight: -2.0,   // Strong negative
	},
}

type SecurityMapping struct {
	Slot10 uint16
	Slot4  uint8
	Weight float32
}

type SecuritySchema struct {
	// Schema definition
}

func NewSecurityNormalizer() *SecurityNormalizer {
	return &SecurityNormalizer{
		schema: &SecuritySchema{},
	}
}

func (sn *SecurityNormalizer) Process(input string) ([]*SecurityRecord, error) {
	// Tokenize and process with SpaCy (placeholder)
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens found")
	}

	records := make([]*SecurityRecord, len(tokens))
	for i, token := range tokens {
		records[i] = &SecurityRecord{
			FileName:     "processed.md",
			ChunkID:      int32(i),
			Content:      token,
			Tokens:       []string{token},
			POSTags:      []int{1}, // Placeholder POS tag
			DepHashes:    []uint32{0},
			SecurityTags: sn.extractSecurityTags(token),
			DomainSig:    0x2400, // Default security domain
			Slot4Raw:     0x01,   // Default NOUN
		}
	}

	return records, nil
}

func (sn *SecurityNormalizer) extractSecurityTags(content string) []string {
	var tags []string
	for tag := range SECURITY_TAG_MAPPINGS {
		if strings.Contains(strings.ToLower(content), tag) {
			tags = append(tags, tag)
		}
	}
	return tags
}
