package factslice

import (
	"encoding/json"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

// synth makes n labelled claims whose raw score is informative but overconfident.
func synth(n int, seed int64) []Sample {
	rng := rand.New(rand.NewSource(seed))
	types := []ClaimType{TypeNumber, TypeEntity, TypeGeneral}
	var out []Sample
	for i := 0; i < n; i++ {
		raw := rng.Float64()
		label := rng.Float64() < 0.2+0.6*raw // true rate is flatter than the raw score claims
		v := NEI
		if raw > 0.55 {
			v = Supported
		}
		out = append(out, Sample{Raw: raw, Label: label, Type: types[i%3], Verdict: v})
	}
	return out
}

func TestRawScore(t *testing.T) {
	if RawScore(Supported, 0.9) != 0.9 || math.Abs(RawScore(Refuted, 0.9)-0.1) > 1e-12 || RawScore(NEI, 0.99) != 0.4 || RawScore(Supported, 7) != 1 {
		t.Fatal("RawScore")
	}
}

func TestIsotonicIsMonotoneAndReproducible(t *testing.T) {
	s := synth(600, 1)
	c1, err := FitIsotonic(s)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(c1.X); i++ {
		if c1.X[i] <= c1.X[i-1] || c1.Y[i] < c1.Y[i-1] {
			t.Fatalf("not monotone at %d: x %v y %v", i, c1.X[i-1:i+1], c1.Y[i-1:i+1])
		}
	}
	prev := -1.0
	for x := 0.0; x <= 1; x += 0.01 {
		if y := c1.Apply(x); y < prev || y < 0 || y > 1 {
			t.Fatalf("Apply(%v) = %v after %v", x, y, prev)
		}
		prev = c1.Apply(x)
	}
	// The same samples in any order give the identical curve and identical JSON.
	shuf := append([]Sample(nil), s...)
	rand.New(rand.NewSource(99)).Shuffle(len(shuf), func(i, j int) { shuf[i], shuf[j] = shuf[j], shuf[i] })
	c2, _ := FitIsotonic(shuf)
	if !reflect.DeepEqual(c1, c2) {
		t.Fatal("calibration depends on input order")
	}
	a, _ := json.Marshal(c1)
	b, _ := json.Marshal(c2)
	if string(a) != string(b) {
		t.Fatal("calibration is not reproducible")
	}
}

func TestIsotonicFixesAnOverconfidentScore(t *testing.T) {
	s := synth(3000, 2)
	raw := Evaluate(s, nil, DefaultThresholds())
	cal, err := Fit(s)
	if err != nil {
		t.Fatal(err)
	}
	after := Evaluate(s, cal, DefaultThresholds())
	if after.ECE >= raw.ECE || after.ECE > 0.03 || after.Brier >= raw.Brier {
		t.Fatalf("raw %+v, calibrated %+v", raw, after)
	}
}

func TestPlattFit(t *testing.T) {
	s := synth(2000, 3)
	p, err := FitPlatt(s)
	if err != nil {
		t.Fatal(err)
	}
	if p.A <= 0 || p.Apply(0.9) <= p.Apply(0.1) || p.Apply(0) < 0 || p.Apply(1) > 1 {
		t.Fatalf("%+v", p)
	}
	q, _ := FitPlatt(s)
	if p != q {
		t.Fatal("Platt fit is not reproducible")
	}
	if _, err := FitPlatt(nil); err == nil {
		t.Fatal("empty fit accepted")
	}
}

func TestFewerThanMinGoldIsNotCalibrated(t *testing.T) {
	cal, err := Fit(synth(MinGold-1, 4))
	if err == nil || cal.IsFitted() {
		t.Fatalf("a %d-claim gold set was treated as calibration: %+v %v", MinGold-1, cal, err)
	}
	if got := cal.Apply(TypeNumber, 0.9); got != 0.9 {
		t.Fatalf("an unfitted calibrator must be the identity, got %v", got)
	}
	var none *Calibrator
	if none.IsFitted() || none.Apply(TypeGeneral, 0.3) != 0.3 {
		t.Fatal("nil calibrator")
	}
}

func TestPerTypeCurvesNeedTheirOwnGold(t *testing.T) {
	s := synth(3*MinGold+30, 5) // each of 3 types has about 160
	cal, err := Fit(s)
	if err != nil || !cal.IsFitted() || len(cal.ByType) != 3 {
		t.Fatalf("%v %+v", err, cal)
	}
	small := append(synth(300, 6), Sample{Raw: 0.9, Label: false, Type: TypeDate, Verdict: Supported})
	cal, _ = Fit(small)
	if _, ok := cal.ByType[TypeDate]; ok {
		t.Fatal("a one-sample type got its own curve")
	}
	if cal.Apply(TypeDate, 0.5) != cal.Default.Apply(0.5) {
		t.Fatal("a type without its own curve must use the default")
	}
	// Save and load round trip.
	data, _ := cal.Marshal()
	back, err := LoadCalibrator(data)
	again, _ := back.Marshal()
	if err != nil || string(again) != string(data) || back.Apply(TypeNumber, 0.7) != cal.Apply(TypeNumber, 0.7) {
		t.Fatalf("round trip: %v", err)
	}
}

func TestEvaluateHandComputed(t *testing.T) {
	th := DefaultThresholds()
	s := []Sample{
		{Raw: 0.9, Label: true, Verdict: Supported},
		{Raw: 0.9, Label: false, Verdict: Supported}, // a false claim called SUPPORTED
		{Raw: 0.1, Label: false, Verdict: Refuted},
		{Raw: 0.1, Label: false, Verdict: NEI},
	}
	r := Evaluate(s, nil, th)
	// Brier: (0.1^2 + 0.9^2 + 0.1^2 + 0.1^2)/4 = 0.21
	if math.Abs(r.Brier-0.21) > 1e-9 || r.FalseClaims != 3 || math.Abs(r.FalseSupportRate-1.0/3) > 1e-9 {
		t.Fatalf("%+v", r)
	}
	// Both 0.9 claims are in the top bin: |0.9 - 0.5| * 2/4 = 0.2; both 0.1: |0.1-0| * 2/4 = 0.05.
	if math.Abs(r.ECE-0.25) > 1e-9 {
		t.Fatalf("ECE %v", r.ECE)
	}
	if r.Accepted != 2 || r.AcceptedPrecision != 0.5 {
		t.Fatalf("%+v", r)
	}
	s[0].HasGate, s[0].Gate = true, GateAccepted
	s[1].HasGate, s[1].Gate = true, GateSeen
	if r := Evaluate(s, nil, th); r.Accepted != 1 || r.AcceptedPrecision != 1 {
		t.Fatalf("%+v", r)
	}
	if r := Evaluate(nil, nil, th); r.N != 0 {
		t.Fatal(r)
	}
}

// The Done-when thresholds cannot be measured without owner gold data (Task 0.5): with
// fewer than 150 labelled claims every criterion says so instead of passing.
func TestDoneWhenThresholdsAreNotMeasurableWithoutGold(t *testing.T) {
	small := Evaluate(synth(40, 7), nil, DefaultThresholds())
	for _, c := range small.Meets(DefaultTargets(), 0) {
		if c.Measurable || c.Met || c.Note == "" {
			t.Errorf("%+v", c)
		}
	}
	big := synth(2000, 8)
	cal, _ := Fit(big)
	r := Evaluate(big, cal, DefaultThresholds())
	cs := r.Meets(DefaultTargets(), 0)
	for _, c := range cs {
		if !c.Measurable {
			t.Errorf("not measurable with %d claims: %+v", r.N, c)
		}
	}
	if !cs[2].Met { // ECE of a fitted curve on its own training set
		t.Errorf("%+v", cs[2])
	}
	// The false-support limit is the larger of the baseline and 2 %.
	if cs := r.Meets(DefaultTargets(), 0.30); cs[0].Limit != 0.30 {
		t.Errorf("limit %v", cs[0].Limit)
	}
	if cs := r.Meets(DefaultTargets(), 0.001); cs[0].Limit != 0.02 {
		t.Errorf("limit %v", cs[0].Limit)
	}
}
