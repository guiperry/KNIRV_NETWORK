package supervision

import "testing"

func TestNoulConfidence(t *testing.T) {
	cases := []struct {
		pYes, want float64
	}{
		{0.0, 1.0},   // certain no → confidence 1
		{1.0, 1.0},   // certain yes → confidence 1
		{0.5, 0.5},   // max ignorance → lowest confidence
		{0.2, 0.8},   // max(p,1-p)
		{0.9, 0.9},   // max(p,1-p)
		{-0.1, 1.0},  // clamped below 0
		{1.4, 1.0},   // clamped above 1
	}
	for _, c := range cases {
		if got := NoulConfidence(c.pYes); got != c.want {
			t.Errorf("NoulConfidence(%v) = %v, want %v", c.pYes, got, c.want)
		}
	}
}

func TestNoulProbabilities(t *testing.T) {
	for _, p := range []float64{0.0, 0.25, 0.5, 0.75, 1.0} {
		pt, pf := NoulProbabilities(p)
		if pt != p {
			t.Errorf("NoulProbabilities(%v).true = %v", p, pt)
		}
		if pf != 1-p {
			t.Errorf("NoulProbabilities(%v).false = %v, want %v", p, pf, 1-p)
		}
		if pt+pf < 0.999 || pt+pf > 1.001 {
			t.Errorf("NoulProbabilities(%v) does not sum to 1", p)
		}
	}
}

func TestPeakProbability(t *testing.T) {
	if got := PeakProbability(nil); got != 0 {
		t.Errorf("PeakProbability(nil) = %v, want 0", got)
	}
	if got := PeakProbability([]float64{}); got != 0 {
		t.Errorf("PeakProbability(empty) = %v, want 0", got)
	}
	if got := PeakProbability([]float64{0.1, 0.7, 0.3}); got != 0.7 {
		t.Errorf("PeakProbability = %v, want 0.7", got)
	}
}

func TestShouldAbstain(t *testing.T) {
	// Below tau → abstain; at or above → decide.
	if !ShouldAbstain(0.4, 0.5) {
		t.Error("peak 0.4 < tau 0.5 should abstain")
	}
	if ShouldAbstain(0.5, 0.5) {
		t.Error("peak 0.5 == tau 0.5 should NOT abstain (threshold is strict <)")
	}
	if ShouldAbstain(0.9, 0.5) {
		t.Error("peak 0.9 >= tau 0.5 should NOT abstain")
	}
}

func TestRoundSeverityForDisplay(t *testing.T) {
	if got := RoundSeverityForDisplay(0.12345, 2); got != 0.12 {
		t.Errorf("RoundSeverityForDisplay(0.12345,2) = %v, want 0.12", got)
	}
	if got := RoundSeverityForDisplay(0.129, 1); got != 0.1 {
		t.Errorf("RoundSeverityForDisplay(0.129,1) = %v, want 0.1", got)
	}
	// Negative decimals are treated as 0 (round to integer).
	if got := RoundSeverityForDisplay(3.7, -1); got != 4.0 {
		t.Errorf("RoundSeverityForDisplay(3.7,-1) = %v, want 4", got)
	}
	// Carrying the float unchanged at 0 decimals of rounding would be lossy; the
	// rule is to round only at display, so the raw value stays separate.
	if got := RoundSeverityForDisplay(0.12345, 0); got != 0.0 {
		t.Errorf("RoundSeverityForDisplay(0.12345,0) = %v, want 0", got)
	}
}
