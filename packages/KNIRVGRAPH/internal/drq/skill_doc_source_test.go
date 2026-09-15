package drq

import (
	"errors"
	"testing"

	"KNIRVGRAPH/internal/nrv"
)

var errLookupBoom = errors.New("skill registry unavailable")

// The uLoRA corpus needs each member error's validated fix. MintSkillTower
// already records resolves_errors/description as skill attributes, so the doc
// source reads those back rather than introducing new state.
func TestTowerSkillDocSourceReadsMintedFix(t *testing.T) {
	store := &fakeTowerStore{records: map[string]SkillRecord{
		"e1": {SkillID: "skill-1", Description: "add a bounds check", ResolvesErrors: []string{"e1"}},
	}}
	source := NewTowerSkillDocSource(store)

	doc, err := source.SkillDocForError("e1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if doc != "add a bounds check" {
		t.Fatalf("doc = %q, want the minted fix", doc)
	}

	// A member with no minted skill is skipped, not failed.
	doc, err = source.SkillDocForError("e2")
	if err != nil {
		t.Fatalf("absent member must not error: %v", err)
	}
	if doc != "" {
		t.Fatalf("doc = %q, want empty for an unresolved member", doc)
	}
}

// A failed lookup must surface, so a broken store cannot silently shrink the
// corpus the adapter is trained on.
func TestTowerSkillDocSourcePropagatesLookupFailure(t *testing.T) {
	source := NewTowerSkillDocSource(&fakeTowerStore{lookupErr: errLookupBoom})
	if _, err := source.SkillDocForError("e1"); err == nil {
		t.Fatal("a failing lookup must be reported")
	}
}

// And the corpus builder must refuse rather than train on a shrunken corpus when
// the store itself is broken.
func TestGatherClusterDatasetsPropagatesLookupFailure(t *testing.T) {
	source := NewTowerSkillDocSource(&fakeTowerStore{lookupErr: errLookupBoom})
	if _, _, err := gatherClusterDatasets(corpusCluster(), source); err == nil {
		t.Fatal("gather must fail when the doc lookup fails")
	}
}

// The real NRV-backed adapter is exercised end to end: mint a skill through the
// tower store, then read it back as a corpus fix.
func TestNRVSkillTowerStoreRoundTripFeedsCorpus(t *testing.T) {
	system := nrv.NewNRVSystem("test-peer", nil)
	store := NewNRVSkillTowerStore(system)

	// Nothing minted yet: the member is skipped rather than failed.
	source := NewTowerSkillDocSource(store)
	if doc, err := source.SkillDocForError("err-1"); err != nil || doc != "" {
		t.Fatalf("expected an empty fix before minting, got %q err=%v", doc, err)
	}

	// Mint through the same path MintSkillTower uses.
	client := NewKNIRVGRAPHClient(store)
	skill := &SkillNode{
		ID:             "skill-1",
		Creator:        "agent-a",
		Description:    "guard the pointer before dereferencing",
		ResolvesErrors: []string{"e1"},
	}
	if err := client.MintSkillTower(skill, []*ErrorNode{{Id: "e1"}}); err != nil {
		t.Fatalf("mint tower: %v", err)
	}

	doc, err := source.SkillDocForError("e1")
	if err != nil {
		t.Fatalf("lookup after mint: %v", err)
	}
	if doc != "guard the pointer before dereferencing" {
		t.Fatalf("doc = %q, want the minted description", doc)
	}

	// And the corpus builder consumes it.
	dataset, ids, err := gatherClusterDatasets(corpusCluster(), source)
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	if len(dataset) != 1 || len(ids) != 1 || ids[0] != "e1" {
		t.Fatalf("expected only the minted member, got %d records %v", len(dataset), ids)
	}
	if dataset[0].CorrectedCompletion != "guard the pointer before dereferencing" {
		t.Fatalf("CorrectedCompletion = %q", dataset[0].CorrectedCompletion)
	}
}
