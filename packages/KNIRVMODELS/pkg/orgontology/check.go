package orgontology

import (
	"strings"

	"knirvmodels/pkg/evidence"
)

// Outcome of checking a structured claim against the organization's ontology.
type Outcome string

const (
	// NoOpinion: the ontology has no fact for this subject and property. The claim
	// goes on to the public sources.
	NoOpinion Outcome = "no_opinion"
	// Supported: the ontology states this value.
	Supported Outcome = "supported"
	// Refuted: the ontology states a different value. The organization's ontology wins
	// over any public source, so this is final for that organization.
	Refuted Outcome = "refuted"
	// Uncertain: the numbers agree but the wording around them does not (a unit, say),
	// so code will not call it either way and the judge decides.
	Uncertain Outcome = "uncertain"
)

// Verdict is a deterministic result with the evidence it rests on.
type Verdict struct {
	Outcome         Outcome        `json:"outcome"`
	Reason          string         `json:"reason"`
	OntologyVersion string         `json:"ontology_version,omitempty"`
	Stated          []string       `json:"ontology_values,omitempty"`
	Hits            []evidence.Hit `json:"hits,omitempty"`
}

// Check compares a claim that names a subject, property and value with what the
// organization's ontology says about that subject and property. No model is involved:
// the same claim and ontology version always give the same verdict.
func Check(ix *evidence.Index, tenant string, c evidence.Claim) (Verdict, error) {
	v := Verdict{Outcome: NoOpinion, OntologyVersion: ix.OntologyVersion(tenant)}
	if strings.TrimSpace(tenant) == "" || v.OntologyVersion == "" {
		v.Reason = "no ontology is loaded for this organization"
		return v, nil
	}
	if strings.TrimSpace(c.Subject) == "" || strings.TrimSpace(c.Property) == "" || strings.TrimSpace(c.Value) == "" {
		v.Reason = "the claim names no subject, property and value to compare"
		return v, nil
	}
	if c.Text == "" {
		c.Text = c.Subject + " " + c.Property + " " + c.Value
	}
	res, err := ix.Search(c, evidence.Options{K: 20, Tenant: tenant, Tiers: []evidence.Tier{evidence.TierOntology}})
	if err != nil {
		return v, err
	}
	wantS, wantP := evidence.Canon(c.Subject), evidence.Canon(c.Property)
	var same []evidence.Hit
	for _, h := range res.Hits {
		if !h.Fact {
			continue
		}
		s, p, val, ok := parseFact(h.Text)
		if ok && evidence.Canon(s) == wantS && evidence.Canon(p) == wantP {
			h.Text = s + " — " + p + ": " + val
			same = append(same, h)
			v.Stated = append(v.Stated, val)
		}
	}
	if len(same) == 0 {
		v.Reason = "the ontology has no fact for this subject and property"
		return v, nil
	}
	v.Hits = same
	uncertain := false
	for _, val := range v.Stated {
		switch compareValues(c.Value, val) {
		case equal:
			v.Outcome, v.Reason = Supported, "the ontology states "+quote(val)
			v.Hits = filterByValue(same, val)
			return v, nil
		case unsure:
			uncertain = true
		}
	}
	if uncertain {
		v.Outcome, v.Reason = Uncertain, "the ontology's value agrees on its numbers but not its wording: "+strings.Join(v.Stated, "; ")
		return v, nil
	}
	v.Outcome = Refuted
	v.Reason = "the ontology states " + strings.Join(quoteAll(v.Stated), " or ") + ", not " + quote(c.Value)
	return v, nil
}

func filterByValue(hits []evidence.Hit, val string) []evidence.Hit {
	var out []evidence.Hit
	for _, h := range hits {
		if _, _, v, ok := parseFact(h.Text); ok && v == val {
			out = append(out, h)
		}
	}
	return out
}

func quote(s string) string { return `"` + s + `"` }

func quoteAll(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = quote(s)
	}
	return out
}

// parseFact reads back "subject — property: value", the text a fact is stored as.
func parseFact(t string) (subject, property, value string, ok bool) {
	s, rest, found := strings.Cut(t, " — ")
	if !found {
		return
	}
	p, v, found := strings.Cut(rest, ": ")
	if !found {
		return
	}
	return s, p, v, true
}

type comparison int

const (
	different comparison = iota
	equal
	unsure
)

// compareValues compares two values the way a person would: dates by calendar day,
// numbers by value (so "$1,000" equals "1000"), everything else by words.
func compareValues(a, b string) comparison {
	da, db := evidence.Dates(a), evidence.Dates(b)
	if len(da) > 0 || len(db) > 0 {
		if sameSet(da, db) {
			return equal
		}
		return different
	}
	na, nb := evidence.Numbers(a), evidence.Numbers(b)
	if len(na) > 0 || len(nb) > 0 {
		if !sameSet(na, nb) {
			return different
		}
		// Same numbers: the rest (units, qualifiers) must not contradict.
		ra, rb := residue(a), residue(b)
		if ra == rb || ra == "" || rb == "" {
			if ra == rb {
				return equal
			}
			return unsure
		}
		return unsure
	}
	if evidence.Canon(a) == evidence.Canon(b) {
		return equal
	}
	return different
}

// residue is a value's words with digits and currency/percent marks removed.
func residue(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' || r == '$' || r == '%' || r == ',' || r == '.' {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return evidence.Canon(b.String())
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
	}
	for _, n := range m {
		if n != 0 {
			return false
		}
	}
	return true
}
