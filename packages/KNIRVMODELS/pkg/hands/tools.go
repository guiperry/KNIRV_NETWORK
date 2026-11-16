package hands

import (
	"fmt"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// StageAssistant is the frame stage HANDS runs its tool loop under. Its tool list
// is owned by the Rust core (core/data/frame_defaults.json); HANDS does not keep
// a second allowlist.
const StageAssistant = "hands.assistant"

// StageTools returns the tool names the given stage may request, after checking
// them against the registry. An unknown tool or a T3 tool is an error: T3 is never
// exposed to a model and the core refuses to register one, so reaching this check
// with a T3 tool means the frame and the core disagree.
func StageTools(reg *frame.Registry, stage string) ([]string, error) {
	if reg == nil {
		return nil, fmt.Errorf("hands tools: no frame registry")
	}
	s, ok := reg.Stage(stage)
	if !ok {
		return nil, fmt.Errorf("hands tools: unknown stage %q", stage)
	}
	for _, name := range s.Tools {
		t, ok := reg.Tool(name)
		if !ok {
			return nil, fmt.Errorf("hands tools: stage %q names unknown tool %q", stage, name)
		}
		if t.Tier == frame.T3Never {
			return nil, fmt.Errorf("hands tools: stage %q exposes T3 tool %q", stage, name)
		}
	}
	return append([]string(nil), s.Tools...), nil
}
