package validation

import (
	"strings"
	"testing"

	"KNIRVCHAIN/internal/types"
)

type mockGraphQueries struct {
	nodeStore *mockNodeStore
}

type mockNodeStore struct {
	nodes map[string]interface{}
}

func (mns *mockNodeStore) GetIdeaNode(id string) (*types.IdeaNode, error) {
	if node, ok := mns.nodes[id].(*types.IdeaNode); ok {
		return node, nil
	}
	return nil, nil
}

func (mns *mockNodeStore) ListNodesByType(nodeType string) ([]string, error) {
	var ids []string
	for id, node := range mns.nodes {
		switch nodeType {
		case "idea_node":
			if _, ok := node.(*types.IdeaNode); ok {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

type mockGraphQueriesWithStore struct {
	nodeStore *mockNodeStore
}

func (mgq *mockGraphQueriesWithStore) GetNodeStore() *mockNodeStore {
	return mgq.nodeStore
}

func TestNoveltyChecker_CalculateTextSimilarity(t *testing.T) {
	nc := &NoveltyChecker{}

	tests := []struct {
		name     string
		text1    string
		text2    string
		expected float64
	}{
		{
			name:     "identical texts",
			text1:    "hello world",
			text2:    "hello world",
			expected: 1.0,
		},
		{
			name:     "completely different",
			text1:    "hello",
			text2:    "goodbye",
			expected: 0.0,
		},
		{
			name:     "partial overlap",
			text1:    "hello world foo",
			text2:    "hello bar world",
			expected: 0.5, // 2/4 = 0.5 (unique words: hello,world,foo vs hello,bar,world)
		},
		{
			name:     "empty text1",
			text1:    "",
			text2:    "hello world",
			expected: 0.0,
		},
		{
			name:     "empty text2",
			text1:    "hello world",
			text2:    "",
			expected: 0.0,
		},
		{
			name:     "both empty",
			text1:    "",
			text2:    "",
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nc.calculateTextSimilarity(tt.text1, tt.text2)
			if result != tt.expected {
				t.Errorf("calculateTextSimilarity(%q, %q) = %v, want %v", tt.text1, tt.text2, result, tt.expected)
			}
		})
	}
}

func TestNoveltyChecker_TokenizeText(t *testing.T) {
	nc := &NoveltyChecker{}

	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			input:    "hello, world.",
			expected: []string{"hello", "world"},
		},
		{
			input:    "foo:bar;baz",
			expected: []string{"foobarbaz"},
		},
		{
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		result := nc.tokenizeText(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("tokenizeText(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("tokenizeText(%q)[%d] = %v, want %v", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestNoveltyChecker_CalculateNoveltyScore(t *testing.T) {
	nc := &NoveltyChecker{}

	tests := []struct {
		name         string
		similarIdeas []SimilarIdea
		dependencies []string
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "no similar ideas",
			similarIdeas: []SimilarIdea{},
			expectedMin:  1.0,
			expectedMax:  1.0,
		},
		{
			name: "low similarity",
			similarIdeas: []SimilarIdea{
				{Similarity: 0.1},
			},
			expectedMin: 0.8,
			expectedMax: 0.9,
		},
		{
			name: "high similarity",
			similarIdeas: []SimilarIdea{
				{Similarity: 0.9},
			},
			expectedMin: 0.0,
			expectedMax: 0.2,
		},
		{
			name: "multiple similar ideas",
			similarIdeas: []SimilarIdea{
				{Similarity: 0.3},
				{Similarity: 0.5},
				{Similarity: 0.7},
			},
			expectedMin: 0.2,
			expectedMax: 0.5,
		},
		{
			name: "with dependencies",
			similarIdeas: []SimilarIdea{
				{Similarity: 0.2},
			},
			dependencies: []string{"dep1", "dep2", "dep3", "dep4", "dep5"},
			expectedMin:  0.6, // 0.8 - 0.2 (max deduction of 0.2)
			expectedMax:  0.8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idea := &types.IdeaNode{
				Dependencies: tt.dependencies,
			}
			result := nc.calculateNoveltyScore(idea, tt.similarIdeas)
			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("calculateNoveltyScore() = %v, want between %v and %v", result, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestNoveltyChecker_GenerateIdeaContent(t *testing.T) {
	nc := &NoveltyChecker{}

	idea := &types.IdeaNode{
		ID:        "idea_001",
		IdeaType:  "insight",
		OriginNIM: "nim_001",
		Dependencies: []string{
			"dep_001",
			"dep_002",
		},
	}

	result := nc.generateIdeaContent(idea)

	if result == "" {
		t.Error("Expected non-empty content")
	}

	// The content should contain the idea type and origin NIM
	if !strings.Contains(result, "insight") {
		t.Errorf("Expected content to contain 'insight', got %q", result)
	}
	if !strings.Contains(result, "nim_001") {
		t.Errorf("Expected content to contain 'nim_001', got %q", result)
	}
}

func TestNoveltyChecker_GenerateIdeaQuery(t *testing.T) {
	nc := &NoveltyChecker{}

	idea := &types.IdeaNode{
		IdeaType:    "hypothesis",
		ContentHash: "abc123",
	}

	result := nc.generateIdeaQuery(idea)

	if result == "" {
		t.Error("Expected non-empty query")
	}

	if len(result) < len("hypothesis")+len("abc123") {
		t.Errorf("Query too short: %q", result)
	}
}

func TestNoveltyChecker_GetMaxSimilarity(t *testing.T) {
	nc := &NoveltyChecker{}

	similarIdeas := []SimilarIdea{
		{Similarity: 0.3},
		{Similarity: 0.9},
		{Similarity: 0.5},
		{Similarity: 0.1},
	}

	result := nc.getMaxSimilarity(similarIdeas)
	if result != 0.9 {
		t.Errorf("getMaxSimilarity() = %v, want 0.9", result)
	}

	emptyResult := nc.getMaxSimilarity([]SimilarIdea{})
	if emptyResult != 0.0 {
		t.Errorf("getMaxSimilarity(empty) = %v, want 0.0", emptyResult)
	}
}

func TestNoveltyChecker_GenerateReasoning(t *testing.T) {
	nc := &NoveltyChecker{}

	tests := []struct {
		name          string
		result        *NoveltyResult
		similarIdeas  []SimilarIdea
		shouldBeNovel bool
	}{
		{
			name: "novel with no similar ideas",
			result: &NoveltyResult{
				IsNovel: true,
				Score:   1.0,
			},
			similarIdeas:  []SimilarIdea{},
			shouldBeNovel: true,
		},
		{
			name: "novel with some similar ideas",
			result: &NoveltyResult{
				IsNovel: true,
				Score:   0.8,
			},
			similarIdeas: []SimilarIdea{
				{Similarity: 0.2},
			},
			shouldBeNovel: true,
		},
		{
			name: "not novel",
			result: &NoveltyResult{
				IsNovel: false,
				Score:   0.5,
			},
			similarIdeas: []SimilarIdea{
				{Similarity: 0.5},
				{Similarity: 0.6},
			},
			shouldBeNovel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reasoning := nc.generateReasoning(tt.result, tt.similarIdeas)
			if reasoning == "" {
				t.Error("Expected non-empty reasoning")
			}
		})
	}
}
