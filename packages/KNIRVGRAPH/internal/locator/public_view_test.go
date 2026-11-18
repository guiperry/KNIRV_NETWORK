package locator_test

import (
	"encoding/json"
	"strings"
	"testing"

	"KNIRVGRAPH/internal/locator"

	"google.golang.org/protobuf/types/known/structpb"
)

const (
	q83Code    = "func leak() { secret := \"hunter2\" }"
	q83Project = "/home/user/private-repo"
)

// q83Context is the context shape errorscan builds, plus an absolute path
// hidden in another key, a nested absolute path and a list entry.
func q83Context() map[string]interface{} {
	return map[string]interface{}{
		"file_path":    "internal/app/app.go",
		"line_number":  float64(12),
		"exit_code":    float64(1),
		"code_context": q83Code,
		"project_path": q83Project,
		"log_note":     "failed reading " + q83Project + "/internal/app/app.go",
		"nested": map[string]interface{}{
			"project_path": q83Project,
			"ok":           "kept",
		},
		"paths": []interface{}{"/abs/one", "rel/two"},
	}
}

func jsonOf(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustStruct(t *testing.T, m map[string]interface{}) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(m)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The public view drops code_context and project_path, and reduces any absolute
// path wherever it sits in the context to its file name.
func TestPublicErrorNodeStripsCodeContextAndAbsolutePaths(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "segfault", q83Context(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	pub := locator.PublicErrorNode(node)
	ctx := pub.GetContext().AsMap()

	for _, gone := range []string{"code_context", "project_path"} {
		if _, ok := ctx[gone]; ok {
			t.Errorf("public context still has %q: %v", gone, ctx)
		}
	}
	if got := ctx["log_note"]; got != "failed reading app.go" {
		t.Errorf("log_note must be reduced to the file name: %v", got)
	}
	if _, ok := ctx["nested"].(map[string]interface{})["project_path"]; ok {
		t.Errorf("public nested context still has project_path: %v", ctx["nested"])
	}
	if got := ctx["nested"].(map[string]interface{})["ok"]; got != "kept" {
		t.Errorf("public nested context lost a safe value: %v", ctx["nested"])
	}
	if list := ctx["paths"].([]interface{}); len(list) != 2 || list[0] != "one" || list[1] != "rel/two" {
		t.Errorf("public list must reduce the absolute entry to its name: %v", list)
	}
	if ctx["file_path"] != "internal/app/app.go" || ctx["line_number"] != float64(12) || ctx["exit_code"] != float64(1) {
		t.Errorf("public context lost safe keys: %v", ctx)
	}
	body := jsonOf(t, pub)
	for _, private := range []string{q83Code, "hunter2", q83Project, "/home/user", "private-repo", "/abs/"} {
		if strings.Contains(body, private) {
			t.Errorf("public node leaks %q: %s", private, body)
		}
	}
}

// The owner-side record is not modified by the public view: code_context and
// project_path are still there for the owner and validators (Q83).
func TestStoredRecordKeepsCodeContextForOwner(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	if _, err := sys.CreateErrorNode("crash", "segfault", q83Context(), 1, nil); err != nil {
		t.Fatal(err)
	}
	nodes := sys.GetAllErrorNodes()
	if len(nodes) != 1 {
		t.Fatalf("want 1 node, got %d", len(nodes))
	}
	stored := nodes[0].GetContext().AsMap()
	if stored["code_context"] != q83Code || stored["project_path"] != q83Project {
		t.Fatalf("owner record must keep code_context and project_path: %v", stored)
	}
	_ = locator.PublicErrorNode(nodes[0])
	after := nodes[0].GetContext().AsMap()
	if after["code_context"] != q83Code || after["project_path"] != q83Project {
		t.Fatalf("public view must not modify the stored record: %v", after)
	}
}

// A sourceless node is redacted too: the context rule does not depend on a source.
func TestPublicErrorNodeRedactsSourcelessContext(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "segfault", q83Context(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if node.Source != nil {
		t.Fatal("test precondition: node must be sourceless")
	}
	if strings.Contains(jsonOf(t, locator.PublicErrorNode(node)), q83Code) {
		t.Fatal("sourceless public node leaks code_context")
	}
}

// pathProbes are the path shapes the validator found leaking. Each value must
// come out reduced to its name, with no directory part left.
var pathProbes = []struct{ in, want string }{
	{"cd;/home/u/secret", "cd;secret"},
	{"cat|/home/u/secret", "cat|secret"},
	{"x>/home/u/secret", "x>secret"},
	{"{/home/u/secret}", "{secret}"},
	{"failed in/home/u/secret", "failed in secret"},
	{`\\server\share\secret`, "secret"},
	{"~bob/secret", "secret"},
	{"~/notes/secret.txt", "secret.txt"},
	{"file:///home/u/secret", "secret"},
	{`C:\Users\me\x.txt`, "x.txt"},
	{"project_root: /home/u/repo", "project_root: repo"},
	{"compile failed at /home/u/x/main.go:3: undefined", "compile failed at main.go:3: undefined"},
	{"ratio 1 /2 items", "ratio 1 2 items"},
	{"internal/x.go", "internal/x.go"},
	// URLs are reduced too, to their last element; the scheme is kept.
	{"https://example.com/docs", "https:docs"},
	{"see http://x/y/z", "see http:z"},
}

// Every probe is reduced to its name; the directory part never survives.
func TestPublicStructReducesPathProbes(t *testing.T) {
	for _, p := range pathProbes {
		got := locator.PublicStruct(mustStruct(t, map[string]interface{}{"v": p.in})).AsMap()["v"]
		if got != p.want {
			t.Errorf("probe %q: got %q, want %q", p.in, got, p.want)
		}
		if s, _ := got.(string); strings.Contains(s, "/home/u") || strings.Contains(s, `server\share`) {
			t.Errorf("probe %q leaks a directory: %q", p.in, s)
		}
	}
}

// A path used as a map key drops the whole entry, at any depth.
func TestPublicStructDropsPathKeys(t *testing.T) {
	in := mustStruct(t, map[string]interface{}{
		"/home/u/secret.txt": "x",
		"meta":               map[string]interface{}{"/home/u/k": float64(1), "ok": "kept"},
		"file_path":          "internal/x.go",
	})
	out := locator.PublicStruct(in).AsMap()
	if _, ok := out["/home/u/secret.txt"]; ok {
		t.Errorf("path key kept: %v", out)
	}
	meta := out["meta"].(map[string]interface{})
	if _, ok := meta["/home/u/k"]; ok || meta["ok"] != "kept" {
		t.Errorf("nested path key kept or safe key lost: %v", meta)
	}
	if out["file_path"] != "internal/x.go" {
		t.Errorf("relative path lost: %v", out)
	}
}

// Metadata is redacted like context: a path in a metadata value or key does not
// reach the public node. (Validator mutant M4 survived before this test.)
func TestPublicErrorNodeRedactsMetadata(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "segfault", map[string]interface{}{"exit_code": float64(1)}, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	node.Metadata = mustStruct(t, map[string]interface{}{
		"origin":           "/home/u/secret/build.log",
		"/home/u/meta-key": "x",
		"nested":           map[string]interface{}{"dir": "~/private/x"},
		"safe":             "kept",
	})
	pub := locator.PublicErrorNode(node).Metadata.AsMap()
	body := jsonOf(t, pub)
	for _, private := range []string{"/home/u", "meta-key", "~/private"} {
		if strings.Contains(body, private) {
			t.Errorf("public metadata leaks %q: %s", private, body)
		}
	}
	if pub["safe"] != "kept" || pub["origin"] != "build.log" {
		t.Errorf("public metadata lost safe or reduced values: %v", pub)
	}
	if node.Metadata.AsMap()["origin"] != "/home/u/secret/build.log" {
		t.Errorf("public view must not modify the stored metadata")
	}
}

// description and failure_context lose their paths; the stored record keeps them.
func TestPublicErrorNodeReducesDescriptionAndFailureContext(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "compile failed at /home/u/x/main.go:3", nil, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	node.FailureContext = []byte("panic in /home/u/secret/run.go")
	pub := locator.PublicErrorNode(node)
	if pub.Description != "compile failed at main.go:3" {
		t.Errorf("description not reduced: %q", pub.Description)
	}
	if string(pub.FailureContext) != "panic in run.go" {
		t.Errorf("failure_context not reduced: %q", pub.FailureContext)
	}
	if string(node.FailureContext) != "panic in /home/u/secret/run.go" {
		t.Errorf("public view must not modify the stored failure_context")
	}
}

// Windows path shapes the validator found leaking (fix 3). A rooted backslash
// path, a drive letter glued to a word, and a path with spaces must each come
// out as its last element only.
var windowsPathProbes = []struct{ in, want string }{
	{`\Users\me\x.txt`, "x.txt"},
	{`foo\Users\me\x.txt`, "foo x.txt"},
	{`Users\me\x.txt`, "x.txt"},
	{`fooC:\Users\me\x.txt`, "foo x.txt"},
	{`fooC:/Users/me/x.txt`, "foo x.txt"},
	{`C:\Program Files\app\x.exe`, "x.exe"},
	{`C:\Program Files\app\x.exe now`, "x.exe now"},
	{`C:\\Users\\me\\x.txt`, "x.txt"},
	{`path=C:\Users\me\x.txt`, "path=x.txt"},
	{`/home/u/my projects/secret`, "secret"},
	{`/Users/me/My Docs/secret.txt`, "secret.txt"},
	{"path /home/u/x: no such", "path x: no such"},
	// Not paths: a backslash escape, a backslash inside a word, and a scheme
	// glued to a word.
	{`line a\nb`, `line a\nb`},
	{`foo\bar`, `foo\bar`},
	{"see https://x", "see https:x"},
	{"https://example.com/home/u/secret", "https:secret"},
}

// Every Windows probe is reduced; the directory part never survives.
func TestPublicStructReducesWindowsPathProbes(t *testing.T) {
	for _, p := range windowsPathProbes {
		got := locator.PublicStruct(mustStruct(t, map[string]interface{}{"v": p.in})).AsMap()["v"]
		if got != p.want {
			t.Errorf("probe %q: got %q, want %q", p.in, got, p.want)
		}
		if s, _ := got.(string); strings.Contains(s, "Users") || strings.Contains(s, "Program") || strings.Contains(s, "/home/u") {
			t.Errorf("probe %q leaks a directory: %q", p.in, s)
		}
	}
}

// Windows path keys are dropped like POSIX ones, including a drive glued to a
// word.
func TestPublicStructDropsWindowsPathKeys(t *testing.T) {
	keys := []string{`\Users\me`, `fooC:\Users\me`, `C:\Program Files\app`, `fooC:/Users/me`}
	for _, k := range keys {
		in := mustStruct(t, map[string]interface{}{
			k:      "v",
			"safe": map[string]interface{}{k: "v", "ok": "kept"},
		})
		body := jsonOf(t, locator.PublicStruct(in).AsMap())
		if strings.Contains(body, "Users") || strings.Contains(body, "Program") || strings.Contains(body, `"v"`) {
			t.Errorf("path key %q not dropped: %s", k, body)
		}
		if !strings.Contains(body, `"ok":"kept"`) {
			t.Errorf("safe sibling lost for key %q: %s", k, body)
		}
	}
}

// The top-level text fields the public view used to pass through unreduced
// (error_type, error_signature, domain, model_origin, resolved_by, cluster_id)
// are reduced; the stored record is unchanged.
func TestPublicErrorNodeReducesTopLevelTextFields(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "segfault", map[string]interface{}{"exit_code": float64(1)}, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	node.ErrorType = `crash at C:\Users\me\x.txt`
	node.ErrorSignature = "sig /home/u/secret/sig.bin"
	node.Domain = "dom /home/u/secret/dom"
	node.ModelOrigin = "~/private/model.gguf"
	node.ResolvedBy = "/home/u/secret/agent"
	node.ClusterId = "cl /home/u/secret/cluster"
	pub := locator.PublicErrorNode(node)
	got := map[string]string{
		"error_type":      pub.ErrorType,
		"error_signature": pub.ErrorSignature,
		"domain":          pub.Domain,
		"model_origin":    pub.ModelOrigin,
		"resolved_by":     pub.ResolvedBy,
		"cluster_id":      pub.ClusterId,
	}
	want := map[string]string{
		"error_type":      "crash at x.txt",
		"error_signature": "sig sig.bin",
		"domain":          "dom dom",
		"model_origin":    "model.gguf",
		"resolved_by":     "agent",
		"cluster_id":      "cl cluster",
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s: got %q, want %q", k, got[k], w)
		}
	}
	if node.ErrorType != `crash at C:\Users\me\x.txt` || node.ModelOrigin != "~/private/model.gguf" {
		t.Errorf("public view must not modify the stored top-level fields")
	}
}

// findResolutionPath puts the full error context in each resolution step's
// parameters, so the public view reduces every step: the context keys
// code_context and project_path are dropped at any depth, and the action and
// skill_id are reduced like other text. The stored steps keep the full context.
func TestPublicErrorNodeReducesResolutionParameters(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "segfault", q83Context(), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	step, err := locator.NewResolutionStep("invoke_skill", map[string]interface{}{
		"skill_id": "skill-1",
		"context":  q83Context(),
	}, "skill /home/u/secret/skill.md", 0.9)
	if err != nil {
		t.Fatal(err)
	}
	node.Resolution = &locator.ResolutionPath{Steps: []*locator.ResolutionStep{&step}}

	pub := locator.PublicErrorNode(node)
	if pub.GetResolution() == nil || len(pub.GetResolution().GetSteps()) != 1 {
		t.Fatalf("public resolution lost its step: %v", pub.GetResolution())
	}
	ps := pub.GetResolution().GetSteps()[0]
	if ps.GetSkillId() != "skill skill.md" {
		t.Errorf("public skill_id not reduced: %q", ps.GetSkillId())
	}
	pctx := ps.GetParameters().AsMap()["context"].(map[string]interface{})
	for _, gone := range []string{"code_context", "project_path"} {
		if _, ok := pctx[gone]; ok {
			t.Errorf("public resolution context still has %q: %v", gone, pctx)
		}
	}
	if pctx["file_path"] != "internal/app/app.go" {
		t.Errorf("public resolution context lost a safe key: %v", pctx)
	}
	if got := pctx["log_note"]; got != "failed reading app.go" {
		t.Errorf("public resolution log_note not reduced: %v", got)
	}
	body := jsonOf(t, pub.GetResolution())
	for _, private := range []string{q83Code, "hunter2", q83Project, "/home/user", "private-repo", "/home/u/secret"} {
		if strings.Contains(body, private) {
			t.Errorf("public resolution leaks %q: %s", private, body)
		}
	}

	// The stored record keeps the full context and the unreduced skill id.
	sctx := node.GetResolution().GetSteps()[0].GetParameters().AsMap()["context"].(map[string]interface{})
	if sctx["project_path"] != q83Project || sctx["code_context"] != q83Code {
		t.Errorf("public view must not modify the stored resolution context: %v", sctx)
	}
	if got := node.GetResolution().GetSteps()[0].GetSkillId(); got != "skill /home/u/secret/skill.md" {
		t.Errorf("public view must not modify the stored skill_id: %q", got)
	}
}
