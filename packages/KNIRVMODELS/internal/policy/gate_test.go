package policy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

func call(stage, tool, args string, p Privilege) Call {
	return Call{Stage: stage, FrameHash: "h", Caller: "u1", Privilege: p, Tool: tool, Args: json.RawMessage(args)}
}

func newGate(t *testing.T) (*Gate, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool_audit.jsonl")
	a, err := OpenAudit(path)
	if err != nil {
		t.Fatal(err)
	}
	return NewGate(frame.Default(), a), path
}

func TestGateRules(t *testing.T) {
	g, _ := newGate(t)
	cases := []struct {
		name string
		c    Call
		want Verdict
		why  string
	}{
		{"allowed read", call("heart.chat", "index.search", `{"query":"x","k":3}`, User), Execute, ""},
		{"not in allow-list", call("heart.chat", "rules.list", `{"domain":"d"}`, User), Deny, "not allowed in stage"},
		{"invented tool", call("heart.chat", "shell.exec", `{}`, Admin), Deny, "not allowed in stage"},
		{"unknown stage", call("nope", "status", `{}`, Admin), Deny, "unknown stage"},
		{"bad type", call("heart.chat", "index.search", `{"query":3,"k":3}`, User), Deny, "want string"},
		{"over bound", call("heart.chat", "index.search", `{"query":"x","k":99}`, User), Deny, "want 0..10"},
		{"extra arg", call("heart.chat", "index.search", `{"query":"x","k":1,"path":"/etc/passwd"}`, User), Deny, "unexpected argument"},
		{"missing arg", call("heart.chat", "index.search", `{"query":"x"}`, User), Deny, "missing argument"},
		{"no-arg tool with args", call("heart.chat", "manifest.get", `{"x":1}`, User), Deny, "unexpected argument"},
		{"no-arg tool", call("heart.chat", "manifest.get", ``, User), Execute, ""},
		{"mutation by user", call("ops.assistant", "rules.add", `{"domain":"d","rule":"r"}`, User), Deny, "needs operator"},
		{"mutation by operator is a proposal", call("ops.assistant", "rules.add", `{"domain":"d","rule":"r"}`, Operator), Propose, "approval"},
	}
	for _, c := range cases {
		d := g.Check(nil, c.c)
		if d.Verdict != c.want || !strings.Contains(d.Reason, c.why) {
			t.Errorf("%s: got %s %q want %s %q", c.name, d.Verdict, d.Reason, c.want, c.why)
		}
	}
}

func TestSessionBudget(t *testing.T) {
	g, _ := newGate(t)
	s := &Session{}
	c := call("heart.chat", "manifest.get", ``, User)
	for i := 0; i < 4; i++ {
		if d := g.Check(s, c); d.Verdict != Execute {
			t.Fatalf("call %d: %+v", i, d)
		}
	}
	if d := g.Check(s, c); d.Verdict != Deny || !strings.Contains(d.Reason, "allows 4") {
		t.Fatalf("5th call: %+v", d)
	}
}

func TestInjectedCallInToolResultIsJustADeniedCall(t *testing.T) {
	g, _ := newGate(t)
	// A retrieved passage told the model to call a T3-style tool. It is not in
	// any registry, so the gate cannot be argued with.
	for _, tool := range []string{"asic.provision", "shutdown", "ExecuteSudoCommand", "ReadFile"} {
		if d := g.Check(nil, call("ops.assistant", tool, `{}`, Admin)); d.Verdict != Deny {
			t.Errorf("%s: %+v", tool, d)
		}
	}
}

func TestOnlyOpsAssistantCanEverPropose(t *testing.T) {
	g, _ := newGate(t)
	for _, id := range g.Reg.StageIDs() {
		s, _ := g.Reg.Stage(id)
		for _, tool := range s.Tools {
			d := g.Check(nil, call(id, tool, validArgsFor(t, g.Reg, tool), Admin))
			if d.Verdict == Propose && id != "ops.assistant" {
				t.Errorf("%s can propose %s", id, tool)
			}
		}
	}
}

func validArgsFor(t *testing.T, r *frame.Registry, tool string) string {
	tl, _ := r.Tool(tool)
	if len(tl.Args) == 0 {
		return ""
	}
	var d map[string]string
	_ = json.Unmarshal(tl.Args, &d)
	out := map[string]any{}
	for k, typ := range d {
		switch {
		case typ == "string":
			out[k] = "x"
		case typ == "bool":
			out[k] = true
		case typ == "object":
			out[k] = map[string]any{}
		case strings.HasPrefix(typ, "int"):
			out[k] = 1
		default:
			out[k] = strings.Split(typ, "|")[0]
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func TestEveryDefaultToolHasACheckableDescriptor(t *testing.T) {
	g, _ := newGate(t)
	for _, n := range g.Reg.ToolNames() {
		tl, _ := g.Reg.Tool(n)
		if err := validateArgs(tl.Args, json.RawMessage(validArgsFor(t, g.Reg, n))); err != nil {
			t.Errorf("%s: %v", n, err)
		}
	}
}

func TestAuditChain(t *testing.T) {
	g, path := newGate(t)
	g.Check(nil, call("heart.chat", "manifest.get", ``, User))
	g.Check(nil, call("heart.chat", "shell", `{}`, User))
	_ = g.Audit.RecordResult(call("heart.chat", "manifest.get", ``, User), []byte("ok"), "ok")
	n, _, err := VerifyChain(path)
	if err != nil || n != 3 {
		t.Fatalf("%d %v", n, err)
	}
	// Reopen continues the chain.
	a2, err := OpenAudit(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = a2.Append(Entry{Stage: "x", Outcome: "deny"})
	if n, _, err := VerifyChain(path); err != nil || n != 4 {
		t.Fatalf("%d %v", n, err)
	}
	// Tampering is detected, and a tampered log is not extended.
	raw, _ := os.ReadFile(path)
	bad := strings.Replace(string(raw), `"outcome":"deny"`, `"outcome":"execute"`, 1)
	_ = os.WriteFile(path, []byte(bad), 0o640)
	if _, _, err := VerifyChain(path); err == nil {
		t.Fatal("edit not detected")
	}
	if _, err := OpenAudit(path); err == nil {
		t.Fatal("tampered log reopened")
	}
}

func TestDeniedCallsAreAudited(t *testing.T) {
	g, path := newGate(t)
	g.Check(nil, call("heart.chat", "asic.provision", `{}`, Admin))
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"outcome":"deny"`) || !strings.Contains(string(raw), "asic.provision") {
		t.Fatalf("%s", raw)
	}
}
