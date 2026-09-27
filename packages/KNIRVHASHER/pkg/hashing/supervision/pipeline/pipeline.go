// Package pipeline materializes redacted, validated supervision datasets.
package pipeline

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"knirvhasher/pkg/hashing/schema"
)

type SilverConfig struct {
	RedactionVersion             string
	AllowedLicenses              []string
	MinTextLength, MaxTextLength int
}

func DefaultSilverConfig() SilverConfig {
	return SilverConfig{RedactionVersion: "v1.0.0", AllowedLicenses: []string{"operator-generated"}, MinTextLength: 1, MaxTextLength: 16384}
}

// SilverEvent is the sole allow-listed model-ready projection. Raw reducer
// state, tool output, and arbitrary JSON are deliberately not representable.
type SilverEvent struct {
	SchemaVersion    int                  `json:"schema_version"`
	EventID          string               `json:"event_id"`
	SessionID        string               `json:"session_id"`
	InstanceID       string               `json:"instance_id"`
	Sequence         uint64               `json:"sequence"`
	EventKind        string               `json:"event_kind"`
	SourceScope      string               `json:"source_scope"`
	TenantScope      schema.TenantScope   `json:"tenant_scope"`
	Workspace        schema.WorkspaceInfo `json:"workspace"`
	Task             schema.TaskInfo      `json:"task"`
	StateBefore      schema.StateBefore   `json:"state_before"`
	PolicyBundleHash string               `json:"policy_bundle_hash"`
	PolicyContext    []schema.PolicyRef   `json:"policy_context"`
	Evidence         []schema.EvidenceRef `json:"evidence"`
	Trace            schema.TraceInfo     `json:"trace"`
	Decision         schema.DecisionLabel `json:"decision"`
	Outcome          schema.OutcomeLabel  `json:"outcome"`
	License          string               `json:"license"`
	RedactionVersion string               `json:"redaction_version"`
	CollectedAt      time.Time            `json:"collected_at"`
	Quarantined      bool                 `json:"quarantined"`
	QuarantineReason string               `json:"quarantine_reason,omitempty"`
	DuplicateOf      string               `json:"duplicate_of,omitempty"`
}
type QuarantineReport struct {
	Total            int            `json:"total"`
	Quarantined      int            `json:"quarantined"`
	Deduplicated     int            `json:"deduplicated"`
	ByReason         map[string]int `json:"by_reason"`
	DuplicatesByHash map[string]int `json:"duplicates_by_hash"`
}

var secretPatterns = []*regexp.Regexp{regexp.MustCompile(`AKIA[0-9A-Z]{16}`), regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36}`), regexp.MustCompile(`github_pat_[A-Za-z0-9_]{82}`), regexp.MustCompile(`(?i)-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`), regexp.MustCompile(`(?i)(?:password|passwd|token|secret|api[_-]?key)\s*[=:]\s*[^\s]{8,}`), regexp.MustCompile(`(?i)(?:postgres|mysql|mongodb|redis)://[^\s]{10,}`)}

func redactString(s string) string {
	for _, p := range secretPatterns {
		s = p.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}
func containsSecret(s string) bool {
	for _, p := range secretPatterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}
func quarantined(reason string) SilverEvent {
	return SilverEvent{SchemaVersion: schema.SupervisionSchemaVersion, Quarantined: true, QuarantineReason: reason}
}
func stringValue(v any) string { s, _ := v.(string); return s }
func licenseAllowed(v string, allowed []string) bool {
	for _, a := range allowed {
		if strings.EqualFold(v, a) {
			return true
		}
	}
	return false
}
func redactAny(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, c := range x {
			if s, ok := c.(string); ok {
				x[k] = redactString(s)
			} else {
				redactAny(c)
			}
		}
	case []any:
		for i, c := range x {
			if s, ok := c.(string); ok {
				x[i] = redactString(s)
			} else {
				redactAny(c)
			}
		}
	}
}

