package normalizer

import (
	"encoding/json"
	"fmt"

	"github.com/knirv/hasher/pipeline/0_DATA_CONNECTOR/internal/cleaner"
)

type SecurityNormalizer struct {
	scrubPII    bool
	securityMap *SecurityMapper
}

func NewSecurityNormalizer(scrubPII bool) *SecurityNormalizer {
	return &SecurityNormalizer{
		scrubPII:    scrubPII,
		securityMap: NewSecurityMapper(),
	}
}

func (n *SecurityNormalizer) Process(data []byte) ([]*SecurityRecord, error) {
	var export SecurityExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	var records []*SecurityRecord
	chunkID := int32(0)

	for _, guardrail := range export.Guardrails {
		record := n.normalizeGuardrail(guardrail, chunkID)
		records = append(records, record)
		chunkID++
	}

	for _, constraint := range export.Constraints {
		record := n.normalizeConstraint(constraint, chunkID)
		records = append(records, record)
		chunkID++
	}

	for _, violation := range export.Violations {
		record := n.normalizeViolation(violation, chunkID)
		records = append(records, record)
		chunkID++
	}

	for _, doc := range export.MarkdownDocs {
		record := n.normalizeMarkdown(doc, chunkID)
		records = append(records, record)
		chunkID++
	}

	for _, user := range export.Users {
		record := n.normalizeUser(user, chunkID)
		records = append(records, record)
		chunkID++
	}

	return records, nil
}

func (n *SecurityNormalizer) normalizeGuardrail(g GuardrailRule, chunkID int32) *SecurityRecord {
	tags := []string{
		fmt.Sprintf("guardrail_%s", g.Type),
		fmt.Sprintf("action_%s", g.Action),
	}

	mapping := n.securityMap.GetMapping(g.Type, g.Action)
	if mapping != nil {
		tags = append(tags, mapping.Tag)
	}

	content := fmt.Sprintf("Guardrail: %s type=%s action=%s", g.Domain, g.Type, g.Action)
	if len(g.Concepts) > 0 {
		content += " concepts=" + joinStrings(g.Concepts, ",")
	}

	return &SecurityRecord{
		FileName:     fmt.Sprintf("guardrail_%s_%d", g.Type, chunkID),
		ChunkID:      chunkID,
		Content:      content,
		Tokens:       tokenize(content),
		POSTags:      n.inferPOSTags(content),
		DepHashes:    n.computeDepHashes(content),
		SecurityTags: tags,
		Slot4:        mapping.Slot4,
		Slot10:       mapping.Slot10,
		Weight:       mapping.Weight,
	}
}

func (n *SecurityNormalizer) normalizeConstraint(c Constraint, chunkID int32) *SecurityRecord {
	tags := []string{"security_constraint"}
	tags = append(tags, c.Tags...)

	mapping := n.securityMap.GetMapping("constraint", "deny")
	if mapping != nil {
		tags = append(tags, mapping.Tag)
	}

	return &SecurityRecord{
		FileName:     fmt.Sprintf("constraint_%s_%d", c.RuleID, chunkID),
		ChunkID:      chunkID,
		Content:      c.Pattern,
		Tokens:       tokenize(c.Pattern),
		POSTags:      n.inferPOSTags(c.Pattern),
		DepHashes:    n.computeDepHashes(c.Pattern),
		SecurityTags: tags,
		Slot4:        mapping.Slot4,
		Slot10:       mapping.Slot10,
		Weight:       mapping.Weight,
	}
}

func (n *SecurityNormalizer) normalizeViolation(v Violation, chunkID int32) *SecurityRecord {
	tags := []string{"security_violation", fmt.Sprintf("severity_%s", v.Severity)}

	mapping := n.securityMap.GetMapping("violation", v.Severity)
	if mapping != nil {
		tags = append(tags, mapping.Tag)
	}

	content := fmt.Sprintf("Violation: %s severity=%s guardrail=%s", v.Message, v.Severity, v.GuardrailType)

	return &SecurityRecord{
		FileName:     fmt.Sprintf("violation_%d", chunkID),
		ChunkID:      chunkID,
		Content:      content,
		Tokens:       tokenize(content),
		POSTags:      n.inferPOSTags(content),
		DepHashes:    n.computeDepHashes(content),
		SecurityTags: tags,
		Slot4:        mapping.Slot4,
		Slot10:       mapping.Slot10,
		Weight:       mapping.Weight,
	}
}

