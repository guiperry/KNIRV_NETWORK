package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"KNIRVGRAPH/internal/drq"
	"KNIRVGRAPH/internal/nrv"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// fakeErrorSource supplies a fixed set of ingested error nodes.
type fakeErrorSource struct {
	nodes []*nrv.ErrorNode
}

func (f *fakeErrorSource) GetAllErrorNodes() []*nrv.ErrorNode { return f.nodes }

// newTestRuntime builds a DRQ runtime over a fixed error source. It uses the
// hash-fallback embedding model (no service), which is deterministic, so
// identical failure contexts genuinely cluster together.
func newTestRuntime(t *testing.T, nodes []*nrv.ErrorNode) *drqRuntime {
	t.Helper()

	nrvSystem := nrv.NewNRVSystem("test-peer", nil)
	// The runtime is normally built against the app's embedding service. Here
	// the clustering manager is built with no service, which selects the
	// deterministic hash embedding — hermetic, and enough to prove that
	// identical failure contexts cluster together.
	embeddingModel := drq.NewEmbeddingModelWithService(drq.BERT_BASE, nil)
	clustering, err := drq.NewDRQClusterManager(embeddingModel, drq.DefaultSimilarityThreshold, drq.DefaultMaxClusterSize)
	if err != nil {
		t.Fatalf("NewDRQClusterManager: %v", err)
	}

	rt := &drqRuntime{
		clustering:     clustering,
		clusterManager: drq.NewClusterManager(drq.ClusterDeps{}),
		logger:         zap.NewNop(),
		errorSource:    &fakeErrorSource{nodes: nodes},
		ingested:       make(map[string]string),
		lastError:      make(map[string]string),
	}
	_ = nrvSystem
	return rt
}

// The loop must actually ingest errors and cluster them — this is the part that
// is fully live today.
func TestDRQStepIngestsAndClusters(t *testing.T) {
	nodes := []*nrv.ErrorNode{
		{Id: "e1", ErrorType: "runtime", Description: "nil pointer in handler", Severity: 2},
		{Id: "e2", ErrorType: "runtime", Description: "nil pointer in handler", Severity: 2},
		{Id: "e3", ErrorType: "runtime", Description: "nil pointer in handler", Severity: 2},
	}
	rt := newTestRuntime(t, nodes)

	rt.step(context.Background())

	if rt.ticks != 1 {
		t.Fatalf("ticks = %d, want 1", rt.ticks)
	}
	if len(rt.ingested) != 3 {
		t.Fatalf("ingested = %d, want 3", len(rt.ingested))
	}

	// Identical failure contexts must land in the same cluster.
	first := rt.ingested["e1"]
	if first == "" {
		t.Fatal("error e1 was not clustered")
	}
	for _, id := range []string{"e2", "e3"} {
		if rt.ingested[id] != first {
			t.Fatalf("error %s clustered into %q, want %q (identical contexts must cluster together)",
				id, rt.ingested[id], first)
		}
	}

	// And the lifecycle must be driving that cluster.
	if got := len(rt.clusterManager.TrackedClusterIDs()); got != 1 {
		t.Fatalf("tracked clusters = %d, want 1", got)
	}
}

// Errors are ingested once: a second tick must not re-cluster them.
func TestDRQStepIsIdempotentAcrossTicks(t *testing.T) {
	rt := newTestRuntime(t, []*nrv.ErrorNode{
		{Id: "e1", ErrorType: "runtime", Description: "boom", Severity: 1},
	})

	rt.step(context.Background())
	clustersAfterFirst := len(rt.clustering.Clusters())
	rt.step(context.Background())

	if rt.ticks != 2 {
		t.Fatalf("ticks = %d, want 2", rt.ticks)
	}
	if got := len(rt.clustering.Clusters()); got != clustersAfterFirst {
		t.Fatalf("clusters went from %d to %d across ticks; ingestion is not idempotent",
			clustersAfterFirst, got)
	}
	if len(rt.ingested) != 1 {
		t.Fatalf("ingested = %d, want 1", len(rt.ingested))
	}
}

// A converged cluster cannot advance without a training backend, and the status
// view must say so rather than reporting silent success.
func TestDRQStatusReportsBlockedStage(t *testing.T) {
	nodes := []*nrv.ErrorNode{
		{Id: "e1", ErrorType: "runtime", Description: "same failure", Severity: 1},
		{Id: "e2", ErrorType: "runtime", Description: "same failure", Severity: 1},
		{Id: "e3", ErrorType: "runtime", Description: "same failure", Severity: 1},
	}
	rt := newTestRuntime(t, nodes)
	// Age the cluster past the maturity threshold and give it a validated
	// solution so convergence is the only thing left to evaluate.
	rt.step(context.Background())

	for _, cluster := range rt.clustering.Clusters() {
		cluster.CreatedAt = cluster.CreatedAt.Add(-2 * time.Hour)
		cluster.Solutions = map[string][]*drq.Solution{
			"e1": {{SolutionID: "s1", ErrorID: "e1", Validated: true}},
		}
	}

	rt.step(context.Background())

	status := rt.Status()
	if !status.Enabled {
		t.Fatal("status must report the loop enabled")
	}
	if status.Ticks != 2 {
		t.Fatalf("ticks = %d, want 2", status.Ticks)
	}
	if status.Clusters != 1 {
		t.Fatalf("clusters = %d, want 1", status.Clusters)
	}

	// With no trainer backend, the lifecycle must be blocked and must say why.
	if len(status.LastErrors) == 0 {
		t.Fatal("expected the blocked lifecycle stage to be reported in status")
	}
	blocked := false
	for _, message := range status.LastErrors {
		if message != "" {
			blocked = true
		}
	}
	if !blocked {
		t.Fatal("expected a non-empty blocker message")
	}
	if status.ClustersDetail[0].Ready != true {
		t.Fatalf("cluster should be converged: %s", status.ClustersDetail[0].NotReadyReason)
	}
}