// NormalizeBronze recursively redacts all textual leaves and rescans the
// serialized projection. Invalid or incompletely-provenanced input is dropped.
func NormalizeBronze(in []byte, cfg SilverConfig) (SilverEvent, bool) {
	var raw map[string]any
	if err := json.Unmarshal(in, &raw); err != nil {
		return quarantined("unparseable JSON"), false
	}
	if !licenseAllowed(stringValue(raw["license"]), cfg.AllowedLicenses) {
		return quarantined("license is not in allow-list"), false
	}
	redactAny(raw)
	clean, err := json.Marshal(raw)
	if err != nil {
		return quarantined("cannot serialize redacted record"), false
	}
	if containsSecret(string(clean)) {
		return quarantined("unredactable secret detected"), false
	}
	var ev SilverEvent
	if err := json.Unmarshal(clean, &ev); err != nil {
		return quarantined("invalid Bronze projection"), false
	}
	ev.SchemaVersion = schema.SupervisionSchemaVersion
	ev.RedactionVersion = cfg.RedactionVersion
	if ev.EventID == "" || ev.PolicyBundleHash == "" || ev.CollectedAt.IsZero() || ev.License == "" {
		return quarantined("incomplete Bronze provenance"), false
	}
	if len(ev.Task.Request) < cfg.MinTextLength || len(ev.Task.Request) > cfg.MaxTextLength {
		return quarantined("task length outside policy"), false
	}
	return ev, true
}

