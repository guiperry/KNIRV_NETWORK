package inferencer

import (
	"context"
	"testing"
)

// TestGenerateMirroredDegradesCleanly: when HEAD's VLM has no backend (503), the mirror
// must still produce HEART's candidate rather than erroring — a missing HEAD is not a
// failure, it is the normal "no vision backend" state.
func TestGenerateMirroredDegradesCleanly(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, true), headDown: true}
	c := d.client(t)
	mr, err := GenerateMirrored(context.Background(), c, "p", "", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Winner != "heart" || mr.Text != "from heart" || mr.Refuted {
		t.Fatalf("head down -> heart must win cleanly: %+v", mr)
	}
}

// TestGenerateMirroredWeightsByHasher: the arbiter honours HASHER-derived weights; the
// higher-weighted model wins, regardless of which one it is.
func TestGenerateMirroredWeightsByHasher(t *testing.T) {
	d := &fakeDaemon{registry: rows(true, true, true)}
	c := d.client(t)
	ctx := context.Background()

	mr, err := GenerateMirrored(ctx, c, "p", "", nil, "", map[string]float64{"heart": 0.4, "head": 0.95})
	if err != nil || mr.Winner != "head" || mr.Text != "from head" {
		t.Fatalf("head should win with the higher weight: %+v %v", mr, err)
	}

	mr, err = GenerateMirrored(ctx, c, "p", "", nil, "", map[string]float64{"heart": 0.95, "head": 0.4})
	if err != nil || mr.Winner != "heart" || mr.Text != "from heart" {
		t.Fatalf("heart should win with the higher weight: %+v %v", mr, err)
	}
}

// TestArbitrateDisqualifiesRefuted: a REFUTED candidate is never chosen.
func TestArbitrateDisqualifiesRefuted(t *testing.T) {
	cands := []mirrorCandidate{
		{model: "head", text: "h", refuted: true, weight: 1.0},
		{model: "heart", text: "he", weight: 0.5},
	}
	mr, err := arbitrate(cands, nil)
	if err != nil || mr.Winner != "heart" || mr.Text != "he" {
		t.Fatalf("refuted head must be disqualified: %+v %v", mr, err)
	}
}

// TestArbitrateNoSurvivor: when every candidate is refuted there is nothing to return.
func TestArbitrateNoSurvivor(t *testing.T) {
	if _, err := arbitrate([]mirrorCandidate{{model: "head", text: "h", refuted: true}}, nil); err == nil {
		t.Fatal("all refuted must error")
	}
}
