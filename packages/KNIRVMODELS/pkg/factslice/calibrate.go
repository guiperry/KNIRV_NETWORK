package factslice

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
)

// MinGold is the smallest gold set per family the plan accepts for a measurement
// or a fit (spec D4: at least 150). Below it nothing here claims to be calibrated
// or to meet a target.
const MinGold = 150

// RawScore turns a verdict and the check's stated confidence into x, the raw estimate
// that a claim is acceptable, which calibration maps to the probability a the gate
// reads. A refutation with confidence c is acceptable with raw probability 1-c; NEI
// is a fixed 0.4 (unresolved, leaning neither way).
func RawScore(v Verdict, conf float64) float64 {
	conf = math.Min(1, math.Max(0, conf))
	switch v {
	case Supported:
		return conf
	case Refuted:
		return 1 - conf
	}
	return 0.4
}

// Sample is one labelled claim from the gold set. Label is true when the claim is
// acceptable (true given the evidence), as labelled by the owner, never by a model.
type Sample struct {
	Raw     float64   `json:"raw"`
	Label   bool      `json:"label"`
	Type    ClaimType `json:"type"`
	Verdict Verdict   `json:"verdict"`
	// Gate, when HasGate, is the level the full pipeline gave the claim, for the
	// accepted-precision figure.
	Gate    GateLevel `json:"gate,omitempty"`
	HasGate bool      `json:"has_gate,omitempty"`
}

// Curve is a monotone piecewise-linear map from raw score to probability.
type Curve struct {
	X []float64 `json:"x"`
	Y []float64 `json:"y"`
}

// Apply maps x through the curve; an empty curve is the identity.
func (c Curve) Apply(x float64) float64 {
	n := len(c.X)
	if n == 0 {
		return clamp01(x)
	}
	if x <= c.X[0] {
		return c.Y[0]
	}
	if x >= c.X[n-1] {
		return c.Y[n-1]
	}
	i := sort.SearchFloat64s(c.X, x)
	x0, x1, y0, y1 := c.X[i-1], c.X[i], c.Y[i-1], c.Y[i]
	if x1 == x0 {
		return y1
	}
	return y0 + (y1-y0)*(x-x0)/(x1-x0)
}

func clamp01(x float64) float64 { return math.Min(1, math.Max(0, x)) }

// FitIsotonic fits a non-decreasing curve by pool-adjacent-violators. Ties in Raw are
// ordered by label so the result does not depend on input order.
func FitIsotonic(samples []Sample) (Curve, error) {
	if len(samples) == 0 {
		return Curve{}, errors.New("no samples")
	}
	s := append([]Sample(nil), samples...)
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Raw != s[j].Raw {
			return s[i].Raw < s[j].Raw
		}
		return !s[i].Label && s[j].Label
	})
	type block struct{ sumX, sumY, w float64 }
	var bs []block
	for _, p := range s {
		y := 0.0
		if p.Label {
			y = 1
		}
		bs = append(bs, block{p.Raw, y, 1})
		for len(bs) > 1 {
			a, b := bs[len(bs)-2], bs[len(bs)-1]
			if a.sumY/a.w <= b.sumY/b.w {
				break
			}
			bs = append(bs[:len(bs)-2], block{a.sumX + b.sumX, a.sumY + b.sumY, a.w + b.w})
		}
	}
	var c Curve
	for _, b := range bs {
		x := b.sumX / b.w
		if n := len(c.X); n > 0 && x <= c.X[n-1] {
			x = math.Nextafter(c.X[n-1], 2)
		}
		c.X = append(c.X, x)
		c.Y = append(c.Y, b.sumY/b.w)
	}
	return c, nil
}

// Platt is sigmoid(A*x+B).
type Platt struct{ A, B float64 }

// Apply maps a raw score.
func (p Platt) Apply(x float64) float64 { return 1 / (1 + math.Exp(-(p.A*x + p.B))) }

