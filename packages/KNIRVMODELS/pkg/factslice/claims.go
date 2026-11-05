package factslice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"knirvmodels/pkg/evidence"
	"knirvmodels/pkg/heartqwen"

	"github.com/guiperry/knirv-sdk-go/models/frame"
)

// Limits on what one extraction will read and return.
const (
	MaxTextBytes = 16 << 10
	MaxClaims    = 32
)

// Extractor pulls checkable claims out of a text with the local model under the
// hasher.extract frame, then cross-checks them with deterministic rules.
type Extractor struct {
	LLM   heartqwen.LLM
	Reg   *frame.Registry
	Facts func() frame.LiveFacts
}

// extractSchema is the constraint sent with the request.
var extractSchema = json.RawMessage(`{"type":"object","required":["claims"],"properties":{"claims":{"type":"array","items":{"type":"object","required":["text","checkable"],"properties":{"text":{"type":"string"},"subject":{"type":"string"},"property":{"type":"string"},"value":{"type":"string"},"type":{"enum":["number","date","entity","definition","general"]},"checkable":{"type":"boolean"}}}}}}`)

type wireClaim struct {
	Text      string `json:"text"`
	Subject   string `json:"subject"`
	Property  string `json:"property"`
	Value     string `json:"value"`
	Type      string `json:"type"`
	Checkable *bool  `json:"checkable"`
}

// ErrExtract is returned when the model's output cannot be used; the caller treats
// the extraction as degraded (the stage's on_failure is an empty claim list).
var ErrExtract = errors.New("claim extraction failed")

// ExtractClaims returns the claims of text. The text is data: it is wrapped in
// delimiters the model is told about, a closing delimiter inside it is neutralized,
// and nothing the model returns can do more than propose claims, which the
// deterministic rules below then check.
func (e *Extractor) ExtractClaims(ctx context.Context, text string) ([]Claim, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	if len(text) > MaxTextBytes {
		text = text[:MaxTextBytes]
	}
	var facts frame.LiveFacts
	if e.Facts != nil {
		facts = e.Facts()
	}
	fr, err := e.Reg.Build("hasher.extract", facts)
	if err != nil {
		return nil, fmt.Errorf("%w: build frame: %v", ErrExtract, err)
	}
	out, err := e.LLM.Complete(ctx, heartqwen.Request{
		Frame:     fr,
		Messages:  []heartqwen.Message{{Role: "user", Content: wrapData("TEXT", text) + "\nList the checkable factual claims in the TEXT as JSON."}},
		Schema:    extractSchema,
		MaxTokens: 512,
		Priority:  frame.Priority(1),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrExtract, err)
	}
	var doc struct {
		Claims []wireClaim `json:"claims"`
	}
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(out)))
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: output is not the claims object: %v", ErrExtract, err)
	}
	return crossCheck(text, doc.Claims), nil
}

// wrapData frames untrusted text as quoted data. Every "<<<" and ">>>" inside it is
// replaced, so it cannot contain, imitate or close any marker, whatever its label.
func wrapData(label, s string) string {
	s = strings.ReplaceAll(s, "<<<", "\u2039\u2039\u2039")
	s = strings.ReplaceAll(s, ">>>", "\u203a\u203a\u203a")
	return "<<<BEGIN " + label + ">>>\n" + s + "\n<<<END " + label + ">>>"
}

// ---- deterministic cross-check ------------------------------------------------

var (
	reNumber = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)
	reYear   = regexp.MustCompile(`\b(1[0-9]{3}|20[0-9]{2})\b`)
	reDate   = regexp.MustCompile(`(?i)\b(?:\d{4}-\d{2}-\d{2}|\d{1,2}\s+(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+\d{4}|(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\.?\s+\d{1,2},?\s+\d{4})\b`)
	reCode   = regexp.MustCompile("(?s)```|`[^`]+`|[{};]\\s*$|^\\s*(?:func|def|class|import|var|const|SELECT|INSERT)\\b|=>|->\\s*\\w+\\(|\\w+\\(.*\\)\\s*[;{]")
	reHedge  = regexp.MustCompile(`(?i)\b(?:i think|i believe|i guess|maybe|perhaps|possibly|probably|might|could be|may be|it seems|apparently|supposedly|in my opinion|imo|arguably|i feel)\b`)
	reOpin   = regexp.MustCompile(`(?i)\b(?:best|worst|beautiful|ugly|awesome|terrible|should|ought to|must|better than|prettiest|favorite|favourite)\b`)
	reImper  = regexp.MustCompile(`(?i)^\s*(?:please|kindly|ignore|forget|do not|don't|make sure|remember to|note that|click|open|run|delete|write|tell|give|show|ensure|let's|let us|you must|you should)\b`)
)

// notCheckable classifies a statement the deterministic rules will not let anyone
// score. A model may mark a checkable claim as not checkable; it may never do the
// reverse for these.
func notCheckable(s string) (bool, string) {
	switch {
	case reCode.MatchString(s):
		return true, "code"
	case strings.HasSuffix(strings.TrimSpace(s), "?"):
		return true, "question"
	case reImper.MatchString(s):
		return true, "instruction"
	case reHedge.MatchString(s):
		return true, "hedged"
	case reOpin.MatchString(s):
		return true, "opinion"
	}
	return false, ""
}

