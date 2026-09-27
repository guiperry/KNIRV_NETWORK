// Package supervision provides a bounded, policy-scoped retrieval adviser.
package supervision

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"knirvhasher/pkg/embeddings"
	"knirvhasher/pkg/hashing/schema"
)

const (
	FormatVersion              = 2
	DefaultK                   = 5
	DefaultSimilarityThreshold = float32(.15)
	DefaultConflictThreshold   = float32(.4)
	DefaultMinConsensus        = float32(.51)
)

type EmbeddingFunc func(string) []float32
type IndexEntry struct {
	EpisodeID         string    `json:"episode_id"`
	Embedding         []float32 `json:"embedding"`
	RecommendedAction string    `json:"recommended_action"`
	NextPhase         string    `json:"next_phase"`
	Risk              string    `json:"risk"`
	PolicyIDs         []string  `json:"policy_ids"`
	RequiredEvidence  []string  `json:"required_evidence"`
	Intent            string    `json:"intent"`
	OutcomeVerified   bool      `json:"outcome_verified"`
	CreatedAt         time.Time `json:"created_at"`
}
type Index struct {
	Version           int          `json:"version"`
	EmbedderID        string       `json:"embedder_id"`
	Dimensions        int          `json:"dimensions"`
	DatasetManifestID string       `json:"dataset_manifest_id"`
	PolicyBundleHash  string       `json:"policy_bundle_hash"`
	Entries           []IndexEntry `json:"entries"`
}

func NewIndex(embedder string, dims int) *Index {
	return &Index{Version: FormatVersion, EmbedderID: embedder, Dimensions: dims, Entries: []IndexEntry{}}
}
func (idx *Index) Add(ep *schema.SupervisionEpisode, embed EmbeddingFunc) error {
	if err := ep.Validate(); err != nil {
		return err
	}
	if err := ep.VerifyEpisodeID(); err != nil {
		return err
	}
	if embed == nil {
		return fmt.Errorf("embedding function is required")
	}
	v := embed(CanonicalRender(ep))
	if len(v) != idx.Dimensions {
		return fmt.Errorf("embedding dimensions = %d, want %d", len(v), idx.Dimensions)
	}
	req := make([]string, len(ep.Decision.RequiredEvidence))
	for i, e := range ep.Decision.RequiredEvidence {
		req[i] = string(e)
	}
	pol := make([]string, len(ep.PolicyContext))
	for i, p := range ep.PolicyContext {
		pol[i] = p.ID
	}
	idx.Entries = append(idx.Entries, IndexEntry{EpisodeID: ep.EpisodeID, Embedding: v, RecommendedAction: ep.Decision.RecommendedAction, NextPhase: string(ep.Decision.NextPhase), Risk: string(ep.Task.Risk), PolicyIDs: pol, RequiredEvidence: req, Intent: string(ep.Task.Intent), OutcomeVerified: ep.Outcome.Verified && !ep.Outcome.HumanOverride, CreatedAt: ep.Provenance.CreatedAt})
	return nil
}
func (idx *Index) Save(path string) error {
	if err := idx.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0640)
}
func LoadIndex(path string) (*Index, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	if err := idx.Validate(); err != nil {
		return nil, err
	}
	return &idx, nil
}
func (idx *Index) Validate() error {
	if idx.Version != FormatVersion || idx.Dimensions <= 0 || idx.EmbedderID == "" || idx.DatasetManifestID == "" || idx.PolicyBundleHash == "" {
		return fmt.Errorf("invalid supervision index metadata")
	}
	seen := map[string]bool{}
	for _, e := range idx.Entries {
		if e.EpisodeID == "" || seen[e.EpisodeID] || len(e.Embedding) != idx.Dimensions {
			return fmt.Errorf("invalid index entry")
		}
		seen[e.EpisodeID] = true
		for _, n := range e.Embedding {
			if math.IsNaN(float64(n)) || math.IsInf(float64(n), 0) {
				return fmt.Errorf("non-finite embedding")
			}
		}
	}
	return nil
}

type RetrievalConfig struct {
	K                                                    int
	SimilarityThreshold, ConflictThreshold, MinConsensus float32
}

