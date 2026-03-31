package classifier

import (
	"context"
	"regexp"
	"strings"
)

type NodeType string

const (
	NodeTypeSkill      NodeType = "skill"
	NodeTypeCapability NodeType = "capability"
	NodeTypeProperty   NodeType = "property"
	NodeTypeError      NodeType = "error"
	NodeTypeContext    NodeType = "context"
	NodeTypeIdea       NodeType = "idea"
)

type NodeClassifier struct {
	errorPatterns      []*regexp.Regexp
	contextKeywords    []string
	ideaMarkers        []string
	taskIndicators     []string
	capabilityKeywords []string
	skillKeywords      []string
	propertyKeywords   []string
}

func NewNodeClassifier() *NodeClassifier {
	return &NodeClassifier{
		errorPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(error|exception|failed|timeout)`),
			regexp.MustCompile(`(?i)(stack trace|panic|fatal|crash)`),
			regexp.MustCompile(`(?i)(bug|issue|broken)`),
		},
		contextKeywords: []string{
			"user prefers", "location", "environment", "setting",
			"timezone", "language preference", "configuration", "config",
		},
		ideaMarkers: []string{
			"what if", "could we", "hypothesis", "concept",
			"imagine", "innovative", "breakthrough", "idea",
		},
		taskIndicators: []string{
			"todo", "remind me", "task", "action item",
			"need to", "must", "should", "deadline",
		},
		capabilityKeywords: []string{
			"can do", "able to", "capable of", "function",
			"method", "api", "service", "tool", "feature",
			"implement", "perform", "execute", "process",
		},
		skillKeywords: []string{
			"lora", "adapter", "fine-tune", "model", "training",
			"knowledge", "expertise", "proficiency", "skill",
			"experience", "specialization",
		},
		propertyKeywords: []string{
			"property", "attribute", "trait", "characteristic",
			"nft", "inference", "prediction", "output",
			"generated", "created", "made",
		},
	}
}

func (c *NodeClassifier) Classify(ctx context.Context, content string) NodeType {
	lowerContent := strings.ToLower(content)

	if c.isError(lowerContent) {
		return NodeTypeError
	}

	if c.isIdea(lowerContent) {
		return NodeTypeIdea
	}

	if c.isContext(lowerContent) {
		return NodeTypeContext
	}

	if c.isCapability(lowerContent) {
		return NodeTypeCapability
	}

	if c.isSkill(lowerContent) {
		return NodeTypeSkill
	}

	if c.isProperty(lowerContent) {
		return NodeTypeProperty
	}

	if c.isTask(lowerContent) {
		return NodeTypeCapability
	}

	return NodeTypeSkill
}

func (c *NodeClassifier) isError(content string) bool {
	for _, pattern := range c.errorPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func (c *NodeClassifier) isContext(content string) bool {
	for _, keyword := range c.contextKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

func (c *NodeClassifier) isIdea(content string) bool {
	for _, marker := range c.ideaMarkers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}

func (c *NodeClassifier) isTask(content string) bool {
	for _, indicator := range c.taskIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

func (c *NodeClassifier) isCapability(content string) bool {
	count := 0
	for _, keyword := range c.capabilityKeywords {
		if strings.Contains(content, keyword) {
			count++
		}
	}
	return count >= 2
}

func (c *NodeClassifier) isSkill(content string) bool {
	count := 0
	for _, keyword := range c.skillKeywords {
		if strings.Contains(content, keyword) {
			count++
		}
	}
	return count >= 2
}

func (c *NodeClassifier) isProperty(content string) bool {
	count := 0
	for _, keyword := range c.propertyKeywords {
		if strings.Contains(content, keyword) {
			count++
		}
	}
	return count >= 2
}

type ClassificationResult struct {
	NodeType    NodeType `json:"node_type"`
	Confidence  float64  `json:"confidence"`
	Reasons     []string `json:"reasons"`
	SuggestedOp string   `json:"suggested_operation"`
}

func (c *NodeClassifier) ClassifyWithDetails(ctx context.Context, content string) ClassificationResult {
	result := ClassificationResult{
		NodeType:   c.Classify(ctx, content),
		Confidence: 0.7,
		Reasons:    []string{},
	}

	switch result.NodeType {
	case NodeTypeError:
		result.Confidence = 0.9
		result.Reasons = []string{"Contains error-related keywords"}
		result.SuggestedOp = "skill_mining"
	case NodeTypeIdea:
		result.Confidence = 0.85
		result.Reasons = []string{"Contains idea markers"}
		result.SuggestedOp = "property_make"
	case NodeTypeContext:
		result.Confidence = 0.8
		result.Reasons = []string{"Contains context keywords"}
		result.SuggestedOp = "capability_mint"
	case NodeTypeCapability:
		result.Confidence = 0.75
		result.Reasons = []string{"Contains capability indicators"}
		result.SuggestedOp = "capability_mint"
	case NodeTypeSkill:
		result.Confidence = 0.7
		result.Reasons = []string{"Default classification"}
		result.SuggestedOp = "skill_mining"
	case NodeTypeProperty:
		result.Confidence = 0.8
		result.Reasons = []string{"Contains property markers"}
		result.SuggestedOp = "property_make"
	}

	return result
}

func (c *NodeClassifier) DetectTransformation(
	ctx context.Context,
	fromNode string,
	content string,
) (string, float64) {
	nodeType := c.Classify(ctx, content)

	switch fromNode {
	case "ErrorNode":
		if nodeType == NodeTypeError {
			return "skill_mining", 0.85
		}
		return "", 0
	case "ContextNode":
		if nodeType == NodeTypeContext || nodeType == NodeTypeCapability {
			return "capability_mint", 0.8
		}
		return "", 0
	case "IdeaNode":
		if nodeType == NodeTypeIdea || nodeType == NodeTypeProperty {
			return "property_make", 0.85
		}
		return "", 0
	default:
		return "", 0
	}
}
