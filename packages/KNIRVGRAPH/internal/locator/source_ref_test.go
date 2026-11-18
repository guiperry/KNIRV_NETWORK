package locator_test

import (
	"path/filepath"
	"strings"
	"testing"

	"KNIRVGRAPH/internal/locator"
	"KNIRVGRAPH/internal/storage"

	gproto "google.golang.org/protobuf/proto"
)

const (
	digestA = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	digestB = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	commit  = "0123456789abcdef0123456789abcdef01234567"
)

func fullSourceRef() *locator.SourceRef {
	return &locator.SourceRef{
		ProjectId:          "proj-42",
		SessionId:          "sess_7",
		CommitSha:          commit,
		Ref:                "main",
		Path:               "internal/app/app.go",
		LineStart:          10,
		LineEnd:            14,
		DiffDigest:         digestA,
		WorkspaceBaseHash:  digestB,
		WorkspaceFinalHash: digestA,
	}
}

// The source survives commit -> store -> reload, and a second system over the
// reopened store sees the same pointers.
func TestSourceRefRoundTripThroughStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")
	store, err := storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	sys := locator.NewLocatorSystem("peer", nil)
	if err := sys.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	want := fullSourceRef()
	node, err := sys.CreateErrorNode("crash", "segfault", map[string]interface{}{"exit_code": 1}, 2, want)
	if err != nil {
		t.Fatal(err)
	}
	if !gproto.Equal(node.Source, want) {
		t.Fatalf("created node source differs: %v", node.Source)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err = storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restarted := locator.NewLocatorSystem("peer", nil)
	if err := restarted.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	nodes := restarted.GetAllErrorNodes()
	if len(nodes) != 1 {
		t.Fatalf("want 1 node after restart, got %d", len(nodes))
	}
	if !gproto.Equal(nodes[0].Source, want) {
		t.Fatalf("reloaded source differs: %v", nodes[0].Source)
	}
}

// A node created without a source (every row written before this change) loads
// with no source, and its other fields are unaffected.
func TestNodeWithoutSourceRefLoadsWithNoSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")
	store, err := storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	sys := locator.NewLocatorSystem("peer", nil)
	if err := sys.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	if _, err := sys.CreateErrorNode("leak", "fd leak", map[string]interface{}{"file": "main.go"}, 3, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restarted := locator.NewLocatorSystem("peer", nil)
	if err := restarted.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	nodes := restarted.GetAllErrorNodes()
	if len(nodes) != 1 || nodes[0].Source != nil || nodes[0].ErrorType != "leak" {
		t.Fatalf("sourceless node must load unchanged: %+v", nodes)
	}
}

func TestValidateSourceRefAcceptsPointersAndDigests(t *testing.T) {
	cases := map[string]*locator.SourceRef{
		"nil":          nil,
		"empty":        {},
		"full":         fullSourceRef(),
		"digest only":  {DiffDigest: digestA},
		"no path":      {ProjectId: "p", CommitSha: "abcdef1"},
		"dotted tag":   {Ref: "v1.2.3"},
		"nested path":  {Path: "a/b/c_d-e.go", LineStart: 1, LineEnd: 1},
		"single line":  {Path: "x.go", LineStart: 5},
		"line range 0": {Path: "x.go"},
	}
	for name, src := range cases {
		if err := locator.ValidateSourceRef(src); err != nil {
			t.Errorf("%s: want accepted, got %v", name, err)
		}
	}
}

// Free text, code and diffs must not fit the pointer fields. Each case names
// the field that carries the bad value.
func TestValidateSourceRefRejectsFreeTextAndCode(t *testing.T) {
	long := strings.Repeat("a", 513)
	cases := map[string]*locator.SourceRef{
		"code in path":           {Path: "x.go\nfunc main() { os.Exit(1) }"},
		"space in path":          {Path: "my file.go"},
		"absolute path":          {Path: "/etc/passwd"},
		"dotdot path":            {Path: "a/../../secret"},
		"newline in project":     {ProjectId: "proj\n+func x()"},
		"space in project":       {ProjectId: "my project"},
		"bad commit":             {CommitSha: "not-a-sha; rm -rf"},
		"diff text as digest":    {DiffDigest: "diff --git a/x b/x"},
		"short digest":           {DiffDigest: "sha256:abcd"},
		"bare hex digest":        {WorkspaceBaseHash: digestA[len("sha256:"):]},
		"bad ref":                {Ref: "main branch"},
		"ref dotdot":             {Ref: "a..b"},
		"oversized path":         {Path: long},
		"lines without path":     {LineStart: 3, LineEnd: 4},
		"line_end without start": {Path: "x.go", LineEnd: 4},
		"reversed lines":         {Path: "x.go", LineStart: 9, LineEnd: 3},
		"line out of range":      {Path: "x.go", LineStart: 20_000_000, LineEnd: 20_000_000},
	}
	for name, src := range cases {
		if err := locator.ValidateSourceRef(src); err == nil {
			t.Errorf("%s: want rejected, got accepted: %v", name, src)
		}
	}
}

// A rejected source must not reach the store: no node, no suite, no row.
func TestCreateErrorNodeRejectsFreeTextSource(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	bad := &locator.SourceRef{Path: "x.go\nfunc leak() {}", DiffDigest: digestA}
	if _, err := sys.CreateErrorNode("leak", "fd leak", nil, 1, bad); err == nil {
		t.Fatal("a source carrying code must be refused")
	} else if strings.Contains(err.Error(), "func leak") {
		t.Fatalf("the error must not echo the rejected value: %v", err)
	}
	if n := len(sys.GetAllErrorNodes()); n != 0 {
		t.Fatalf("a refused source must not create a node, got %d", n)
	}
}

// The caller's source is copied, so later changes to it do not reach the node.
func TestCreateErrorNodeClonesSource(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	src := fullSourceRef()
	node, err := sys.CreateErrorNode("crash", "segfault", nil, 1, src)
	if err != nil {
		t.Fatal(err)
	}
	src.ProjectId = "changed"
	if node.Source.ProjectId != "proj-42" {
		t.Fatalf("node must keep its own copy, got %q", node.Source.ProjectId)
	}
}

// The public view keeps digests and drops every private pointer (plan D-12).
func TestPublicErrorNodeShowsDigestsOnly(t *testing.T) {
	sys := locator.NewLocatorSystem("peer", nil)
	node, err := sys.CreateErrorNode("crash", "segfault", nil, 1, fullSourceRef())
	if err != nil {
		t.Fatal(err)
	}
	pub := locator.PublicErrorNode(node)
	if pub == node {
		t.Fatal("public view must be a copy, not the stored node")
	}
	got := pub.Source
	if got.ProjectId != "" || got.SessionId != "" || got.Ref != "" || got.Path != "" ||
		got.LineStart != 0 || got.LineEnd != 0 || got.CommitSha != "" {
		t.Fatalf("public view leaks private fields: %+v", got)
	}
	if got.DiffDigest != digestA || got.WorkspaceBaseHash != digestB || got.WorkspaceFinalHash != digestA {
		t.Fatalf("public view must keep digests: %+v", got)
	}
	if node.Source.ProjectId != "proj-42" {
		t.Fatal("the stored node must not be modified by the public view")
	}
	if locator.PublicErrorNode(&locator.ErrorNode{Id: "x"}).Source != nil {
		t.Fatal("a sourceless node stays sourceless in the public view")
	}
}