// The disabled state must be reported explicitly, never as a silent absence.
func TestDRQStatusReportsDisabled(t *testing.T) {
	app := &App{drqDisabledReason: "no embedding service"}
	status := app.DRQStatus()
	if status.Enabled {
		t.Fatal("a nil loop must report disabled")
	}
	if status.DisabledReason != "no embedding service" {
		t.Fatalf("disabled reason = %q", status.DisabledReason)
	}
}

// toDRQErrorNode must carry enough information for clustering and attribution.
func TestToDRQErrorNode(t *testing.T) {
	node := &nrv.ErrorNode{
		Id:          "e1",
		ErrorType:   "runtime",
		Description: "nil pointer",
		Context:     mustContext(t, map[string]interface{}{"resolved_by": "agent-7"}),
		Severity:    3,
	}
	converted := toDRQErrorNode(node)
	if converted == nil {
		t.Fatal("expected a converted node")
	}
	if converted.Id != "e1" {
		t.Fatalf("id = %q", converted.Id)
	}
	if converted.Domain != "runtime" {
		t.Fatalf("domain = %q, want the error type", converted.Domain)
	}
	if converted.Complexity != 3 {
		t.Fatalf("complexity = %d, want severity 3", converted.Complexity)
	}
	if converted.ResolvedBy != "agent-7" {
		t.Fatalf("resolved by = %q, want agent-7 from context", converted.ResolvedBy)
	}
	if len(converted.FailureContext) == 0 {
		t.Fatal("failure context must be populated: it is what gets embedded")
	}
	// The embedded text must distinguish different errors, not just be the raw
	// context map.
	if !strings.Contains(string(converted.FailureContext), "nil pointer") {
		t.Fatalf("failure context should include the description, got %s", converted.FailureContext)
	}

	if toDRQErrorNode(nil) != nil {
		t.Fatal("nil input must yield nil")
	}
	if toDRQErrorNode(&nrv.ErrorNode{}) != nil {
		t.Fatal("a node with no id must be rejected")
	}
}

// The loop must be constructible without the caller supplying an embedding
// service: mining needs embeddings, but it must not be silently switched off by
// the unrelated document-processing toggle that builds the app's service.
func TestNewDRQRuntimeBuildsItsOwnEmbeddingService(t *testing.T) {
	rt, err := newDRQRuntime(nrv.NewNRVSystem("test-peer", nil), nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("expected DRQ to build its own embedding service, got: %v", err)
	}
	if rt == nil {
		t.Fatal("expected a runtime")
	}
	if rt.clustering == nil {
		t.Fatal("expected a clustering manager")
	}
	// The similarity threshold must have been defaulted, not left at zero.
	if rt.clustering.SimilarityThreshold() <= 0 {
		t.Fatalf("similarity threshold = %v, want the default", rt.clustering.SimilarityThreshold())
	}
}

// A nil NRV system is a genuine construction failure and must be reported.
func TestNewDRQRuntimeRequiresNRVSystem(t *testing.T) {
	if _, err := newDRQRuntime(nil, nil, nil, zap.NewNop()); err == nil {
		t.Fatal("expected an error with no NRV system")
	}
}

// The loop must be reachable from a normally-constructed App: this is the
// end-to-end check that wiring exists rather than merely compiling.
func TestAppConstructionEnablesDRQLoop(t *testing.T) {
	app, err := NewApp(t.TempDir(), 18099, false)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	status := app.DRQStatus()
	if !status.Enabled {
		t.Fatalf("DRQ loop must be enabled on a normally-constructed App; disabled reason: %q",
			status.DisabledReason)
	}
	if status.SimilarityThreshold <= 0 {
		t.Fatalf("similarity threshold = %v, want the default", status.SimilarityThreshold)
	}
	// No errors have been ingested yet, so the loop should be idle and honest
	// about that rather than reporting phantom clusters.
	if status.Clusters != 0 {
		t.Fatalf("clusters = %d, want 0 before any error is ingested", status.Clusters)
	}
}

// Errors with no resolver attribution cannot become a mintable skill: there
// would be no owner to pay. The status must show the cluster stalling rather
// than the loop inventing an owner.
func TestClusterWithoutAttributionCannotMint(t *testing.T) {
	nodes := []*nrv.ErrorNode{
		{Id: "e1", ErrorType: "runtime", Description: "same", Severity: 1},
		{Id: "e2", ErrorType: "runtime", Description: "same", Severity: 1},
		{Id: "e3", ErrorType: "runtime", Description: "same", Severity: 1},
	}
	rt := newTestRuntime(t, nodes)
	rt.step(context.Background())

	for _, cluster := range rt.clustering.Clusters() {
		if len(cluster.AgentCounts) != 0 {
			t.Fatalf("expected no agent attribution, got %v", cluster.AgentCounts)
		}
	}
}

// mustContext builds the protobuf Struct the canonical node carries, for tests
// that construct nodes directly. It fails the test rather than returning an
// error, since a malformed context in a fixture is a test bug.
func mustContext(t *testing.T, values map[string]interface{}) *structpb.Struct {
	t.Helper()
	fields, err := nrv.NewErrorContextStruct(values)
	if err != nil {
		t.Fatalf("build error context: %v", err)
	}
	return fields
}
