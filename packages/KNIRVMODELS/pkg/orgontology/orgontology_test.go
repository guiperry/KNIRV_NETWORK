package orgontology

import (
	"encoding/json"
	"strings"
	"testing"

	"knirvmodels/pkg/evidence"
)

const acme = `{
 "organization_id": "acme", "name": "Acme",
 "ontology": {
  "trade_secrets": ["The Phoenix formula uses 14% zirconium"],
  "user_data": ["customer SSN"],
  "policies": ["Refunds over $500 require manager approval before they are issued."],
  "industry_jargon": {"MTTR": "mean time to resolve an error node"},
  "decision_rights": {"refund_approval": "manager", "data_retention_years": 7},
  "stakeholder_hierarchy": {"Security Lead": {"reports_to": "CISO"}}
 }}`

func load(t *testing.T, doc string) Config {
	t.Helper()
	var c Config
	if err := json.Unmarshal([]byte(doc), &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func open(t *testing.T) *evidence.Index {
	t.Helper()
	ix, err := evidence.Open(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ix.Close)
	return ix
}

func TestSyncIndexesFactsAndPassagesButNeverSecrets(t *testing.T) {
	ix := open(t)
	r, err := Sync(ix, load(t, acme))
	if err != nil {
		t.Fatal(err)
	}
	if r.Facts != 4 || r.Passages != 1 || r.Unchanged || ix.OntologyVersion("acme") != r.Version {
		t.Fatalf("%+v", r)
	}
	for _, q := range []string{"Phoenix formula zirconium", "customer SSN"} {
		res, _ := ix.Search(evidence.Claim{Text: q}, evidence.Options{K: 10, Tenant: "acme"})
		for _, h := range res.Hits {
			if strings.Contains(h.Text, "zirconium") || strings.Contains(h.Text, "SSN") {
				t.Fatalf("a secret was indexed: %+v", h)
			}
		}
	}
	// Changing only a secret does not change the version, and a second sync is a no-op.
	c := load(t, acme)
	c.Ontology.TradeSecrets = []string{"something else"}
	if c.Version() != r.Version {
		t.Fatal("secrets must not affect the version")
	}
	if r2, _ := Sync(ix, c); !r2.Unchanged {
		t.Fatalf("%+v", r2)
	}
}

func TestCheckIsDeterministicAndOntologyWins(t *testing.T) {
	ix := open(t)
	// A public source says otherwise about the same subject and property.
	_, err := ix.Ingest(evidence.Batch{Tier: evidence.TierReference, Docs: []evidence.Doc{{ID: "w", Title: "Refunds",
		Facts: []evidence.Fact{{Subject: "refund approval", Property: "approver", Value: "director"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Sync(ix, load(t, acme)); err != nil {
		t.Fatal(err)
	}
	cl := func(s, p, v string) evidence.Claim { return evidence.Claim{Subject: s, Property: p, Value: v} }
	cases := []struct {
		c    evidence.Claim
		want Outcome
	}{
		{cl("Decision Rights", "Refund Approval", "Manager"), Supported},
		{cl("decision rights", "refund approval", "director"), Refuted},
		{cl("decision rights", "data retention years", "7 years"), Uncertain}, // same number, extra wording
		{cl("decision rights", "data retention years", "7"), Supported},
		{cl("decision rights", "data retention years", "10"), Refuted},
		{cl("Security Lead", "reports to", "CISO"), Supported},
		{cl("Security Lead", "reports to", "CEO"), Refuted},
		{cl("MTTR", "definition", "Mean Time To Resolve an error node."), Supported},
		{cl("the moon", "orbits", "earth"), NoOpinion},
	}
	for _, tc := range cases {
		v, err := Check(ix, "acme", tc.c)
		if err != nil || v.Outcome != tc.want {
			t.Errorf("%+v: got %s (%s) err=%v, want %s", tc.c, v.Outcome, v.Reason, err, tc.want)
		}
		if tc.want != NoOpinion && (v.OntologyVersion == "" || len(v.Hits) == 0) {
			t.Errorf("%+v: a verdict must carry the ontology version and its evidence: %+v", tc.c, v)
		}
	}
	// Ontology hits rank above the public source in a plain search.
	res, _ := ix.Search(evidence.Claim{Text: "refund approval", Subject: "decision rights", Property: "refund approval"}, evidence.Options{K: 5, Tenant: "acme"})
	if len(res.Hits) == 0 || res.Hits[0].Tier != evidence.TierOntology {
		t.Fatalf("ontology must rank first: %+v", res.Hits)
	}
}

func TestOtherOrganizationsNeverSeeIt(t *testing.T) {
	ix := open(t)
	Sync(ix, load(t, acme))
	for _, tenant := range []string{"globex", ""} {
		v, _ := Check(ix, tenant, evidence.Claim{Subject: "decision rights", Property: "refund approval", Value: "manager"})
		if v.Outcome != NoOpinion {
			t.Fatalf("tenant %q saw acme's ontology: %+v", tenant, v)
		}
		res, _ := ix.Search(evidence.Claim{Text: "refund approval manager"}, evidence.Options{K: 5, Tenant: tenant})
		for _, h := range res.Hits {
			if h.Tier == evidence.TierOntology {
				t.Fatalf("tenant %q got an ontology hit", tenant)
			}
		}
	}
}

func TestANewVersionRetiresTheOldOneEvenForUnchangedFacts(t *testing.T) {
	ix := open(t)
	v1, _ := Sync(ix, load(t, acme))
	c := load(t, acme)
	c.Ontology.DecisionRights["refund_approval"] = "director"
	v2, err := Sync(ix, c)
	if err != nil || v2.Version == v1.Version || v2.Previous != v1.Version {
		t.Fatalf("%+v %v", v2, err)
	}
	got := func(s, p, v string) Outcome {
		x, _ := Check(ix, "acme", evidence.Claim{Subject: s, Property: p, Value: v})
		return x.Outcome
	}
	if got("decision rights", "refund approval", "manager") != Refuted || got("decision rights", "refund approval", "director") != Supported {
		t.Fatal("the old value must be gone and the new one current")
	}
	// A fact that did not change is still there (same text, new version).
	if got("Security Lead", "reports to", "CISO") != Supported {
		t.Fatal("an unchanged fact must survive a version change")
	}
	// Going back to the first ontology works too.
	if _, err := Sync(ix, load(t, acme)); err != nil || got("decision rights", "refund approval", "manager") != Supported {
		t.Fatalf("rollback: %v", err)
	}
	// Offboarding hides everything, and the choice survives a reopen.
	Remove(ix, "acme")
	if got("decision rights", "refund approval", "manager") != NoOpinion {
		t.Fatal("removed ontology still answers")
	}
}

func TestVersionSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	ix, _ := evidence.Open(dir, "")
	r, _ := Sync(ix, load(t, acme))
	ix.Close()
	ix2, err := evidence.Open(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer ix2.Close()
	if ix2.OntologyVersion("acme") != r.Version {
		t.Fatal("current version lost on reopen")
	}
	if v, _ := Check(ix2, "acme", evidence.Claim{Subject: "decision rights", Property: "refund approval", Value: "manager"}); v.Outcome != Supported {
		t.Fatalf("%+v", v)
	}
}

func TestSyncNeedsAnOrganizationAndAnOntology(t *testing.T) {
	ix := open(t)
	if _, err := Sync(ix, Config{Ontology: &Ontology{}}); err == nil {
		t.Fatal("no organization")
	}
	if _, err := Sync(ix, Config{OrganizationID: "x"}); err == nil {
		t.Fatal("no ontology")
	}
}