func fingerprint(ev SilverEvent) string {
	b, _ := json.Marshal(struct {
		Task     schema.TaskInfo
		State    schema.StateBefore
		Policy   string
		Trace    schema.TraceInfo
		Decision schema.DecisionLabel
	}{ev.Task, ev.StateBefore, ev.PolicyBundleHash, ev.Trace, ev.Decision})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func DedupSilver(events []SilverEvent) ([]SilverEvent, QuarantineReport) {
	seen := map[string]string{}
	out := make([]SilverEvent, 0, len(events))
	r := QuarantineReport{ByReason: map[string]int{}, DuplicatesByHash: map[string]int{}}
	for _, ev := range events {
		r.Total++
		h := fingerprint(ev)
		if _, ok := seen[h]; ok {
			r.Quarantined++
			r.Deduplicated++
			r.ByReason["duplicate"]++
			r.DuplicatesByHash[h]++
			continue
		}
		seen[h] = ev.EventID
		out = append(out, ev)
	}
	return out, r
}

type ProvenanceValidator struct{ PolicyBundleHash, RedactionVersion string }

func (v *ProvenanceValidator) Validate(ep *schema.SupervisionEpisode) error {
	if err := ep.Validate(); err != nil {
		return err
	}
	if err := ep.VerifyEpisodeID(); err != nil {
		return err
	}
	if v.PolicyBundleHash != "" && ep.Provenance.PolicyBundleHash != v.PolicyBundleHash {
		return errors.New("policy bundle hash mismatch")
	}
	if v.RedactionVersion != "" && ep.Provenance.RedactionVersion != v.RedactionVersion {
		return errors.New("redaction version mismatch")
	}
	return nil
}

type SplitSpec struct {
	Name     string
	From, To time.Time
	Repos    []string
}
type SplitGenerator struct{ Splits []SplitSpec }

func (g *SplitGenerator) Assign(ep schema.SupervisionEpisode) (schema.SplitName, error) {
	for _, s := range g.Splits {
		if (s.From.IsZero() || !ep.Provenance.CreatedAt.Before(s.From)) && (s.To.IsZero() || !ep.Provenance.CreatedAt.After(s.To)) && (len(s.Repos) == 0 || contains(s.Repos, ep.Workspace.RepoFamily)) {
			switch s.Name {
			case "train":
				return schema.SplitTrain, nil
			case "validation":
				return schema.SplitValidation, nil
			case "test":
				return schema.SplitTest, nil
			default:
				return "", fmt.Errorf("invalid split %q", s.Name)
			}
		}
	}
	return "", errors.New("episode does not match an explicit split")
}
func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func (g *SplitGenerator) BuildManifest(episodes []schema.SupervisionEpisode, contrasts []schema.ContrastSet, embedder, version, tokenizer string, dims int) (*schema.DatasetManifest, error) {
	if len(episodes) == 0 {
		return nil, errors.New("empty Gold snapshot")
	}
	assign := map[string]schema.SplitName{}
	ids := make([]string, 0, len(episodes))
	policy, redaction := episodes[0].Provenance.PolicyBundleHash, episodes[0].Provenance.RedactionVersion
	for i := range episodes {
		if err := (&ProvenanceValidator{policy, redaction}).Validate(&episodes[i]); err != nil {
			return nil, fmt.Errorf("episode %d: %w", i, err)
		}
		split, err := g.Assign(episodes[i])
		if err != nil {
			return nil, err
		}
		ids = append(ids, episodes[i].EpisodeID)
		assign[episodes[i].EpisodeID] = split
	}
	cids := make([]string, 0, len(contrasts))
	for i := range contrasts {
		if err := contrasts[i].Validate(); err != nil {
			return nil, err
		}
		a, okA := assign[contrasts[i].AnchorEpisodeID]
		b, okB := assign[contrasts[i].VariantEpisodeID]
		if !okA || !okB || a != b {
			return nil, schema.ErrSplitLeakage
		}
		cids = append(cids, contrasts[i].ContrastSetID)
		assign[contrasts[i].ContrastSetID] = a
	}
	m := &schema.DatasetManifest{SchemaVersion: schema.SupervisionSchemaVersion, CreatedAt: time.Now().UTC(), PolicyBundleHash: policy, RedactionVersion: redaction, EmbeddingModel: embedder, EmbeddingVersion: version, EmbeddingDims: dims, TokenizerVersion: tokenizer, EpisodeCount: len(ids), ContrastSetCount: len(cids), EpisodeIDs: ids, ContrastSetIDs: cids, SplitAssignments: assign, SourceHashes: ids, License: episodes[0].Provenance.License}
	m.ManifestID = m.ComputeManifestID()
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

func Materialize(events []SilverEvent, g *SplitGenerator, embedder, version, tokenizer string, dims int) ([]schema.SupervisionEpisode, *schema.DatasetManifest, error) {
	clean, _ := DedupSilver(events)
	eps := make([]schema.SupervisionEpisode, 0, len(clean))
	for _, ev := range clean {
		ep := schema.SupervisionEpisode{SchemaVersion: schema.SupervisionSchemaVersion, TenantScope: ev.TenantScope, Workspace: ev.Workspace, Task: ev.Task, StateBefore: ev.StateBefore, PolicyContext: ev.PolicyContext, Evidence: ev.Evidence, Trace: ev.Trace, Decision: ev.Decision, Outcome: ev.Outcome, Provenance: schema.ProvenanceInfo{Source: ev.SourceScope, PolicyBundleHash: ev.PolicyBundleHash, RedactionVersion: ev.RedactionVersion, License: ev.License, CreatedAt: ev.CollectedAt}}
		ep.EpisodeID = ep.ComputeEpisodeID()
		if err := ep.Validate(); err != nil {
			return nil, nil, err
		}
		eps = append(eps, ep)
	}
	m, err := g.BuildManifest(eps, nil, embedder, version, tokenizer, dims)
	return eps, m, err
}

func WriteJSONL(path string, eps []schema.SupervisionEpisode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	for _, ep := range eps {
		if err := ep.Validate(); err != nil {
			return err
		}
		if err := ep.VerifyEpisodeID(); err != nil {
			return err
		}
		b, _ := json.Marshal(ep)
		if _, err := w.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}
func ReadJSONL(r io.Reader) ([]schema.SupervisionEpisode, error) {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 64<<10), 64<<20)
	var out []schema.SupervisionEpisode
	line := 0
	for s.Scan() {
		line++
		if strings.TrimSpace(s.Text()) == "" {
			continue
		}
		var ep schema.SupervisionEpisode
		if err := json.Unmarshal(s.Bytes(), &ep); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if err := ep.Validate(); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		if err := ep.VerifyEpisodeID(); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		out = append(out, ep)
	}
	return out, s.Err()
}
func SortedEpisodeIDs(eps []schema.SupervisionEpisode) []string {
	ids := make([]string, len(eps))
	for i := range eps {
		ids[i] = eps[i].EpisodeID
	}
	sort.Strings(ids)
	return ids
}
