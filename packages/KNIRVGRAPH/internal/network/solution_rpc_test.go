package network

import (
	"KNIRVGRAPH/internal/drq"
	"KNIRVGRAPH/internal/indexing"
	"KNIRVGRAPH/internal/nrv"
	"KNIRVGRAPH/internal/query"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type recordingApp struct{ recorded []*drq.Solution }

func (a *recordingApp) IsNetworkPaused() bool                             { return false }
func (a *recordingApp) IsProcessingEnabled() bool                         { return true }
func (a *recordingApp) GetIndexManager() *indexing.IndexManager           { return nil }
func (a *recordingApp) GetQueryProcessor() *query.QueryProcessor          { return nil }
func (a *recordingApp) SubsystemHealth(context.Context) map[string]error { return nil }
func (a *recordingApp) DRQStatus() drq.Status                             { return drq.Status{} }
func (a *recordingApp) RecordGradedSolution(sol *drq.Solution) (string, error) {
	a.recorded = append(a.recorded, sol)
	return "cluster-1", nil
}

func TestSwarmSolutionsAreGradedAgainstTheErrorNodeSuite(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	app := &recordingApp{}
	store := nrv.NewErrorTestSuiteStore(&testKV{m: map[string][]byte{}})
	rpc := &RPCServer{logger: zap.NewNop(), app: app}
	rpc.SetErrorTestSuiteStore(store)
	r := mux.NewRouter()
	r.HandleFunc("/economics/proof/solution", rpc.submitSolutionProof).Methods("POST")

	body := map[string]interface{}{"error_node_id": "e1", "solver_id": "swarm-7", "outputs": map[string]string{}}
	if rec := doJSON(t, r, "POST", "/economics/proof/solution", "", body); rec.Code != http.StatusUnauthorized {
		t.Fatalf("solution proofs move NRN and must be internal-only, got %d", rec.Code)
	}
	if rec := doJSON(t, r, "POST", "/economics/proof/solution", "secret", body); rec.Code != http.StatusConflict {
		t.Fatalf("an error node without tests cannot grade solutions, got %d", rec.Code)
	}

	for i := 1; i <= nrv.ErrorTestSuiteSize; i++ {
		if _, err := store.ContributeTest("e1", nrv.ErrorTestCase{ID: fmt.Sprintf("t%d", i), Input: "in", Expected: fmt.Sprintf("out %d", i), AuthorID: "dev"}); err != nil {
			t.Fatal(err)
		}
	}
	outputs := map[string]string{}
	for i := 1; i <= nrv.ErrorTestSuiteSize; i++ {
		outputs[fmt.Sprintf("t%d", i)] = fmt.Sprintf("out %d", i)
	}
	outputs["t8"] = "wrong"

	rec := doJSON(t, r, "POST", "/economics/proof/solution", "secret", map[string]interface{}{"error_node_id": "e1", "solver_id": "swarm-7", "outputs": outputs})
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != http.StatusOK || resp["validated"] != false || resp["reward_earned"] != nil {
		t.Fatalf("a 7/8 solution must be recorded unvalidated and unpaid: %d %v", rec.Code, resp)
	}
	if len(app.recorded) != 1 || app.recorded[0].Validated || app.recorded[0].ValidationScore != 7.0/8.0 {
		t.Fatalf("the failed solution must still reach DRQ, unvalidated: %+v", app.recorded)
	}

	outputs["t8"] = "out 8"
	rec = doJSON(t, r, "POST", "/economics/proof/solution", "secret", map[string]interface{}{"error_node_id": "e1", "solver_id": "swarm-7", "outputs": outputs})
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if rec.Code != http.StatusOK || resp["validated"] != true || resp["cluster_id"] != "cluster-1" {
		t.Fatalf("an 8/8 solution must be validated and recorded in its cluster: %d %v", rec.Code, resp)
	}
	last := app.recorded[len(app.recorded)-1]
	if !last.Validated || last.ErrorID != "e1" || last.AgentID != "swarm-7" {
		t.Fatalf("unexpected recorded solution: %+v", last)
	}
}
