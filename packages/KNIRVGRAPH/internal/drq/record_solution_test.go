package drq

import (
	"context"
	"testing"
)

func TestRecordedSolutionsAttachOnTheDRQGoroutineAndOnlyValidatedCredit(t *testing.T) {
	cm := NewClusterManager(ClusterDeps{})

	// A solution for an error that isn't clustered yet is queued, not lost.
	clusterID, err := cm.RecordSolution(&Solution{SolutionID: "s0", ErrorID: "e1", AgentID: "early", Validated: true})
	if err != nil || clusterID != "" {
		t.Fatalf("expected a queued solution, got %q %v", clusterID, err)
	}

	cluster := &ErrorCluster{ClusterID: "c1", Status: CLUSTER_ACTIVE, Errors: []*ErrorNode{{Id: "e1"}}}
	if err := cm.Track(cluster); err != nil {
		t.Fatal(err)
	}
	if id, _ := cm.RecordSolution(&Solution{SolutionID: "s1", ErrorID: "e1", AgentID: "swarm-a", Validated: true, ValidationScore: 1}); id != "c1" {
		t.Fatalf("expected the tracked cluster to be reported, got %q", id)
	}
	cm.RecordSolution(&Solution{SolutionID: "s2", ErrorID: "e1", AgentID: "swarm-b", Validated: false, ValidationScore: 0.5})

	// Nothing mutates the cluster until the DRQ loop processes it.
	if len(cluster.Solutions) != 0 {
		t.Fatal("RecordSolution must not mutate a live cluster off the DRQ goroutine")
	}
	_ = cm.ProcessCluster(context.Background(), "c1") // may not converge; attachment happens first

	if got := cm.countValidatedSolutions(cluster); got != 2 {
		t.Fatalf("expected 2 validated solutions (early + swarm-a), got %d", got)
	}
	if cluster.AgentCounts["swarm-a"] != 1 || cluster.AgentCounts["early"] != 1 || cluster.AgentCounts["swarm-b"] != 0 {
		t.Fatalf("only validated solutions may credit their agent: %v", cluster.AgentCounts)
	}
	if rate := cm.calculateValidationRate(cluster); rate < 0.66 || rate > 0.67 {
		t.Fatalf("the failed solution must count against the validation rate, got %v", rate)
	}
}
