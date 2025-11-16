package deep_prompts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEliteProblemSolverPrompt(t *testing.T) {
	tests := []struct {
		name        string
		codeContent string
		expected    string
	}{
		{
			name:        "basic code content",
			codeContent: "func test() { return nil }",
			expected:    EliteProblemSolverPrompt,
		},
		{
			name:        "empty code content",
			codeContent: "",
			expected:    EliteProblemSolverPrompt,
		},
		{
			name:        "complex code content",
			codeContent: `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}`,
			expected: EliteProblemSolverPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEliteProblemSolverPrompt(tt.codeContent)

			// Verify the prompt template is used
			assert.Contains(t, result, "#CONTEXT:")
			assert.Contains(t, result, "#ROLE:")
			assert.Contains(t, result, "#RESPONSE GUIDELINES:")
			assert.Contains(t, result, "#TASK CRITERIA:")
			assert.Contains(t, result, "#ERROR ANALYSIS:")

			// Verify code content is embedded
			assert.Contains(t, result, tt.codeContent)
			assert.Contains(t, result, "--- CODE WITH RECURRING ERROR ---")
			assert.Contains(t, result, "--- END CODE ---")
		})
	}
}

func TestGetSystemsArchitectThinkingPrompt(t *testing.T) {
	tests := []struct {
		name         string
		knowledgeGraph string
		expected      string
	}{
		{
			name:         "basic knowledge graph",
			knowledgeGraph: "entity: ComponentA\nrelations: depends_on ComponentB",
			expected:     SystemsArchitectThinkingPrompt,
		},
		{
			name:         "empty knowledge graph",
			knowledgeGraph: "",
			expected:     SystemsArchitectThinkingPrompt,
		},
		{
			name: "complex knowledge graph",
			knowledgeGraph: `{
  "entities": [
    {
      "name": "UserService",
      "type": "service",
      "observations": ["handles user authentication"]
    }
  ],
  "relations": [
    {
      "from": "UserService",
      "to": "Database",
      "type": "depends_on"
    }
  ]
}`,
			expected: SystemsArchitectThinkingPrompt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSystemsArchitectThinkingPrompt(tt.knowledgeGraph)

			// Verify the prompt template is used
			assert.Contains(t, result, "SYSTEMS THINKING ARCHITECT FOR CODE ANALYSIS")
			assert.Contains(t, result, "#CONTEXT:")
			assert.Contains(t, result, "#ROLE:")
			assert.Contains(t, result, "#YOUR MISSION:")
			assert.Contains(t, result, "#ANALYSIS FRAMEWORK:")

			// Verify knowledge graph is embedded
			assert.Contains(t, result, tt.knowledgeGraph)
			assert.Contains(t, result, "--- KNOWLEDGE GRAPH ---")
			assert.Contains(t, result, "--- END KNOWLEDGE GRAPH ---")
		})
	}
}

