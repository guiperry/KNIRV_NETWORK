package supervision

import (
	"knirvhasher/pkg/hashing/schema"
	"testing"
	"time"
)

func episode() schema.SupervisionEpisode {
	e := schema.SupervisionEpisode{SchemaVersion: 1, TenantScope: schema.TenantScopeLocal, Workspace: schema.WorkspaceInfo{Fingerprint: "workspace"}, Task: schema.TaskInfo{Request: "review", Intent: schema.IntentReview, Risk: schema.RiskLow}, StateBefore: schema.StateBefore{Phase: schema.PhaseDevelop, Budget: map[string]int{"loop": 1}}, PolicyContext: []schema.PolicyRef{{ID: "p", ContentHash: "hash", Applicability: "required"}}, Trace: schema.TraceInfo{Agent: "agent"}, Decision: schema.DecisionLabel{RecommendedAction: "request_review", NextPhase: schema.PhaseDevelopReview, Confidence: .8}, Outcome: schema.OutcomeLabel{Verified: true}, Provenance: schema.ProvenanceInfo{Source: "cli", PolicyBundleHash: "policy", RedactionVersion: "v1", License: "operator-generated", CreatedAt: time.Unix(1, 0).UTC()}}
	e.EpisodeID = e.ComputeEpisodeID()
	return e
}
func manifest(e schema.SupervisionEpisode) *schema.DatasetManifest {
	m := &schema.DatasetManifest{SchemaVersion: 1, CreatedAt: time.Unix(2, 0).UTC(), PolicyBundleHash: "policy", RedactionVersion: "v1", EmbeddingModel: "test", EmbeddingVersion: "v1", EmbeddingDims: 2, TokenizerVersion: "v1", EpisodeCount: 1, EpisodeIDs: []string{e.EpisodeID}, SplitAssignments: map[string]schema.SplitName{e.EpisodeID: schema.SplitTrain}, SourceHashes: []string{e.EpisodeID}, License: "operator-generated"}
	m.ManifestID = m.ComputeManifestID()
	return m
}
func TestRetrieverKeepsManifestAndPolicySeparate(t *testing.T) {
	e := episode()
	m := manifest(e)
	idx, err := BuildIndexForManifest([]schema.SupervisionEpisode{e}, m, func(string) []float32 { return []float32{1, 0} })
	if err != nil {
		t.Fatal(err)
	}
	r := NewRetriever(idx, func(string) []float32 { return []float32{1, 0} }, "model")
	a, _ := r.Advise(SuperviseContext{CurrentPhase: "develop", PolicyBundleHash: "policy", PolicyRefs: []schema.PolicyRef{{ID: "p"}}, TaskSummary: e.Task})
	if a.Abstain || a.DatasetManifestHash != m.ManifestID || a.PolicyBundleHash != "policy" {
		t.Fatalf("bad advice: %#v", a)
	}
	a, _ = r.Advise(SuperviseContext{CurrentPhase: "develop", PolicyBundleHash: "other", PolicyRefs: []schema.PolicyRef{{ID: "p"}}, TaskSummary: e.Task})
	if !a.Abstain {
		t.Fatal("mismatched policy accepted")
	}
}