func DefaultRetrievalConfig() *RetrievalConfig {
	return &RetrievalConfig{K: DefaultK, SimilarityThreshold: DefaultSimilarityThreshold, ConflictThreshold: DefaultConflictThreshold, MinConsensus: DefaultMinConsensus}
}

type RankResult struct {
	Entry      IndexEntry `json:"entry"`
	Similarity float32    `json:"similarity"`
}
type RankedResults []RankResult

func (r RankedResults) Len() int { return len(r) }
func (r RankedResults) Less(i, j int) bool {
	if r[i].Similarity == r[j].Similarity {
		return r[i].Entry.EpisodeID < r[j].Entry.EpisodeID
	}
	return r[i].Similarity > r[j].Similarity
}
func (r RankedResults) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r RankedResults) TopK(k int) RankedResults {
	if k <= 0 {
		return nil
	}
	if len(r) <= k {
		return r
	}
	return r[:k]
}
func (idx *Index) Query(v []float32) (RankedResults, error) {
	if len(v) != idx.Dimensions {
		return nil, fmt.Errorf("query embedding dimensions = %d, want %d", len(v), idx.Dimensions)
	}
	out := make(RankedResults, 0, len(idx.Entries))
	for _, e := range idx.Entries {
		out = append(out, RankResult{e, embeddings.CosineSimilarity(v, e.Embedding)})
	}
	sort.Sort(out)
	return out, nil
}

type SuperviseContext struct {
	CurrentPhase     string               `json:"current_phase"`
	RemainingBudget  map[string]int       `json:"remaining_budget"`
	PolicyBundleHash string               `json:"policy_bundle_hash"`
	PolicyRefs       []schema.PolicyRef   `json:"policy_refs"`
	EvidenceSummary  []schema.EvidenceRef `json:"evidence_summary"`
	TaskSummary      schema.TaskInfo      `json:"task_summary"`
}
type SupervisorAdvice struct {
	RecommendedAction   string   `json:"recommended_action"`
	NextPhase           string   `json:"next_phase"`
	MissingEvidence     []string `json:"missing_evidence,omitempty"`
	PolicyReferences    []string `json:"policy_references,omitempty"`
	Confidence          float32  `json:"confidence"`
	Abstain             bool     `json:"abstain"`
	AbstainReason       string   `json:"abstain_reason,omitempty"`
	RetrievalRefs       []string `json:"retrieval_refs,omitempty"`
	ModelVersion        string   `json:"model_version"`
	DatasetManifestHash string   `json:"dataset_manifest_hash"`
	PolicyBundleHash    string   `json:"policy_bundle_hash"`
}
type Retriever interface {
	Advise(SuperviseContext) (SupervisorAdvice, error)
}
type SemanticRetriever struct {
	idx      *Index
	config   *RetrievalConfig
	embed    EmbeddingFunc
	modelVer string
}

