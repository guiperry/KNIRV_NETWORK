package hands

import (
	"context"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"

	"knirvmodels/pkg/heartqwen"
)

// The production frame is the source of truth for HANDS' tools; these tests pin
// that StageTools reads it and refuses what the core must never expose.
func TestStageToolsFromProductionFrame(t *testing.T) {
	reg := frame.Default()
	names, err := StageTools(reg, StageAssistant)
	if err != nil {
		t.Fatalf("StageTools(%s): %v", StageAssistant, err)
	}
	if len(names) == 0 {
		t.Fatalf("%s exposes no tools", StageAssistant)
	}
	for _, n := range names {
		tool, ok := reg.Tool(n)
		if !ok {
			t.Fatalf("tool %q not in registry", n)
		}
		if tool.Tier == frame.T3Never {
			t.Fatalf("T3 tool %q reachable from %s", n, StageAssistant)
		}
	}
}

func TestStageToolsRejectsUnknownStageAndNilRegistry(t *testing.T) {
	if _, err := StageTools(frame.Default(), "no.such.stage"); err == nil {
		t.Fatal("unknown stage must error")
	}
	if _, err := StageTools(nil, StageAssistant); err == nil {
		t.Fatal("nil registry must error")
	}
}

// fakeGen records the prompt the loop hands the generator and returns a fixed
// completion, so the framing contract is checked without a model.
type fakeGen struct {
	prompt string
	n      int
}

func (f *fakeGen) GenerateText(prompt string, n int, _ float32, _ int) (string, error) {
	f.prompt, f.n = prompt, n
	return "ok", nil
}

func TestLLMFramesSystemThenUserTurnsOnly(t *testing.T) {
	g := &fakeGen{}
	req := heartqwen.Request{
		Frame:     frame.Frame{System: "SYSTEM"},
		Messages:  []heartqwen.Message{{Role: "user", Content: "task"}, {Role: "assistant", Content: "hidden"}, {Role: "user", Content: "tool result"}},
		MaxTokens: 64,
	}
	out, err := LLM{Gen: g}.Complete(context.Background(), req)
	if err != nil || out != "ok" {
		t.Fatalf("Complete = %q, %v", out, err)
	}
	if !strings.HasPrefix(g.prompt, "SYSTEM") || strings.Contains(g.prompt, "hidden") {
		t.Fatalf("prompt framing wrong: %q", g.prompt)
	}
	if !strings.Contains(g.prompt, "task") || !strings.Contains(g.prompt, "tool result") {
		t.Fatalf("user turns missing: %q", g.prompt)
	}
	if g.n != 64 {
		t.Fatalf("MaxTokens not honoured: %d", g.n)
	}
}
