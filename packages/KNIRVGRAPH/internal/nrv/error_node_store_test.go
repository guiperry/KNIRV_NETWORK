package nrv_test

import (
	"path/filepath"
	"testing"

	"KNIRVGRAPH/internal/nrv"
	"KNIRVGRAPH/internal/storage"
)

// Error nodes and their test suites share one store, so a restart keeps both:
// previously the suites survived while the nodes they key on vanished.
func TestErrorNodesSurviveRestartWithTheirTestSuites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graph.db")

	store, err := storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	sys := nrv.NewNRVSystem("peer", nil)
	if err := sys.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}
	node, err := sys.CreateErrorNode("leak", "fd leak", map[string]interface{}{"file": "main.go"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	suites := nrv.NewErrorTestSuiteStore(store)
	if _, err := suites.ContributeTest(node.Id, nrv.ErrorTestCase{ID: "t1", Input: "in", Expected: "out", AuthorID: "dev"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// Restart: a fresh NRV system over the reopened store.
	store, err = storage.NewBluntDBStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	restarted := nrv.NewNRVSystem("peer", nil)
	if err := restarted.SetErrorNodeStore(store); err != nil {
		t.Fatal(err)
	}

	nodes := restarted.GetAllErrorNodes()
	if len(nodes) != 1 || nodes[0].Id != node.Id {
		t.Fatalf("error node must survive the restart, got %d nodes", len(nodes))
	}
	got := nodes[0]
	if got.ErrorType != "leak" || got.Description != "fd leak" || got.Severity != 3 ||
		nrv.ErrorContextMap(got)["file"] != "main.go" || !got.GetTimestamp().AsTime().Equal(node.GetTimestamp().AsTime()) {
		t.Fatalf("restored node differs: %+v", got)
	}
	if vectors, _ := restarted.ResolveTarget(node.Id); len(vectors) == 0 {
		t.Fatal("the restored node's resolution vector must be rebuilt")
	}
	if suite, err := nrv.NewErrorTestSuiteStore(store).Get(node.Id); err != nil || len(suite.Tests) != 1 {
		t.Fatalf("the node's test suite must survive too: %v", err)
	}
}