func NewRetriever(idx *Index, embed EmbeddingFunc, modelVer string) *SemanticRetriever {
	if embed == nil {
		embed = func(s string) []float32 { return embeddings.NewDeterministicService().GetEmbedding(s) }
	}
	return &SemanticRetriever{idx: idx, config: DefaultRetrievalConfig(), embed: embed, modelVer: modelVer}
}
func RenderContext(c SuperviseContext) string {
	var b strings.Builder
	fmt.Fprintf(&b, "TASK intent=%s risk=%s request=%s STATE phase=%s budget=", c.TaskSummary.Intent, c.TaskSummary.Risk, c.TaskSummary.Request, c.CurrentPhase)
	keys := make([]string, 0, len(c.RemainingBudget))
	for k := range c.RemainingBudget {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s:%d ", k, c.RemainingBudget[k])
	}
	b.WriteString("POLICIES")
	for _, p := range c.PolicyRefs {
		b.WriteString(" ")
		b.WriteString(p.ID)
	}
	b.WriteString(" EVIDENCE")
	for _, e := range c.EvidenceSummary {
		b.WriteString(" ")
		b.WriteString(string(e.Class))
	}
	return b.String()
}
func CanonicalRender(ep *schema.SupervisionEpisode) string {
	return RenderContext(SuperviseContext{CurrentPhase: string(ep.StateBefore.Phase), RemainingBudget: ep.StateBefore.Budget, PolicyBundleHash: ep.Provenance.PolicyBundleHash, PolicyRefs: ep.PolicyContext, EvidenceSummary: ep.Evidence, TaskSummary: ep.Task})
}
func (r *SemanticRetriever) Advise(c SuperviseContext) (SupervisorAdvice, error) {
	if r.idx == nil {
		return r.abstain("index unavailable", c), nil
	}
	if c.CurrentPhase == "" || c.PolicyBundleHash == "" || len(c.PolicyRefs) == 0 {
		return r.abstain("incomplete state or policy context", c), nil
	}
	if c.PolicyBundleHash != r.idx.PolicyBundleHash {
		return r.abstain("index policy bundle is incompatible", c), nil
	}
	all, err := r.idx.Query(r.embed(RenderContext(c)))
	if err != nil {
		return r.abstain("retrieval error", c), err
	}
	verified := make(RankedResults, 0, len(all))
	for _, x := range all {
		if x.Entry.OutcomeVerified {
			verified = append(verified, x)
		}
	}
	top := verified.TopK(r.config.K)
	if len(top) == 0 || top[0].Similarity < r.config.SimilarityThreshold {
		return r.abstain("no sufficiently similar verified precedent", c), nil
	}
	votes := map[string]int{}
	for _, x := range top {
		votes[x.Entry.RecommendedAction]++
	}
	action, count := majority(votes)
	conflict := float32(len(top)-count) / float32(len(top))
	if conflict > r.config.ConflictThreshold {
		return r.abstain("conflicting precedents", c), nil
	}
	consensus := float32(count) / float32(len(top))
	if consensus < r.config.MinConsensus {
		return r.abstain("insufficient consensus", c), nil
	}
	return SupervisorAdvice{RecommendedAction: action, NextPhase: phase(top, action), MissingEvidence: missing(c, top), Confidence: clamp(top[0].Similarity*.6 + consensus*.4), RetrievalRefs: refs(top), ModelVersion: r.modelVer, DatasetManifestHash: r.idx.DatasetManifestID, PolicyBundleHash: r.idx.PolicyBundleHash}, nil
}
func (r *SemanticRetriever) abstain(reason string, c SuperviseContext) SupervisorAdvice {
	id := ""
	if r.idx != nil {
		id = r.idx.DatasetManifestID
	}
	return SupervisorAdvice{Abstain: true, AbstainReason: reason, ModelVersion: r.modelVer, DatasetManifestHash: id, PolicyBundleHash: c.PolicyBundleHash}
}
func majority(v map[string]int) (string, int) {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	best, n := "", 0
	for _, k := range keys {
		if v[k] > n {
			best, n = k, v[k]
		}
	}
	return best, n
}
func phase(rs RankedResults, a string) string {
	for _, r := range rs {
		if r.Entry.RecommendedAction == a {
			return r.Entry.NextPhase
		}
	}
	return ""
}
func refs(rs RankedResults) []string {
	o := make([]string, len(rs))
	for i, r := range rs {
		o[i] = r.Entry.EpisodeID
	}
	return o
}
func missing(c SuperviseContext, rs RankedResults) []string {
	present := map[string]bool{}
	for _, e := range c.EvidenceSummary {
		present[string(e.Class)] = true
	}
	m := map[string]bool{}
	for _, r := range rs {
		for _, e := range r.Entry.RequiredEvidence {
			if !present[e] {
				m[e] = true
			}
		}
	}
	o := make([]string, 0, len(m))
	for k := range m {
		o = append(o, k)
	}
	sort.Strings(o)
	return o
}
func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func BuildIndexForManifest(eps []schema.SupervisionEpisode, m *schema.DatasetManifest, embed EmbeddingFunc) (*Index, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest is required")
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if err := m.VerifyManifestID(); err != nil {
		return nil, err
	}
	idx := NewIndex(m.EmbeddingModel, m.EmbeddingDims)
	idx.DatasetManifestID = m.ManifestID
	idx.PolicyBundleHash = m.PolicyBundleHash
	for i := range eps {
		if err := idx.Add(&eps[i], embed); err != nil {
			return nil, err
		}
	}
	return idx, nil
}
