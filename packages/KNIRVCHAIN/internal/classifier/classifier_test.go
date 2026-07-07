package classifier

import (
	"context"
	"testing"
)

func TestNodeClassifier_Classify(t *testing.T) {
	nc := NewNodeClassifier()

	tests := []struct {
		name    string
		content string
		want    NodeType
	}{
		{
			name:    "error content",
			content: "The system crashed with a stack trace error",
			want:    NodeTypeError,
		},
		{
			name:    "exception content",
			content: "NullPointerException in the main thread",
			want:    NodeTypeError,
		},
		{
			name:    "idea content",
			content: "What if we could use machine learning to predict user behavior",
			want:    NodeTypeIdea,
		},
		{
			name:    "innovation content",
			content: "This is a breakthrough innovative concept for the platform",
			want:    NodeTypeIdea,
		},
		{
			name:    "context content",
			content: "User prefers dark mode in their settings configuration",
			want:    NodeTypeContext,
		},
		{
			name:    "capability content",
			content: "This API can perform data processing and execute complex transformations",
			want:    NodeTypeCapability,
		},
		{
			name:    "skill content",
			content: "The LoRA adapter enables fine-tuning of the model for specialized knowledge",
			want:    NodeTypeSkill,
		},
		{
			name:    "property content",
			content: "The NFT represents a unique inference property created by the model",
			want:    NodeTypeProperty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			got := nc.Classify(ctx, tt.content)
			if got != tt.want {
				t.Errorf("Classify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNodeClassifier_ClassifyWithDetails(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	result := nc.ClassifyWithDetails(ctx, "The system failed with a critical error")

	if result.NodeType != NodeTypeError {
		t.Errorf("Expected NodeTypeError, got %v", result.NodeType)
	}
	if result.Confidence == 0 {
		t.Error("Expected non-zero confidence")
	}
	if len(result.Reasons) == 0 {
		t.Error("Expected non-empty reasons")
	}
	if result.SuggestedOp == "" {
		t.Error("Expected non-empty suggested operation")
	}
}

func TestNodeClassifier_DetectTransformation(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	tests := []struct {
		name     string
		fromNode string
		content  string
		wantOp   string
		wantConf float64
	}{
		{
			name:     "error to skill mining",
			fromNode: "ErrorNode",
			content:  "The system crashed with an exception error",
			wantOp:   "skill_mining",
			wantConf: 0.85,
		},
		{
			name:     "context to capability mint",
			fromNode: "ContextNode",
			content:  "User prefers dark mode in their environment settings",
			wantOp:   "capability_mint",
			wantConf: 0.8,
		},
		{
			name:     "idea to property make",
			fromNode: "IdeaNode",
			content:  "Imagine a breakthrough innovative hypothesis concept",
			wantOp:   "property_make",
			wantConf: 0.85,
		},
		{
			name:     "no transformation",
			fromNode: "SkillNode",
			content:  "Some content",
			wantOp:   "",
			wantConf: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOp, gotConf := nc.DetectTransformation(ctx, tt.fromNode, tt.content)
			if gotOp != tt.wantOp {
				t.Errorf("DetectTransformation() op = %v, want %v", gotOp, tt.wantOp)
			}
			if gotConf != tt.wantConf {
				t.Errorf("DetectTransformation() conf = %v, want %v", gotConf, tt.wantConf)
			}
		})
	}
}

func TestNodeClassifier_ErrorPatterns(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	errorContents := []string{
		"error occurred",
		"exception thrown",
		"failed to connect",
		"timeout occurred",
		"system panic",
		"critical crash",
		"bug in code",
		"broken pipe",
	}

	for _, content := range errorContents {
		result := nc.Classify(ctx, content)
		if result != NodeTypeError {
			t.Errorf("Expected NodeTypeError for '%s', got %v", content, result)
		}
	}
}

func TestNodeClassifier_IdeaMarkers(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	ideaContents := []string{
		"What if we could",
		"Could we implement",
		"My hypothesis is",
		"This is a concept",
		"Imagine if",
		"innovative solution",
		"breakthrough idea",
	}

	for _, content := range ideaContents {
		result := nc.Classify(ctx, content)
		if result != NodeTypeIdea {
			t.Errorf("Expected NodeTypeIdea for '%s', got %v", content, result)
		}
	}
}

func TestNodeClassifier_CapabilityKeywords(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	capabilityContent := "The API can perform data processing and execute complex transformations"

	result := nc.Classify(ctx, capabilityContent)
	if result != NodeTypeCapability {
		t.Errorf("Expected NodeTypeCapability, got %v", result)
	}
}

func TestNodeClassifier_SkillKeywords(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	skillContent := "The LoRA adapter enables fine-tuning of the model with specialized knowledge"

	result := nc.Classify(ctx, skillContent)
	if result != NodeTypeSkill {
		t.Errorf("Expected NodeTypeSkill, got %v", result)
	}
}

func TestNodeClassifier_PropertyKeywords(t *testing.T) {
	nc := NewNodeClassifier()
	ctx := context.Background()

	propertyContent := "This NFT represents a unique inference property generated by the model"

	result := nc.Classify(ctx, propertyContent)
	if result != NodeTypeProperty {
		t.Errorf("Expected NodeTypeProperty, got %v", result)
	}
}