// FitPlatt fits the sigmoid by Newton steps on the log loss with a small ridge. It
// runs a fixed number of iterations, so the same samples give the same fit.
func FitPlatt(samples []Sample) (Platt, error) {
	if len(samples) == 0 {
		return Platt{}, errors.New("no samples")
	}
	a, b := 1.0, 0.0
	const ridge = 1e-6
	for it := 0; it < 100; it++ {
		var gA, gB, hAA, hAB, hBB float64
		for _, s := range samples {
			p := 1 / (1 + math.Exp(-(a*s.Raw + b)))
			y := 0.0
			if s.Label {
				y = 1
			}
			d := p - y
			w := p * (1 - p)
			gA += d * s.Raw
			gB += d
			hAA += w * s.Raw * s.Raw
			hAB += w * s.Raw
			hBB += w
		}
		hAA += ridge
		hBB += ridge
		det := hAA*hBB - hAB*hAB
		if math.Abs(det) < 1e-12 {
			break
		}
		a -= (hBB*gA - hAB*gB) / det
		b -= (hAA*gB - hAB*gA) / det
	}
	return Platt{a, b}, nil
}

// Calibrator maps a raw score to the probability a, per claim type where the gold
// set is large enough and otherwise by the all-types curve. Until it has been fitted
// on at least MinGold labelled claims it is the identity and Fitted is false, and the
// engine then refuses to take any claim past "seen".
type Calibrator struct {
	Fitted  bool                `json:"fitted"`
	Samples int                 `json:"samples"`
	Default Curve               `json:"default"`
	ByType  map[ClaimType]Curve `json:"by_type,omitempty"`
}

// Apply returns the calibrated probability for a raw score of the given claim type.
func (c *Calibrator) Apply(t ClaimType, raw float64) float64 {
	if c == nil || !c.Fitted {
		return clamp01(raw)
	}
	if cv, ok := c.ByType[t]; ok {
		return cv.Apply(raw)
	}
	return c.Default.Apply(raw)
}

// IsFitted reports whether a gold-fitted curve is installed.
func (c *Calibrator) IsFitted() bool { return c != nil && c.Fitted }

// Fit fits the all-types curve on samples and a curve per type that has at least
// MinGold samples of its own. It returns an unfitted identity calibrator and an error
// when there are fewer than MinGold samples overall.
func Fit(samples []Sample) (*Calibrator, error) {
	if len(samples) < MinGold {
		return &Calibrator{Samples: len(samples)}, errors.New("fewer than 150 labelled claims: not calibrated")
	}
	def, err := FitIsotonic(samples)
	if err != nil {
		return nil, err
	}
	cal := &Calibrator{Fitted: true, Samples: len(samples), Default: def, ByType: map[ClaimType]Curve{}}
	by := map[ClaimType][]Sample{}
	for _, s := range samples {
		by[s.Type] = append(by[s.Type], s)
	}
	for t, ss := range by {
		if len(ss) >= MinGold {
			if cv, err := FitIsotonic(ss); err == nil {
				cal.ByType[t] = cv
			}
		}
	}
	return cal, nil
}

// MarshalJSON / Load keep a fitted calibrator on disk.
func (c *Calibrator) Marshal() ([]byte, error) { return json.MarshalIndent(c, "", "  ") }