func (n *SecurityNormalizer) normalizeMarkdown(doc MarkdownDoc, chunkID int32) *SecurityRecord {
	tags := []string{"domain_knowledge", fmt.Sprintf("type_%s", doc.Type)}

	return &SecurityRecord{
		FileName:     fmt.Sprintf("doc_%s_%d", doc.ID, chunkID),
		ChunkID:      chunkID,
		Content:      doc.Content,
		Tokens:       tokenize(doc.Content),
		POSTags:      n.inferPOSTags(doc.Content),
		DepHashes:    n.computeDepHashes(doc.Content),
		SecurityTags: tags,
		Slot4:        0x01,
		Slot10:       0x1000,
		Weight:       0.0,
	}
}

func (n *SecurityNormalizer) normalizeUser(user UserProfile, chunkID int32) *SecurityRecord {
	tags := []string{"user_profile", fmt.Sprintf("role_%s", user.Role)}

	content := fmt.Sprintf("User: %s role=%s", user.Username, user.Role)

	return &SecurityRecord{
		FileName:     fmt.Sprintf("user_%s_%d", user.ID, chunkID),
		ChunkID:      chunkID,
		Content:      content,
		Tokens:       tokenize(content),
		POSTags:      n.inferPOSTags(content),
		DepHashes:    n.computeDepHashes(content),
		SecurityTags: tags,
		Slot4:        0x01,
		Slot10:       0x1000,
		Weight:       0.0,
	}
}

func (n *SecurityNormalizer) inferPOSTags(content string) []int {
	tags := make([]int, 0, len(n.tokenizeSimple(content)))
	for _, tok := range n.tokenizeSimple(content) {
		tags = append(tags, guessPOSTag(tok))
	}
	return tags
}

func (n *SecurityNormalizer) computeDepHashes(content string) []uint32 {
	tokens := n.tokenizeSimple(content)
	hashes := make([]uint32, 0, len(tokens))
	for i, tok := range tokens {
		hashes = append(hashes, simpleHash(tok+string(rune('0'+i%10))))
	}
	return hashes
}

func (n *SecurityNormalizer) tokenizeSimple(text string) []string {
	tokens := make([]string, 0)
	var current []byte
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' || r == ',' || r == '=' {
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = nil
			}
		} else {
			current = append(current, byte(r))
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	return tokens
}

func simpleHash(s string) uint32 {
	var h uint32 = 2166136261
	for _, c := range s {
		h ^= uint32(c)
		h *= 16777619
	}
	return h
}

func guessPOSTag(token string) int {
	switch {
	case token == "User" || token == "Guardrail" || token == "Violation" || token == "Constraint":
		return 0x01
	case token == "no" || token == "not" || token == "deny" || token == "block":
		return 0x04
	case token == "in" || token == "on" || token == "for":
		return 0x07
	default:
		return 0x01
	}
}

type SecurityExport struct {
	OrgID        string          `json:"org_id"`
	Users        []UserProfile   `json:"users"`
	Guardrails   []GuardrailRule `json:"guardrails"`
	Constraints  []Constraint    `json:"constraints"`
	Violations   []Violation     `json:"violations"`
	MarkdownDocs []MarkdownDoc   `json:"markdown_docs"`
	ExportedAt   string          `json:"exported_at"`
}

type UserProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role"`
}

type GuardrailRule struct {
	Type     string   `json:"type"`
	Action   string   `json:"action"`
	Domain   string   `json:"domain,omitempty"`
	Concepts []string `json:"concepts,omitempty"`
}

type Constraint struct {
	RuleID  string   `json:"rule_id"`
	Pattern string   `json:"pattern"`
	Tags    []string `json:"tags"`
}

type Violation struct {
	GuardrailType string `json:"guardrail_type"`
	Message       string `json:"message"`
	Severity      string `json:"severity"`
}

type MarkdownDoc struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

func tokenize(text string) []string {
	tokens := make([]string, 0)
	var current []byte
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = nil
			}
		} else {
			current = append(current, byte(r))
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	return tokens
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
