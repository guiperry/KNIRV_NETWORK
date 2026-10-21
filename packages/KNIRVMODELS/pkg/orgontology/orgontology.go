// Package orgontology loads an organization's onboarding ontology into the evidence
// index as the ontology tier, and checks claims against it without a model.
//
// The ontology is what the admin entered during onboarding (backend_server's
// OrganizationConfig). It is lists of text plus a few maps, not a graph, so the
// adapter does two things with it: every text entry becomes a citeable passage, and
// every map entry becomes a subject/property/value fact that code can compare exactly.
//
// Trade secrets and user data are never indexed. They exist so guardrails can stop
// them leaking; putting them in an evidence index would let a verdict quote them.
package orgontology

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"knirvmodels/pkg/evidence"
)

// Config is the part of backend_server's OrganizationConfig this adapter reads. It
// is declared here, not imported: services do not import each other, they exchange JSON.
type Config struct {
	ID             string    `json:"id,omitempty"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name,omitempty"`
	Ontology       *Ontology `json:"ontology"`
}

// Ontology mirrors onboarding.Ontology.
type Ontology struct {
	TradeSecrets         []string          `json:"trade_secrets,omitempty"`
	BusinessLogic        []string          `json:"business_logic,omitempty"`
	UserData             []string          `json:"user_data,omitempty"`
	Rules                []string          `json:"rules,omitempty"`
	Regulations          []string          `json:"regulations,omitempty"`
	Procedures           []string          `json:"procedures,omitempty"`
	Policies             []string          `json:"policies,omitempty"`
	FAQs                 []string          `json:"faqs,omitempty"`
	CustomerService      []string          `json:"customer_service,omitempty"`
	IndustryJargon       map[string]string `json:"industry_jargon,omitempty"`
	StakeholderHierarchy map[string]any    `json:"stakeholder_hierarchy,omitempty"`
	DecisionRights       map[string]any    `json:"decision_rights,omitempty"`
}

// Tenant is the organization the ontology belongs to.
func (c Config) Tenant() string {
	if t := strings.TrimSpace(c.OrganizationID); t != "" {
		return t
	}
	return strings.TrimSpace(c.ID)
}

// Excluded names the categories that are never indexed.
var Excluded = []string{"trade_secrets", "user_data"}

// indexed is the ontology with the excluded categories removed: what the version covers.
func (o *Ontology) indexed() Ontology {
	if o == nil {
		return Ontology{}
	}
	c := *o
	c.TradeSecrets, c.UserData = nil, nil
	return c
}

// Version identifies the indexed content: the same ontology always has the same
// version, and changing a secret (which is not indexed) does not change it.
func (c Config) Version() string {
	ind := c.Ontology.indexed()
	b, _ := json.Marshal(ind) // map keys are sorted, so this is canonical
	h := sha256.Sum256(append([]byte(c.Tenant()+"\x00"), b...))
	return "ont1:" + hex.EncodeToString(h[:8])
}

type textList struct {
	key, label string
	items      []string
}

func (o Ontology) lists() []textList {
	return []textList{
		{"rules", "rules", o.Rules}, {"policies", "policies", o.Policies},
		{"regulations", "regulations", o.Regulations}, {"procedures", "procedures", o.Procedures},
		{"business_logic", "business logic", o.BusinessLogic}, {"faqs", "FAQs", o.FAQs},
		{"customer_service", "customer service", o.CustomerService},
	}
}

// Entry counts what Docs produced.
type Counts struct {
	Passages int `json:"passages"`
	Facts    int `json:"facts"`
}

