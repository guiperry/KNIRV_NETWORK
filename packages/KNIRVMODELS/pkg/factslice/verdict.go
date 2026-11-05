// Package factslice is the verification engine behind HASHER's Verify (Task 3.3,
// three_layer_inference.md section 5): claim extraction, an order-swapped blinded
// judge, deterministic structured checks, a calibrated confidence and the seven-level
// denial-gate spectrum. No model emits a gate level; Gate is a pure function of the
// checks. A degraded or failed check can never exceed "uncommitted" and is never
// charged.
package factslice

import (
	"knirvmodels/pkg/evidence"
)

// Verdict is the FEVER-style outcome for one claim.
type Verdict string

const (
	Supported Verdict = "SUPPORTED"
	Refuted   Verdict = "REFUTED"
	NEI       Verdict = "NEI" // not enough information
)

// Valid reports whether v is one of the three verdicts.
func (v Verdict) Valid() bool { return v == Supported || v == Refuted || v == NEI }

// GateLevel is the denial-gate spectrum (section 5.7), 0 (refused) to 6 (committed).
type GateLevel int

const (
	GateRefused     GateLevel = iota // 0
	GateDiscouraged                  // 1
	GateUncommitted                  // 2
	GateSeen                         // 3
	GateAttested                     // 4
	GateAccepted                     // 5
	GateCommitted                    // 6
)

var gateNames = [...]string{"refused", "discouraged", "uncommitted", "seen", "attested", "accepted", "committed"}

func (g GateLevel) String() string {
	if g < 0 || int(g) >= len(gateNames) {
		return "invalid"
	}
	return gateNames[g]
}

// ClaimType decides which calibration curve applies and how a claim is checked.
type ClaimType string

const (
	TypeNumber     ClaimType = "number"     // a quantity or count
	TypeDate       ClaimType = "date"       // a date or year
	TypeEntity     ClaimType = "entity"     // a named thing and a property of it
	TypeDefinition ClaimType = "definition" // "X is a Y"
	TypeGeneral    ClaimType = "general"
)

// Valid reports whether t is a known claim type.
func (t ClaimType) Valid() bool {
	switch t {
	case TypeNumber, TypeDate, TypeEntity, TypeDefinition, TypeGeneral:
		return true
	}
	return false
}

// Claim is one factual statement pulled out of a text.
type Claim struct {
	Text     string    `json:"text"`
	Subject  string    `json:"subject,omitempty"`
	Property string    `json:"property,omitempty"`
	Value    string    `json:"value,omitempty"`
	Type     ClaimType `json:"type"`
	// Checkable is false for opinion, instruction, code and hedged statements; they
	// are marked and skipped, never scored. Why says which.
	Checkable bool   `json:"checkable"`
	Why       string `json:"why,omitempty"`
	// Origin is "model" or "regex" (a claim the deterministic cross-check added).
	Origin string `json:"origin,omitempty"`
}

// ev returns the claim in the evidence package's form.
func (c Claim) ev() evidence.Claim {
	return evidence.Claim{Text: c.Text, Subject: c.Subject, Property: c.Property, Value: c.Value}
}

// Source says what produced a claim's verdict.
type Source string

const (
	SourceOntology   Source = "ontology"   // the organization's ontology (deterministic)
	SourceStructured Source = "structured" // exact value comparison on fact spans (deterministic)
	SourceJudge      Source = "judge"      // the order-swapped model judge
	SourceNone       Source = "none"       // no evidence, or the check could not run
)

// ClaimResult is everything the gate reads about one checkable claim.
type ClaimResult struct {
	Claim   Claim   `json:"claim"`
	Verdict Verdict `json:"verdict"`
	Source  Source  `json:"source"`
	// Confidence is the raw confidence the check reported; Calibrated is the
	// calibrated probability a that the claim is acceptable (what the gate reads).
	Confidence float64 `json:"confidence"`
	Calibrated float64 `json:"calibrated"`

	// Evidence is every span the claim was checked against, in rank order. Cited
	// holds the span ids the verdict rests on, a subset of Evidence.
	Evidence []evidence.Hit `json:"evidence"`
	Cited    []string       `json:"cited,omitempty"`

	// Judge facts.
	Agreement         bool    `json:"agreement"` // the order-swapped pair agreed
	DisagreeToRefuted bool    `json:"disagree_toward_refuted,omitempty"`
	Margin            float64 `json:"margin"`

	// Evidence facts the gate reads.
	ReferenceTier bool `json:"reference_tier"` // the verdict rests on reference or ontology evidence
	Corroborated  bool `json:"corroborated"`
	Witnessed     bool `json:"witnessed"` // every cited span has a verified provenance proof

	// Seal facts: only the billing/proof/certificate path sets these (Task 3.9).
	Proof       bool `json:"proof"`
	Certificate bool `json:"certificate"`
	Anchored    bool `json:"anchored"`

	// A degraded check (retrieval or judge down), a failed one, or a deferred one
	// can never exceed "uncommitted" and is never charged.
	Degraded bool   `json:"degraded,omitempty"`
	Failed   bool   `json:"failed,omitempty"`
	Pending  bool   `json:"pending,omitempty"`
	Reason   string `json:"reason,omitempty"`
	// Cached is true when the result came from the claim cache.
	Cached bool `json:"cached,omitempty"`
	// FlashUsed is true when live providers were consulted for this claim; Queries is
	// exactly what was sent to them, and LiveNotes explains a fallback or a stale page.
	FlashUsed bool     `json:"flash_used,omitempty"`
	Queries   []string `json:"outbound_queries,omitempty"`
	LiveNotes []string `json:"live_notes,omitempty"`

	Gate GateLevel `json:"gate"`
}

// Result is the verification of one text.
type Result struct {
	Claims []ClaimResult `json:"claims"`
	// Unchecked are the claims marked not checkable (opinion, instruction, code, hedge).
	Unchecked []Claim `json:"unchecked,omitempty"`
	// Gate is the weakest checkable claim; UncheckedFraction is the share of all
	// extracted claims that were not checkable.
	Gate              GateLevel `json:"gate"`
	UncheckedFraction float64   `json:"unchecked_fraction"`
	// Calibrated is false until a curve fitted on gold data is installed; until then
	// no claim can pass "seen" (provisional, see Engine).
	Calibrated bool `json:"calibrated"`
	// Chargeable is false when any check was degraded, failed or pending: a charge
	// always has a matching proof, and a degraded check has neither.
	Chargeable bool `json:"chargeable"`

	// FlashUsed, OutboundQueries and Notes aggregate the claims' live lookups.
	FlashUsed       bool     `json:"flash_used,omitempty"`
	OutboundQueries []string `json:"outbound_queries,omitempty"`
	Notes           []string `json:"notes,omitempty"`

	// Legacy fields of the {answer, citations, confidence, refused} contract.
	Refused    bool    `json:"refused"`
	IsAccurate bool    `json:"is_accurate"`
	Confidence float64 `json:"confidence"`
	// Citations are positions into the concatenation of every claim's Evidence, in
	// claim order, for the spans each verdict rests on.
	Citations []int `json:"citations"`
}