// numbersIn returns the canonical numbers, dates and years of s.
func numbersIn(s string) []string { return evidence.ExtractFeatures(s).Numbers }

// classify gives a claim its type by rule.
func classify(s string) ClaimType {
	switch {
	case reDate.MatchString(s):
		return TypeDate
	case len(reNumber.FindAllString(s, -1)) > 0 && !onlyYear(s):
		return TypeNumber
	case onlyYear(s):
		return TypeDate
	case regexp.MustCompile(`(?i)\b(?:is|are) (?:a|an|the)\b`).MatchString(s):
		return TypeDefinition
	case len(evidence.ExtractFeatures(s).Entities) > 0:
		return TypeEntity
	}
	return TypeGeneral
}

func onlyYear(s string) bool {
	nums := reNumber.FindAllString(s, -1)
	return len(nums) > 0 && len(nums) == len(reYear.FindAllString(s, -1))
}

// crossCheck turns the model's proposals into claims.
//
//   - A claim whose numbers or dates do not appear in the source text is dropped:
//     the model invented it (grounding).
//   - Deterministic rules decide that code, questions, instructions, hedges and
//     opinions are not checkable, whatever the model said.
//   - The claim type comes from rules when the model's is missing or unknown.
	//   - A sentence of the source that carries a number, date or a named entity, is
	//     checkable by the rules and is covered by no claim is added as a regex claim, so a
	//     model that skips a figure -- or skips a factual statement about a named thing
	//     entirely -- cannot make it disappear (extraction-recall miss from the 3.4 live
	//     run: "The capital of Atlantis is Poseidonis." returned no claim at all).
func crossCheck(source string, proposed []wireClaim) []Claim {
	srcNums := map[string]bool{}
	for _, n := range numbersIn(source) {
		srcNums[n] = true
	}
	var out []Claim
	seen := map[string]bool{}
	for _, w := range proposed {
		text := strings.TrimSpace(w.Text)
		if text == "" || w.Checkable == nil || len(out) >= MaxClaims {
			continue
		}
		key := strings.ToLower(text)
		if seen[key] {
			continue
		}
		grounded := true
		for _, n := range numbersIn(text) {
			if !srcNums[n] {
				grounded = false
			}
		}
		if !grounded {
			continue
		}
		c := Claim{Text: text, Subject: strings.TrimSpace(w.Subject), Property: strings.TrimSpace(w.Property),
			Value: strings.TrimSpace(w.Value), Type: ClaimType(w.Type), Checkable: *w.Checkable, Origin: "model"}
		if !c.Type.Valid() {
			c.Type = classify(text)
		}
		if bad, why := notCheckable(text); bad {
			c.Checkable, c.Why = false, why
		} else if !c.Checkable {
			c.Why = "marked not checkable"
		}
		seen[key] = true
		out = append(out, c)
	}
	for _, s := range sentences(source) {
		if len(out) >= MaxClaims {
			continue
		}
		hasFig := len(numbersIn(s)) > 0
		hasEnt := len(evidence.ExtractFeatures(s).Entities) > 0
		if !hasFig && !hasEnt {
			continue
		}
		if bad, _ := notCheckable(s); bad || covered(s, out) {
			continue
		}
		out = append(out, Claim{Text: s, Type: classify(s), Checkable: true, Origin: "regex"})
	}
	return out
}

// covered reports whether some existing claim already covers s: every number s
// carries, and at least one named entity s carries, is present in the claim.
func covered(s string, claims []Claim) bool {
	needN := numbersIn(s)
	needE := map[string]bool{}
	for _, e := range evidence.ExtractFeatures(s).Entities {
		needE[e] = true
	}
	if len(needN) == 0 && len(needE) == 0 {
		return false
	}
	for _, c := range claims {
		haveN := map[string]bool{}
		for _, n := range numbersIn(c.Text + " " + c.Value) {
			haveN[n] = true
		}
		haveE := map[string]bool{}
		for _, e := range evidence.ExtractFeatures(c.Text + " " + c.Value).Entities {
			haveE[e] = true
		}
		numsCovered := true
		for _, n := range needN {
			if !haveN[n] {
				numsCovered = false
			}
		}
		entCovered := false
		for e := range needE {
			if haveE[e] {
				entCovered = true
			}
		}
		if (len(needN) == 0 || numsCovered) && (len(needE) == 0 || entCovered) {
			return true
		}
	}
	return false
}

var reSentence = regexp.MustCompile(`[^.!?\n]+[.!?]?`)

func sentences(s string) []string {
	var out []string
	for _, m := range reSentence.FindAllString(s, -1) {
		if m = strings.TrimSpace(m); len(m) >= 8 {
			out = append(out, m)
		}
	}
	return out
}

// Prepare turns a claim a caller supplied into a Claim, applying the same
// deterministic rules as extraction: code, questions, instructions, hedges and
// opinions are not checkable, and the type comes from the text.
func Prepare(text, subject, property, value string) Claim {
	c := Claim{Text: strings.TrimSpace(text), Subject: strings.TrimSpace(subject), Property: strings.TrimSpace(property),
		Value: strings.TrimSpace(value), Checkable: true, Origin: "caller"}
	c.Type = classify(c.Text + " " + c.Value)
	if bad, why := notCheckable(c.Text); bad {
		c.Checkable, c.Why = false, why
	}
	if c.Text == "" {
		c.Checkable, c.Why = false, "empty"
	}
	return c
}
