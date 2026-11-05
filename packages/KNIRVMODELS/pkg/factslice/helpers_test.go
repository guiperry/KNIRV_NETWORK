package factslice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/heartqwen"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// The frame registry runs in the SDK's Rust core, loaded digest-pinned. A locally
// built core is not published, so seed a throwaway data directory from KNIRVSDK/dist
// as the other packages' tests do. KNIRV_SDK_DATA_DIR, when set, wins.
func TestMain(m *testing.M) {
	cleanup := func() {}
	if os.Getenv("KNIRV_SDK_DATA_DIR") == "" {
		if matches, _ := filepath.Glob(filepath.Join("..", "..", "..", "KNIRVSDK", "dist", "*", "knirv_core.wasm")); len(matches) > 0 {
			sort.Slice(matches, func(i, j int) bool {
				a, _ := os.Stat(matches[i])
				b, _ := os.Stat(matches[j])
				return a.ModTime().After(b.ModTime())
			})
			if dir, err := os.MkdirTemp("", "factslice-core-"); err == nil {
				v := filepath.Base(filepath.Dir(matches[0]))
				if data, err := os.ReadFile(matches[0]); err == nil && os.MkdirAll(filepath.Join(dir, v), 0o700) == nil &&
					os.WriteFile(filepath.Join(dir, v, "knirv_core.wasm"), data, 0o600) == nil {
					os.Setenv("KNIRV_SDK_DATA_DIR", dir)
					cleanup = func() { os.RemoveAll(dir) }
				}
			}
		}
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}

// fakeLLM answers by stage. Hooks see the request, so a test can behave like a
// position-biased or hostile model.
type fakeLLM struct {
	mu      sync.Mutex
	extract func(user string) (string, error)
	judge   func(user string, call int) (string, error)
	calls   map[string][]string // stage -> user messages
}

func (f *fakeLLM) Complete(_ context.Context, r heartqwen.Request) (string, error) {
	f.mu.Lock()
	if f.calls == nil {
		f.calls = map[string][]string{}
	}
	user := ""
	if len(r.Messages) > 0 {
		user = r.Messages[len(r.Messages)-1].Content
	}
	f.calls[r.Frame.Stage] = append(f.calls[r.Frame.Stage], user)
	n := len(f.calls[r.Frame.Stage])
	f.mu.Unlock()
	switch r.Frame.Stage {
	case "hasher.extract":
		return f.extract(user)
	case "hasher.judge":
		return f.judge(user, n)
	}
	return "", nil
}

func (f *fakeLLM) count(stage string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls[stage])
}

var passageID = regexp.MustCompile(`<<<BEGIN PASSAGE ([0-9a-f]+)>>>`)

// shownIDs returns the passage ids shown in a judge prompt, in the order shown.
func shownIDs(user string) []string {
	var out []string
	for _, m := range passageID.FindAllStringSubmatch(user, -1) {
		out = append(out, m[1])
	}
	return out
}

func judgeJSON(verdict string, conf float64, spans ...string) string {
	b, _ := json.Marshal(map[string]any{"verdict": verdict, "confidence": conf, "spans": spans})
	return string(b)
}

func extractJSON(claims ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"claims": claims})
	return string(b)
}

func newJudge(llm heartqwen.LLM) *Judge { return &Judge{LLM: llm, Reg: frame.Default()} }

func newExtractor(llm heartqwen.LLM) *Extractor { return &Extractor{LLM: llm, Reg: frame.Default()} }

func hit(id, text string, tier evidence.Tier, snapshot string) evidence.Hit {
	return evidence.Hit{SpanID: id, Text: text, Tier: tier, Snapshot: snapshot, Title: "SECRET-TITLE-" + id, URL: "https://secret.example/" + id}
}

var _ = strings.TrimSpace
