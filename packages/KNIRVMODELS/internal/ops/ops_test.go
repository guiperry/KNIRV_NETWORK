package ops

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

func full() *Set {
	one := func() any { return map[string]int{"n": 1} }
	f := func(context.Context, json.RawMessage) (any, error) { return 1, nil }
	return Standard(Deps{Manifest: one, Registry: one, Health: one, WASMRender: f, WASMVerify: f, RulesList: f, HeadStatus: f, HasherValidate: f, HasherVerify: f, IndexStatus: f, IndexSearch: f, PipelineStatus: f, PipelineLogs: f, ErrorsHistory: f, HeadReason: f})
}

// Dangerous implementation helpers may exist in the operator CLI, but they
// must never cross the frame boundary into the model-visible manifest.
func TestUnsafeToolsAreAbsentFromManifest(t *testing.T) {
	reg := frame.Default()
	for _, name := range reg.ToolNames() {
		if strings.HasPrefix(name, "driver.") || strings.HasPrefix(name, "asic.") ||
			name == "shutdown" || name == "chat.ExecuteSudoCommand" || name == "chat.ReadFile" {
			t.Errorf("unsafe tool exposed in manifest: %s", name)
		}
	}
}

// Every implemented tool must exist in the frame registry (the model can only
// be told about tools that are registered), and the T2 tools must stay
// unimplemented here: mutations are proposals, never executed by the loop.
func TestImplementedToolsAreRegistered(t *testing.T) {
	reg := frame.Default()
	for _, n := range full().Names() {
		tl, ok := reg.Tool(n)
		if !ok {
			t.Errorf("%s is implemented but the model is never told about it", n)
		}
		if tl.Tier == frame.T2Mutate {
			t.Errorf("%s is a mutating tool and must not be executable by the loop", n)
		}
	}
}

// The tools the frame advertises but this node cannot run. The list may only
// shrink: adding a tool to the frame without code shows up here.
func TestUnimplementedToolList(t *testing.T) {
	reg := frame.Default()
	s := full()
	var missing []string
	for _, n := range reg.ToolNames() {
		if tl, _ := reg.Tool(n); tl.Tier != frame.T2Mutate && !s.Has(n) {
			missing = append(missing, n)
		}
	}
	sort.Strings(missing)
	want := []string{}
	if len(missing) != len(want) {
		t.Fatalf("unimplemented read/compute tools changed:\n got %v\nwant %v", missing, want)
	}
	for i := range want {
		if missing[i] != want[i] {
			t.Fatalf("unimplemented read/compute tools changed:\n got %v\nwant %v", missing, want)
		}
	}
}

func TestExec(t *testing.T) {
	s := full()
	out, err := s.Exec(context.Background(), "manifest.get", nil)
	if err != nil || string(out) != `{"n":1}` {
		t.Fatalf("%s %v", out, err)
	}
	if _, err := s.Exec(context.Background(), "errors.history", json.RawMessage(`{"limit":1}`)); err != nil {
		t.Fatalf("errors.history: %v", err)
	}
	if _, err := New().Exec(context.Background(), "errors.history", nil); !errors.Is(err, ErrNotAvailable) {
		t.Fatalf("empty set: got %v", err)
	}
	if !full().Has("status") || Standard(Deps{}).Has("status") {
		t.Fatal("status needs registry and health")
	}
}
