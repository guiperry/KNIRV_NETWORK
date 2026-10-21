// Package policy is the deterministic gate between a model's tool request and
// anything that executes. The model recommends; this code decides. Every
// decision, allowed or not, is appended to the audit chain.
package policy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// Privilege is the authority of the caller, taken from the backend auth
// context, never from model output.
type Privilege int

const (
	User Privilege = iota
	Operator
	Admin
)

func (p Privilege) String() string { return [...]string{"user", "operator", "admin"}[p] }

// Verdict is the gate's answer.
type Verdict string

const (
	Execute Verdict = "execute" // run the tool now
	Propose Verdict = "propose" // T2: return to the caller for approval; do not run
	Deny    Verdict = "deny"
)

// Call is one tool request from a model.
type Call struct {
	Stage     string
	FrameHash string
	Caller    string
	Privilege Privilege
	Tool      string
	Args      json.RawMessage
}

// Decision is the verdict plus why.
type Decision struct {
	Verdict Verdict
	Reason  string
}

// Gate checks calls against the frame registry.
type Gate struct {
	Reg *frame.Registry
	// MinPrivilege per tier. Defaults: T0, T1 user; T2 operator.
	MinPrivilege map[frame.Tier]Privilege
	Audit        *Audit
}

// NewGate returns a gate over reg writing to audit (which may be nil in tests).
func NewGate(reg *frame.Registry, audit *Audit) *Gate {
	return &Gate{Reg: reg, Audit: audit, MinPrivilege: map[frame.Tier]Privilege{
		frame.T0Read: User, frame.T1Compute: User, frame.T2Mutate: Operator,
	}}
}

// Session counts calls inside one model loop so a stage's tool budget holds.
type Session struct {
	mu    sync.Mutex
	calls int
}

// Check applies the five rules in order and audits the result. A nil session
// skips only the budget rule.
func (g *Gate) Check(s *Session, c Call) Decision {
	d := g.decide(s, c)
	if g.Audit != nil {
		_ = g.Audit.Append(Entry{Stage: c.Stage, FrameHash: c.FrameHash, Caller: c.Caller, Tool: c.Tool,
			ArgsHash: frame.Hash(string(c.Args)), Outcome: string(d.Verdict), Reason: d.Reason})
	}
	return d
}

func (g *Gate) decide(s *Session, c Call) Decision {
	deny := func(f string, a ...any) Decision { return Decision{Deny, fmt.Sprintf(f, a...)} }
	stage, ok := g.Reg.Stage(c.Stage)
	if !ok {
		return deny("unknown stage %q", c.Stage)
	}
	// 1. The tool is in this stage's allow-list. A tool the model invented, a
	// T3 operation, or another stage's tool all look the same here.
	if !g.Reg.Allows(c.Stage, c.Tool) {
		return deny("tool %q is not allowed in stage %s", c.Tool, c.Stage)
	}
	tool, _ := g.Reg.Tool(c.Tool)
	// 2. Arguments validate against the tool's schema.
	if err := validateArgs(tool.Args, c.Args); err != nil {
		return deny("bad arguments for %s: %v", c.Tool, err)
	}
	// 3. The caller's privilege permits the tier.
	if need, ok := g.MinPrivilege[tool.Tier]; !ok || c.Privilege < need {
		return deny("%s needs %s privilege; caller is %s", c.Tool, need, c.Privilege)
	}
	// 4. Budget.
	if s != nil {
		s.mu.Lock()
		s.calls++
		over := stage.MaxToolCalls > 0 && s.calls > stage.MaxToolCalls
		s.mu.Unlock()
		if over {
			return deny("stage %s allows %d tool calls", c.Stage, stage.MaxToolCalls)
		}
	}
	// 5. Mutating tools are never executed by the loop.
	if tool.Tier == frame.T2Mutate {
		return Decision{Propose, "mutating tool: needs operator approval"}
	}
	return Decision{Execute, ""}
}

// validateArgs checks args against the tool's descriptor: an object whose keys
// are exactly the descriptor's, each value matching its type word. Type words:
// string, bool, object, int, "int<=N", or "a|b|c" for a fixed set.
func validateArgs(desc, args json.RawMessage) error {
	var want map[string]string
	if len(desc) > 0 {
		if err := json.Unmarshal(desc, &want); err != nil {
			return fmt.Errorf("tool descriptor is not checkable: %w", err)
		}
	}
	var got map[string]json.RawMessage
	if len(args) > 0 && string(args) != "null" {
		if err := json.Unmarshal(args, &got); err != nil {
			return fmt.Errorf("arguments must be a JSON object")
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			return fmt.Errorf("unexpected argument %q", k)
		}
	}
	for k, typ := range want {
		v, ok := got[k]
		if !ok {
			return fmt.Errorf("missing argument %q", k)
		}
		if err := checkType(typ, v); err != nil {
			return fmt.Errorf("argument %q: %w", k, err)
		}
	}
	return nil
}

func checkType(typ string, v json.RawMessage) error {
	switch {
	case typ == "string":
		var s string
		if json.Unmarshal(v, &s) != nil {
			return fmt.Errorf("want string")
		}
	case typ == "bool":
		var b bool
		if json.Unmarshal(v, &b) != nil {
			return fmt.Errorf("want bool")
		}
	case typ == "object":
		var m map[string]any
		if json.Unmarshal(v, &m) != nil {
			return fmt.Errorf("want object")
		}
	case typ == "int" || strings.HasPrefix(typ, "int<="):
		var f float64
		if json.Unmarshal(v, &f) != nil || f != float64(int64(f)) {
			return fmt.Errorf("want integer")
		}
		if max, ok := strings.CutPrefix(typ, "int<="); ok {
			n, err := strconv.ParseInt(max, 10, 64)
			if err != nil {
				return fmt.Errorf("bad bound %q", max)
			}
			if int64(f) > n || f < 0 {
				return fmt.Errorf("want 0..%d", n)
			}
		}
	case strings.Contains(typ, "|"):
		var s string
		if json.Unmarshal(v, &s) != nil {
			return fmt.Errorf("want one of %s", typ)
		}
		for _, o := range strings.Split(typ, "|") {
			if s == o {
				return nil
			}
		}
		return fmt.Errorf("want one of %s", typ)
	default:
		return fmt.Errorf("unknown type word %q", typ)
	}
	return nil
}
