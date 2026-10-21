package validation

import (
	"path/filepath"
	"testing"
)

func TestJournalPersistsAcrossOpen(t *testing.T) {
	j := &RuleJournal{Path: filepath.Join(t.TempDir(), "sub", "rules.jsonl")}
	v, err := j.Open()
	if err != nil {
		t.Fatal(err)
	}
	r, _ := NewLogicalRule("constraint", []string{}, "must be < 10", "t")
	if err := j.Add(v, "classification", r); err != nil {
		t.Fatal(err)
	}
	if err := j.Delete(v, "anomaly_detection", 0); err != nil {
		t.Fatal(err)
	}

	v2, err := j.Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(v2.KnowledgeBase.Domains["classification"]); got != 2 {
		t.Errorf("classification rules = %d, want 2", got)
	}
	if got := len(v2.KnowledgeBase.Domains["anomaly_detection"]); got != 0 {
		t.Errorf("anomaly_detection rules = %d, want 0 (default deleted)", got)
	}
}

func TestJournalFailedMutationNotJournalled(t *testing.T) {
	j := &RuleJournal{Path: filepath.Join(t.TempDir(), "rules.jsonl")}
	v, _ := j.Open()
	if err := j.Delete(v, "nope", 0); err == nil {
		t.Fatal("expected error")
	}
	v2, err := j.Open()
	if err != nil {
		t.Fatal(err)
	}
	if len(v2.KnowledgeBase.Domains["classification"]) != 1 {
		t.Error("defaults changed")
	}
}