func TestFormatPrompt(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "single string argument",
			format:   "Hello %s!",
			args:     []interface{}{"World"},
			expected: "Hello World!",
		},
		{
			name:     "multiple string arguments",
			format:   "User: %s, Age: %s",
			args:     []interface{}{"John", "25"},
			expected: "User: John, Age: 25",
		},
		{
			name:     "no arguments",
			format:   "No placeholders here",
			args:     []interface{}{},
			expected: "No placeholders here",
		},
		{
			name:     "more args than placeholders",
			format:   "Hello %s",
			args:     []interface{}{"World", "Extra"},
			expected: "Hello World",
		},
		{
			name:     "fewer args than placeholders",
			format:   "Hello %s, %s",
			args:     []interface{}{"World"},
			expected: "Hello World, %s",
		},
		{
			name:     "non-string argument (should be ignored)",
			format:   "Count: %s",
			args:     []interface{}{123},
			expected: "Count: %s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPrompt(tt.format, tt.args...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSprintf(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "single replacement",
			format:   "Hello %s!",
			args:     []interface{}{"World"},
			expected: "Hello World!",
		},
		{
			name:     "multiple replacements",
			format:   "%s says %s to %s",
			args:     []interface{}{"John", "hello", "Jane"},
			expected: "John says hello to Jane",
		},
		{
			name:     "no replacements needed",
			format:   "No placeholders",
			args:     []interface{}{},
			expected: "No placeholders",
		},
		{
			name:     "more args than placeholders",
			format:   "Hello %s",
			args:     []interface{}{"World", "ignored"},
			expected: "Hello World",
		},
		{
			name:     "replacement in middle",
			format:   "Start %s end",
			args:     []interface{}{"middle"},
			expected: "Start middle end",
		},
		{
			name:     "multiple same placeholder",
			format:   "%s and %s",
			args:     []interface{}{"first", "second"},
			expected: "first and second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sprintf(tt.format, tt.args...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceFirst(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		old      string
		new      string
		expected string
	}{
		{
			name:     "basic replacement",
			s:        "hello world",
			old:      "world",
			new:      "golang",
			expected: "hello golang",
		},
		{
			name:     "replace at beginning",
			s:        "world hello",
			old:      "world",
			new:      "golang",
			expected: "golang hello",
		},
		{
			name:     "replace in middle",
			s:        "hello world test",
			old:      "world",
			new:      "golang",
			expected: "hello golang test",
		},
		{
			name:     "no occurrence",
			s:        "hello universe",
			old:      "world",
			new:      "golang",
			expected: "hello universe",
		},
		{
			name:     "empty old string",
			s:        "hello world",
			old:      "",
			new:      "golang",
			expected: "golanghello world",
		},
		{
			name:     "empty new string",
			s:        "hello world",
			old:      "world",
			new:      "",
			expected: "hello ",
		},
		{
			name:     "multiple occurrences - only first replaced",
			s:        "world world world",
			old:      "world",
			new:      "golang",
			expected: "golang world world",
		},
		{
			name:     "special characters",
			s:        "hello\nworld",
			old:      "\n",
			new:      " ",
			expected: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceFirst(tt.s, tt.old, tt.new)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexOf(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "found at beginning",
			s:        "hello world",
			substr:   "hello",
			expected: 0,
		},
		{
			name:     "found in middle",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "found at end",
			s:        "hello world",
			substr:   "world",
			expected: 6,
		},
		{
			name:     "not found",
			s:        "hello world",
			substr:   "golang",
			expected: -1,
		},
		{
			name:     "empty substring",
			s:        "hello world",
			substr:   "",
			expected: 0,
		},
		{
			name:     "substring longer than string",
			s:        "hi",
			substr:   "hello",
			expected: -1,
		},
		{
			name:     "multiple occurrences - returns first",
			s:        "test test test",
			substr:   "test",
			expected: 0,
		},
		{
			name:     "case sensitive",
			s:        "Hello World",
			substr:   "hello",
			expected: -1,
		},
		{
			name:     "unicode characters",
			s:        "héllo wörld",
			substr:   "wörld",
			expected: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indexOf(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPromptConstants tests that the prompt constants contain expected content
func TestPromptConstants(t *testing.T) {
	// Test EliteProblemSolverPrompt
	assert.Contains(t, EliteProblemSolverPrompt, "#CONTEXT:")
	assert.Contains(t, EliteProblemSolverPrompt, "#ROLE:")
	assert.Contains(t, EliteProblemSolverPrompt, "#RESPONSE GUIDELINES:")
	assert.Contains(t, EliteProblemSolverPrompt, "#TASK CRITERIA:")
	assert.Contains(t, EliteProblemSolverPrompt, "#ERROR ANALYSIS:")
	assert.Contains(t, EliteProblemSolverPrompt, "%s")

	// Test SystemsArchitectThinkingPrompt
	assert.Contains(t, SystemsArchitectThinkingPrompt, "SYSTEMS THINKING ARCHITECT")
	assert.Contains(t, SystemsArchitectThinkingPrompt, "#CONTEXT:")
	assert.Contains(t, SystemsArchitectThinkingPrompt, "#ROLE:")
	assert.Contains(t, SystemsArchitectThinkingPrompt, "#YOUR MISSION:")
	assert.Contains(t, SystemsArchitectThinkingPrompt, "#ANALYSIS FRAMEWORK:")
	assert.Contains(t, SystemsArchitectThinkingPrompt, "%s")
}

// Benchmark tests for performance
func BenchmarkGetEliteProblemSolverPrompt(b *testing.B) {
	codeContent := `func test() {
		if err != nil {
			return err
		}
		return nil
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetEliteProblemSolverPrompt(codeContent)
	}
}

func BenchmarkGetSystemsArchitectThinkingPrompt(b *testing.B) {
	knowledgeGraph := `{"entities": [{"name": "Test"}]}`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetSystemsArchitectThinkingPrompt(knowledgeGraph)
	}
}

func BenchmarkSprintf(b *testing.B) {
	format := "Hello %s, welcome to %s!"
	args := []interface{}{"User", "System"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sprintf(format, args...)
	}
}

func BenchmarkReplaceFirst(b *testing.B) {
	s := "This is a long string with multiple words and this word appears multiple times"
	old := "word"
	new := "term"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		replaceFirst(s, old, new)
	}
}

func BenchmarkIndexOf(b *testing.B) {
	s := "This is a long string with multiple words and this word appears multiple times"
	substr := "word"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexOf(s, substr)
	}
}