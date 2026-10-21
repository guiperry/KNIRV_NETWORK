// Package ops implements the tools the model may call. The names and schemas
// come from the frame registry (what the model is told); this package supplies
// the code behind them. A tool the frame lists but this package does not
// implement fails with ErrNotAvailable, and a test pins that list so it only
// shrinks.
package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNotAvailable is returned for a tool the frame names that this node does not implement yet.
var ErrNotAvailable = errors.New("tool is not available on this node")

// Func runs one tool and returns its JSON result.
type Func func(ctx context.Context, args json.RawMessage) (any, error)

// Set is a table of implemented tools. It satisfies heartqwen.Executor.
type Set struct{ fns map[string]Func }

// New returns an empty set.
func New() *Set { return &Set{fns: map[string]Func{}} }

// Register adds a tool; names must already exist in the frame registry (checked by tests).
func (s *Set) Register(name string, f Func) *Set { s.fns[name] = f; return s }

// Has reports whether the tool is implemented.
func (s *Set) Has(name string) bool { _, ok := s.fns[name]; return ok }

// Names lists the implemented tools.
func (s *Set) Names() []string {
	out := make([]string, 0, len(s.fns))
	for n := range s.fns {
		out = append(out, n)
	}
	return out
}

// Exec runs a tool. The caller must already have passed the policy gate.
func (s *Set) Exec(ctx context.Context, tool string, args json.RawMessage) ([]byte, error) {
	f, ok := s.fns[tool]
	if !ok {
		return nil, fmt.Errorf("%s: %w", tool, ErrNotAvailable)
	}
	v, err := f(ctx, args)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// Deps are the daemon facilities the read-only tools report on. Any nil entry
// leaves its tool unimplemented.
type Deps struct {
	Manifest func() any
	Registry func() any
	Health   func() any
	// WASMRender and WASMVerify back wasm.render and wasm.verify.
	WASMRender, WASMVerify Func
	// RulesList, HeadStatus and HasherValidate back rules.list, head.status and hasher.validate.
	RulesList, HeadStatus, HasherValidate Func
	// IndexStatus and IndexSearch back index.status and index.search.
	IndexStatus, IndexSearch Func
}

// Standard builds the T0 tools the daemon can serve now.
func Standard(d Deps) *Set {
	s := New()
	wrap := func(f func() any) Func {
		return func(context.Context, json.RawMessage) (any, error) { return f(), nil }
	}
	if d.Manifest != nil {
		s.Register("manifest.get", wrap(d.Manifest))
	}
	if d.Registry != nil {
		s.Register("models.registry", wrap(d.Registry))
	}
	if d.Health != nil {
		s.Register("models.health", wrap(d.Health))
	}
	for name, f := range map[string]Func{"rules.list": d.RulesList, "head.status": d.HeadStatus, "hasher.validate": d.HasherValidate, "index.status": d.IndexStatus, "index.search": d.IndexSearch} {
		if f != nil {
			s.Register(name, f)
		}
	}
	if d.WASMRender != nil {
		s.Register("wasm.render", d.WASMRender)
	}
	if d.WASMVerify != nil {
		s.Register("wasm.verify", d.WASMVerify)
	}
	if d.Registry != nil && d.Health != nil {
		s.Register("status", func(context.Context, json.RawMessage) (any, error) {
			return map[string]any{"models": d.Registry(), "health": d.Health()}, nil
		})
	}
	return s
}
