package connector

import (
	"path/filepath"
	"testing"
)

func TestPriorityRunnerFallsThroughRateLimitedHuggingFaceOnce(t *testing.T) {
	hfCalls := 0
	arxivCalls := 0
	runner := PriorityRunner{
		StatePath: filepath.Join(t.TempDir(), "source_state.json"),
		Override:  TierHuggingFace,
		HuggingFace: func() ([]RawRecord, bool, error) {
			hfCalls++
			return nil, true, nil
		},
		Arxiv: func() ([]RawRecord, bool, error) {
			arxivCalls++
			return []RawRecord{{DatasetID: "arxiv", Text: "fallback record"}}, false, nil
		},
	}

	records, tier, err := runner.Run()
	if err != nil {
		t.Fatal(err)
	}
	if tier != TierArxiv || len(records) != 1 {
		t.Fatalf("tier=%s records=%d", tier, len(records))
	}
	if hfCalls != 1 || arxivCalls != 1 {
		t.Fatalf("hfCalls=%d arxivCalls=%d", hfCalls, arxivCalls)
	}
}
