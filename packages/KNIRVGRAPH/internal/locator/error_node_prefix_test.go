package locator_test

import (
	"path/filepath"
	"testing"

	"KNIRVGRAPH/internal/locator"
	"KNIRVGRAPH/internal/storage"

	"google.golang.org/protobuf/encoding/protojson"
)

// New error nodes are written under graph:error_nodes:, and rows still under the
// legacy nrv:error_nodes: prefix load as a fallback until the migration moves them.
func TestErrorNodesWriteNewPrefixAndReadLegacyFallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")
	store, err := storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	// A legacy row written before the rename, with no current-prefix twin.
	legacy := &locator.ErrorNode{Id: "legacy-1", ErrorType: "leak", Description: "old", Severity: 1}
	raw, err := protojson.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put([]byte("nrv:error_nodes:legacy-1"), raw); err != nil {
		t.Fatal(err)
	}

	sys := locator.NewLocatorSystem("peer", nil)
	if err := sys.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	if n := len(sys.GetAllErrorNodes()); n != 1 {
		t.Fatalf("legacy row must load through the fallback, got %d nodes", n)
	}

	fresh, err := sys.CreateErrorNode("leak", "new", map[string]interface{}{"file": "a.go"}, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := store.Has([]byte("graph:error_nodes:" + fresh.Id)); !ok {
		t.Fatal("new error node must be written under graph:error_nodes:")
	}
	if ok, _ := store.Has([]byte("nrv:error_nodes:" + fresh.Id)); ok {
		t.Fatal("nothing may write under the legacy prefix")
	}
}

// When a row exists under both prefixes, the current-prefix row wins.
func TestCurrentPrefixWinsOverLegacyDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")
	store, err := storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	oldRaw, _ := protojson.Marshal(&locator.ErrorNode{Id: "dup", ErrorType: "leak", Description: "legacy copy", Severity: 1})
	newRaw, _ := protojson.Marshal(&locator.ErrorNode{Id: "dup", ErrorType: "leak", Description: "current copy", Severity: 1})
	if err := store.Put([]byte("nrv:error_nodes:dup"), oldRaw); err != nil {
		t.Fatal(err)
	}
	if err := store.Put([]byte("graph:error_nodes:dup"), newRaw); err != nil {
		t.Fatal(err)
	}

	sys := locator.NewLocatorSystem("peer", nil)
	if err := sys.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	nodes := sys.GetAllErrorNodes()
	if len(nodes) != 1 || nodes[0].Description != "current copy" {
		t.Fatalf("current-prefix row must win, got %+v", nodes)
	}
}
