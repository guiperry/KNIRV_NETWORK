package hands

import (
	"context"
	"strings"

	"github.com/guiperry/knirv-sdk-go/models/frame"

	"knirvmodels/internal/policy"
	"knirvmodels/pkg/heartqwen"
)

// Generator is the part of an Engine the tool loop needs. *Engine satisfies it;
// tests may supply a fake.
type Generator interface {
	GenerateText(prompt string, n int, temperature float32, topK int) (string, error)
}

// LLM adapts a Generator to heartqwen.LLM. The frame is the system message and
// every user turn (the task, then tool results) is concatenated behind it. The
// loop supplies stage.MaxOutput as MaxTokens, so generation length is bounded by
// the frame, never by a caller-chosen N.
type LLM struct{ Gen Generator }

// Complete implements heartqwen.LLM.
func (l LLM) Complete(_ context.Context, req heartqwen.Request) (string, error) {
	var b strings.Builder
	b.WriteString(req.Frame.System)
	for _, m := range req.Messages {
		if m.Role != "user" {
			continue
		}
		b.WriteString("\n\n")
		b.WriteString(m.Content)
	}
	return l.Gen.GenerateText(b.String(), req.MaxTokens, 0, 0)
}

// NewLoop returns the policy-gated tool loop for HANDS: the shared heartqwen
// engine with HANDS as the model. Every tool call goes through gate (T0/T1
// execute, T2 proposal, T3 denied and audited) and ex is only called after the
// gate allows it.
func NewLoop(gen Generator, reg *frame.Registry, gate *policy.Gate, ex heartqwen.Executor) *heartqwen.Engine {
	return &heartqwen.Engine{
		Reg:   reg,
		Facts: func() frame.LiveFacts { return frame.LiveFacts{Model: "hands", NodeMode: "local"} },
		LLM:   LLM{Gen: gen},
		Gate:  gate,
		Exec:  ex,
	}
}