// LoadCalibrator reads one written by Marshal.
func LoadCalibrator(data []byte) (*Calibrator, error) {
	var c Calibrator
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Report is what a calibration is measured by.
type Report struct {
	N int `json:"n"`
	// ECE is the expected calibration error over ten equal-width bins; Brier the
	// mean squared error of the calibrated probability.
	ECE   float64 `json:"ece"`
	Brier float64 `json:"brier"`
	// FalseSupportRate is the share of false claims the verdict called SUPPORTED:
	// the key safety metric, which gates promotion of any judge version.
	FalseSupportRate float64 `json:"false_support_rate"`
	FalseClaims      int     `json:"false_claims"`
	// AcceptedPrecision is the share of claims at the accepted gate (Gate >= 5 when the
	// samples carry it, else a >= thresholds.Accepted with a SUPPORTED verdict) that are true.
	AcceptedPrecision float64 `json:"accepted_precision"`
	Accepted          int     `json:"accepted"`
}

// Evaluate measures cal on samples.
func Evaluate(samples []Sample, cal *Calibrator, t Thresholds) Report {
	r := Report{N: len(samples)}
	if len(samples) == 0 {
		return r
	}
	const bins = 10
	var cnt [bins]int
	var sumP, sumY [bins]float64
	falseSupported, acceptedTrue := 0, 0
	for _, s := range samples {
		a := cal.Apply(s.Type, s.Raw)
		y := 0.0
		if s.Label {
			y = 1
		}
		r.Brier += (a - y) * (a - y)
		b := int(a * bins)
		if b >= bins {
			b = bins - 1
		}
		cnt[b]++
		sumP[b] += a
		sumY[b] += y
		if !s.Label {
			r.FalseClaims++
			if s.Verdict == Supported {
				falseSupported++
			}
		}
		accepted := s.Verdict == Supported && a >= t.Accepted
		if s.HasGate {
			accepted = s.Gate >= GateAccepted
		}
		if accepted {
			r.Accepted++
			if s.Label {
				acceptedTrue++
			}
		}
	}
	n := float64(len(samples))
	r.Brier /= n
	for i := 0; i < bins; i++ {
		if cnt[i] > 0 {
			r.ECE += float64(cnt[i]) / n * math.Abs(sumP[i]/float64(cnt[i])-sumY[i]/float64(cnt[i]))
		}
	}
	if r.FalseClaims > 0 {
		r.FalseSupportRate = float64(falseSupported) / float64(r.FalseClaims)
	}
	if r.Accepted > 0 {
		r.AcceptedPrecision = float64(acceptedTrue) / float64(r.Accepted)
	}
	return r
}

// Targets are the Done-when thresholds of Task 3.3 (spec targets; the owner confirms
// them under open D-H2).
type Targets struct {
	MaxFalseSupport      float64 // max(rule baseline, this)
	MinAcceptedPrecision float64
	MaxCalibrationGap    float64 // ECE
	MinSamples           int
}

// DefaultTargets are false-support at most max(baseline, 2 %), accepted precision at
// least 85 %, calibration within 5 points, on at least 150 gold claims.
func DefaultTargets() Targets {
	return Targets{MaxFalseSupport: 0.02, MinAcceptedPrecision: 0.85, MaxCalibrationGap: 0.05, MinSamples: MinGold}
}

// Criterion is one Done-when line. Measurable is false when the gold set is too small
// (or empty): the criterion is then neither met nor failed.
type Criterion struct {
	Name       string  `json:"name"`
	Measurable bool    `json:"measurable"`
	Met        bool    `json:"met"`
	Value      float64 `json:"value"`
	Limit      float64 `json:"limit"`
	Note       string  `json:"note,omitempty"`
}

// Meets reports each target. baselineFalseSupport is the false-support rate of the
// rule-only baseline on the same gold set.
func (r Report) Meets(t Targets, baselineFalseSupport float64) []Criterion {
	enough := r.N >= t.MinSamples
	note := ""
	if !enough {
		note = "not measurable: fewer than the required labelled gold claims (Task 0.5, owner gold data)"
	}
	fsLimit := math.Max(baselineFalseSupport, t.MaxFalseSupport)
	cs := []Criterion{
		{Name: "false_support_rate", Measurable: enough && r.FalseClaims > 0, Value: r.FalseSupportRate, Limit: fsLimit, Note: note},
		{Name: "accepted_precision", Measurable: enough && r.Accepted > 0, Value: r.AcceptedPrecision, Limit: t.MinAcceptedPrecision, Note: note},
		{Name: "calibration_ece", Measurable: enough, Value: r.ECE, Limit: t.MaxCalibrationGap, Note: note},
	}
	cs[0].Met = cs[0].Measurable && r.FalseSupportRate <= fsLimit
	cs[1].Met = cs[1].Measurable && r.AcceptedPrecision >= t.MinAcceptedPrecision
	cs[2].Met = cs[2].Measurable && r.ECE <= t.MaxCalibrationGap
	return cs
}
