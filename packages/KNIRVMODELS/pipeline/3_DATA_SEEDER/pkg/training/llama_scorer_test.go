package training

import "testing"

type fixedCandidateScorer struct{}

func (fixedCandidateScorer) Score([]byte, *TrainingRecord) (float64, error) { return 1, nil }

func TestCandidateScorerCannotChangeProofAcceptance(t *testing.T) {
	h := NewEvolutionaryHarness(1)
	h.SetDifficultyMask(0xffffffff)
	h.SetCandidateScorer(fixedCandidateScorer{}, 1)
	if h.IsWinningSeed(1, 2) {
		t.Fatal("advisory scorer must not alter deterministic acceptance")
	}
	if !h.IsWinningSeed(2, 2) {
		t.Fatal("deterministic acceptance changed")
	}
}