// Docs turns the ontology into evidence documents, each carrying the ontology
// version as its revision so a later version supersedes it.
func (c Config) Docs() ([]evidence.Doc, Counts) {
	var docs []evidence.Doc
	var n Counts
	if c.Ontology == nil {
		return nil, n
	}
	org, ver := c.Tenant(), c.Version()
	o := c.Ontology.indexed()
	name := strings.TrimSpace(c.Name)
	if name == "" {
		name = org
	}
	base := func(id, title string) evidence.Doc {
		return evidence.Doc{ID: id, Title: title, URL: "ontology://" + org + "/" + id, License: "organization-owned",
			Source: "ontology:" + org, Revision: ver}
	}
	for _, l := range o.lists() {
		for i, text := range l.items {
			if strings.TrimSpace(text) == "" {
				continue
			}
			d := base(fmt.Sprintf("%s/%d", l.key, i), name+" "+l.label)
			d.Text = strings.TrimSpace(text)
			docs = append(docs, d)
			n.Passages++
		}
	}
	addFact := func(cat, subject, property, value string) {
		if subject == "" || property == "" || strings.TrimSpace(value) == "" {
			return
		}
		d := base(fmt.Sprintf("%s/%s/%s", cat, evidence.Canon(subject), evidence.Canon(property)+"="+shortHash(value)), name+" "+strings.ReplaceAll(cat, "_", " "))
		d.Facts = []evidence.Fact{{Subject: subject, Property: property, Value: value}}
		docs = append(docs, d)
		n.Facts++
	}
	terms := make([]string, 0, len(o.IndustryJargon))
	for t := range o.IndustryJargon {
		terms = append(terms, t)
	}
	sort.Strings(terms)
	for _, t := range terms {
		addFact("industry_jargon", t, "definition", o.IndustryJargon[t])
	}
	flatten("stakeholder_hierarchy", nil, o.StakeholderHierarchy, addFact)
	flatten("decision_rights", nil, o.DecisionRights, addFact)
	return docs, n
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:4])
}

// flatten walks a map into facts. {"CEO": {"reports_to": "Board"}} is the fact
// (CEO, reports_to, Board); a top-level scalar {"refund_approval": "manager"} is
// (<category>, refund_approval, manager); a list gives one fact per element.
func flatten(cat string, path []string, v any, add func(cat, subject, property, value string)) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			flatten(cat, append(append([]string{}, path...), k), t[k], add)
		}
	case []any:
		for _, e := range t {
			flatten(cat, path, e, add)
		}
	case nil:
	default:
		if len(path) == 0 {
			return
		}
		subject, property := strings.ReplaceAll(cat, "_", " "), path[len(path)-1]
		if len(path) >= 2 {
			subject = strings.Join(path[:len(path)-1], " ")
		}
		add(cat, subject, strings.ReplaceAll(property, "_", " "), scalar(t))
	}
}

func scalar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	}
	return fmt.Sprint(v)
}

// SyncResult reports what a sync did.
type SyncResult struct {
	Tenant    string   `json:"tenant"`
	Version   string   `json:"ontology_version"`
	Previous  string   `json:"previous_version,omitempty"`
	Unchanged bool     `json:"unchanged"`
	Passages  int      `json:"passages"`
	Facts     int      `json:"facts"`
	Excluded  []string `json:"not_indexed"`
	Segment   string   `json:"segment,omitempty"`
}

// Sync loads cfg into the index and makes its version current. It is safe to call on
// every onboarding update: an unchanged ontology writes nothing, and a changed one
// retires the old version only after the new one is fully ingested, so a reader
// never sees a half-loaded ontology.
func Sync(ix *evidence.Index, cfg Config) (SyncResult, error) {
	res := SyncResult{Tenant: cfg.Tenant(), Version: cfg.Version(), Excluded: Excluded}
	if res.Tenant == "" {
		return res, errors.New("orgontology: organization_id is required")
	}
	if cfg.Ontology == nil {
		return res, errors.New("orgontology: the configuration has no ontology")
	}
	res.Previous = ix.OntologyVersion(res.Tenant)
	docs, n := cfg.Docs()
	res.Passages, res.Facts = n.Passages, n.Facts
	if res.Previous == res.Version {
		res.Unchanged = true
		return res, nil
	}
	if len(docs) > 0 {
		br, err := ix.Ingest(evidence.Batch{Tier: evidence.TierOntology, Tenant: res.Tenant, Docs: docs})
		if err != nil {
			return res, err
		}
		res.Segment = br.Segment
	}
	return res, ix.SetOntologyVersion(res.Tenant, res.Version)
}

// Remove retires an organization's ontology (offboarding): none of it is returned again.
func Remove(ix *evidence.Index, tenant string) error { return ix.SetOntologyVersion(tenant, "") }
