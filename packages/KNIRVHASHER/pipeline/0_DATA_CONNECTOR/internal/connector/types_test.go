package connector

import "testing"

func TestNormalizeRow(t *testing.T) {
	tests := []struct {
		name string
		row  map[string]any
		want string
	}{
		{"text", map[string]any{"text": "hello"}, "hello"},
		{"instruction", map[string]any{"instruction": "do", "output": "it"}, "### Instruction:\ndo\n\n### Response:\nit"},
		{"input", map[string]any{"input": "in", "output": "out"}, "in\nout"},
		{"messages", map[string]any{"messages": []any{map[string]any{"content": "one"}, map[string]any{"content": "two"}}}, "one\ntwo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeRow(tt.row); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
