package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	SupervisionSchemaVersion = 1
	SupervisionEpisodeKind   = "supervision_episode"
	ContrastSetKind          = "contrast_set"
	DatasetManifestKind      = "dataset_manifest"
)

var (
	ErrInvalidSchemaVersion    = errors.New("invalid supervision schema version")
	ErrMissingRequiredField    = errors.New("missing required field")
	ErrHashMismatch            = errors.New("content hash mismatch")
	ErrUnknownVersion          = errors.New("unknown schema version")
	ErrProvenanceIncomplete    = errors.New("provenance incomplete")
	ErrRedactionFailed         = errors.New("redaction failed")
	ErrSplitLeakage            = errors.New("split leakage detected")
	ErrInconsistentPolicy      = errors.New("inconsistent policy version")
	ErrInconsistentRedaction   = errors.New("inconsistent redaction version")
)

type TenantScope string

const (
	TenantScopeLocal              TenantScope = "local"
	TenantScopeOperatorApproved   TenantScope = "operator-approved-tenant"
)

type Intent string

const (
	IntentPlan          Intent = "plan"
	IntentInvestigate   Intent = "investigate"
	IntentDevelop       Intent = "develop"
	IntentReview        Intent = "review"
	IntentValidate      Intent = "validate"
	IntentRecover       Intent = "recover"
	IntentAskOperator   Intent = "ask_operator"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type SafetyDecision string

const (
	SafetyAllow              SafetyDecision = "allow"
	SafetyRequireConfirmation SafetyDecision = "require_confirmation"
	SafetyDeny               SafetyDecision = "deny"
	SafetyAbstain            SafetyDecision = "abstain"
)

type PhaseName string

const (
	PhasePlan         PhaseName = "plan"
	PhasePlanReview   PhaseName = "plan_review"
	PhaseDevelop      PhaseName = "develop"
	PhaseDevelopReview PhaseName = "develop_review"
	PhaseCommit       PhaseName = "commit"
	PhaseCommitCleanup PhaseName = "commit_cleanup"
	PhaseTerminal     PhaseName = "terminal"
)

type EvidenceClass string

const (
	EvidenceClassPlan   EvidenceClass = "plan"
	EvidenceClassDiff   EvidenceClass = "diff"
	EvidenceClassReport EvidenceClass = "report"
	EvidenceClassTest   EvidenceClass = "test"
	EvidenceClassLog    EvidenceClass = "log"
)

type ActionClass string

const (
	ActionClassRead     ActionClass = "read"
	ActionClassTest     ActionClass = "test"
	ActionClassWrite    ActionClass = "write"
	ActionClassDelete   ActionClass = "delete"
	ActionClassOverwrite ActionClass = "overwrite"
	ActionClassDeploy   ActionClass = "deploy"
)

type OutcomeStatus string

const (
	OutcomeVerifiedSuccess OutcomeStatus = "verified_success"
	OutcomeFailedValidation OutcomeStatus = "failed_validation"
	OutcomeAborted         OutcomeStatus = "aborted"
	OutcomeHumanOverride   OutcomeStatus = "human_override"
	OutcomeLaterRegression OutcomeStatus = "later_regression"
)

type ContrastRelation string

const (
	ContrastMustChange    ContrastRelation = "must_change"
	ContrastMustNotChange ContrastRelation = "must_not_change"
	ContrastMustAbstain   ContrastRelation = "must_abstain"
)

type LabelSource string

const (
	LabelSourceDeterministicGate LabelSource = "deterministic_gate"
	LabelSourceHumanReview       LabelSource = "human_review"
)

type ReviewStatus string

const (
	ReviewStatusPending   ReviewStatus = "pending"
	ReviewStatusApproved  ReviewStatus = "approved"
	ReviewStatusRejected  ReviewStatus = "rejected"
)

type SplitName string

const (
	SplitTrain     SplitName = "train"
	SplitValidation SplitName = "validation"
	SplitTest      SplitName = "test"
)

type RedactionStatus string

const (
	RedactionStatusClean     RedactionStatus = "clean"
	RedactionStatusQuarantine RedactionStatus = "quarantine"
)

type WorkspaceInfo struct {
	Fingerprint string `json:"fingerprint"`
	RepoFamily  string `json:"repo_family"`
	Branch      string `json:"branch"`
}

type TaskInfo struct {
	Request string `json:"request"`
	Intent  Intent `json:"intent"`
	Risk    RiskLevel `json:"risk"`
}

type StateBefore struct {
	Phase        PhaseName          `json:"phase"`
	Budget       map[string]int     `json:"budget"`
	AgentChain   []string           `json:"agent_chain"`
	LoopIteration int               `json:"loop_iteration"`
}

type PolicyRef struct {
	ID             string `json:"id"`
	ContentHash    string `json:"content_hash"`
	Applicability  string `json:"applicability"`
}

type EvidenceRef struct {
	Class  EvidenceClass `json:"class"`
	Hash   string        `json:"hash"`
	Summary string       `json:"summary"`
}

type TraceInfo struct {
	Agent       string      `json:"agent"`
	Proposal    string      `json:"proposal"`
	ToolClasses []ActionClass `json:"tool_classes"`
}

type DecisionLabel struct {
	RecommendedAction string         `json:"recommended_action"`
	NextPhase         PhaseName      `json:"next_phase"`
	RequiredEvidence  []EvidenceClass `json:"required_evidence"`
	Approval          string         `json:"approval"`
	Abstain           bool           `json:"abstain"`
	Confidence        float32        `json:"confidence"`
}

type OutcomeLabel struct {
	Verified        bool `json:"verified"`
	TestsPassed     bool `json:"tests_passed"`
	HumanOverride   bool `json:"human_override"`
}

type ProvenanceInfo struct {
	Source            string    `json:"source"`
	PolicyBundleHash  string    `json:"policy_bundle_hash"`
	RedactionVersion  string    `json:"redaction_version"`
	License           string    `json:"license"`
	CreatedAt         time.Time `json:"created_at"`
}

type SupervisionEpisode struct {
	SchemaVersion int             `json:"schema_version"`
	EpisodeID     string          `json:"episode_id"`
	TenantScope   TenantScope     `json:"tenant_scope"`
	Workspace     WorkspaceInfo   `json:"workspace"`
	Task          TaskInfo        `json:"task"`
	StateBefore   StateBefore     `json:"state_before"`
	PolicyContext []PolicyRef     `json:"policy_context"`
	Evidence      []EvidenceRef   `json:"evidence"`
	Trace         TraceInfo       `json:"trace"`
	Decision      DecisionLabel   `json:"decision"`
	Outcome       OutcomeLabel    `json:"outcome"`
	Provenance    ProvenanceInfo  `json:"provenance"`
}

func (e *SupervisionEpisode) Validate() error {
	if e.SchemaVersion != SupervisionSchemaVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrInvalidSchemaVersion, e.SchemaVersion, SupervisionSchemaVersion)
	}
	if strings.TrimSpace(e.EpisodeID) == "" {
		return fmt.Errorf("%w: episode_id", ErrMissingRequiredField)
	}
	if strings.TrimSpace(string(e.TenantScope)) == "" {
		return fmt.Errorf("%w: tenant_scope", ErrMissingRequiredField)
	}
	if e.TenantScope != TenantScopeLocal && e.TenantScope != TenantScopeOperatorApproved {
		return fmt.Errorf("invalid tenant_scope: %s", e.TenantScope)
	}
	if strings.TrimSpace(e.Workspace.Fingerprint) == "" {
		return fmt.Errorf("%w: workspace.fingerprint", ErrMissingRequiredField)
	}
	if strings.TrimSpace(e.Task.Request) == "" {
		return fmt.Errorf("%w: task.request", ErrMissingRequiredField)
	}
	if e.Task.Intent == "" {
		return fmt.Errorf("%w: task.intent", ErrMissingRequiredField)
	}
	if e.Task.Risk == "" {
		return fmt.Errorf("%w: task.risk", ErrMissingRequiredField)
	}
	if e.StateBefore.Phase == "" {
		return fmt.Errorf("%w: state_before.phase", ErrMissingRequiredField)
	}
	if e.Decision.RecommendedAction == "" {
		return fmt.Errorf("%w: decision.recommended_action", ErrMissingRequiredField)
	}
	if e.Decision.NextPhase == "" {
		return fmt.Errorf("%w: decision.next_phase", ErrMissingRequiredField)
	}
	if e.Provenance.Source == "" {
		return fmt.Errorf("%w: provenance.source", ErrMissingRequiredField)
	}
	if e.Provenance.PolicyBundleHash == "" {
		return fmt.Errorf("%w: provenance.policy_bundle_hash", ErrMissingRequiredField)
	}
	if e.Provenance.RedactionVersion == "" {
		return fmt.Errorf("%w: provenance.redaction_version", ErrMissingRequiredField)
	}
	if e.Provenance.License == "" {
		return fmt.Errorf("%w: provenance.license", ErrMissingRequiredField)
	}
	if e.Provenance.CreatedAt.IsZero() {
		return fmt.Errorf("%w: provenance.created_at", ErrMissingRequiredField)
	}
	for i, p := range e.PolicyContext {
		if p.ID == "" || p.ContentHash == "" || p.Applicability == "" {
			return fmt.Errorf("policy_context[%d]: missing required fields", i)
		}
	}
	for i, ev := range e.Evidence {
		if ev.Class == "" || ev.Hash == "" {
			return fmt.Errorf("evidence[%d]: missing required fields", i)
		}
	}
	if e.Trace.Agent == "" {
		return fmt.Errorf("%w: trace.agent", ErrMissingRequiredField)
	}
	return nil
}

func (e *SupervisionEpisode) ComputeEpisodeID() string {
	canonical := e.canonicalPayload()
	hash := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func (e *SupervisionEpisode) canonicalPayload() string {
	type canonicalEpisode struct {
		SchemaVersion int             `json:"schema_version"`
		TenantScope   TenantScope     `json:"tenant_scope"`
		Workspace     WorkspaceInfo   `json:"workspace"`
		Task          TaskInfo        `json:"task"`
		StateBefore   StateBefore     `json:"state_before"`
		PolicyContext []PolicyRef     `json:"policy_context"`
		Evidence      []EvidenceRef   `json:"evidence"`
		Trace         TraceInfo       `json:"trace"`
		Decision      DecisionLabel   `json:"decision"`
		Outcome       OutcomeLabel    `json:"outcome"`
		Provenance    ProvenanceInfo  `json:"provenance"`
	}
	ce := canonicalEpisode{
		SchemaVersion: e.SchemaVersion,
		TenantScope:   e.TenantScope,
		Workspace:     e.Workspace,
		Task:          e.Task,
		StateBefore:   e.StateBefore,
		PolicyContext: e.PolicyContext,
		Evidence:      e.Evidence,
		Trace:         e.Trace,
		Decision:      e.Decision,
		Outcome:       e.Outcome,
		Provenance:    e.Provenance,
	}
	data, _ := json.Marshal(ce)
	return string(data)
}

func (e *SupervisionEpisode) VerifyEpisodeID() error {
	expected := e.ComputeEpisodeID()
	if e.EpisodeID != expected {
		return fmt.Errorf("%w: episode_id %s != computed %s", ErrHashMismatch, e.EpisodeID, expected)
	}
	return nil
}

type ContrastSet struct {
	SchemaVersion    int              `json:"schema_version"`
	ContrastSetID    string           `json:"contrast_set_id"`
	AnchorEpisodeID  string           `json:"anchor_episode_id"`
	VariantEpisodeID string           `json:"variant_episode_id"`
	Relation         ContrastRelation `json:"relation"`
	Intervention     InterventionInfo `json:"intervention"`
	Expected         ExpectedDiff     `json:"expected"`
	LabelSource      LabelSource      `json:"label_source"`
	Review           ReviewInfo       `json:"review"`
}

type InterventionInfo struct {
	Field          string `json:"field"`
	Before         string `json:"before"`
	After          string `json:"after"`
	SemanticDelta  string `json:"semantic_delta"`
}

type ExpectedDiff struct {
	AnchorAction     string   `json:"anchor_action"`
	VariantAction    string   `json:"variant_action"`
	InvariantFields  []string `json:"invariant_fields"`
}

type ReviewInfo struct {
	Status       ReviewStatus `json:"status"`
	Reviewer     string       `json:"reviewer"`
	RubricVersion string      `json:"rubric_version"`
}

func (c *ContrastSet) Validate() error {
	if c.SchemaVersion != SupervisionSchemaVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrInvalidSchemaVersion, c.SchemaVersion, SupervisionSchemaVersion)
	}
	if strings.TrimSpace(c.ContrastSetID) == "" {
		return fmt.Errorf("%w: contrast_set_id", ErrMissingRequiredField)
	}
	if strings.TrimSpace(c.AnchorEpisodeID) == "" {
		return fmt.Errorf("%w: anchor_episode_id", ErrMissingRequiredField)
	}
	if strings.TrimSpace(c.VariantEpisodeID) == "" {
		return fmt.Errorf("%w: variant_episode_id", ErrMissingRequiredField)
	}
	if c.AnchorEpisodeID == c.VariantEpisodeID {
		return errors.New("anchor and variant episode IDs must differ")
	}
	if c.Relation != ContrastMustChange && c.Relation != ContrastMustNotChange && c.Relation != ContrastMustAbstain {
		return fmt.Errorf("invalid contrast relation: %s", c.Relation)
	}
	if c.Intervention.Field == "" {
		return fmt.Errorf("%w: intervention.field", ErrMissingRequiredField)
	}
	if c.Expected.AnchorAction == "" || c.Expected.VariantAction == "" {
		return fmt.Errorf("%w: expected.anchor_action or expected.variant_action", ErrMissingRequiredField)
	}
	if c.LabelSource != LabelSourceDeterministicGate && c.LabelSource != LabelSourceHumanReview {
		return fmt.Errorf("invalid label_source: %s", c.LabelSource)
	}
	if c.Review.Status != ReviewStatusPending && c.Review.Status != ReviewStatusApproved && c.Review.Status != ReviewStatusRejected {
		return fmt.Errorf("invalid review status: %s", c.Review.Status)
	}
	if c.Review.RubricVersion == "" {
		return fmt.Errorf("%w: review.rubric_version", ErrMissingRequiredField)
	}
	return nil
}

func (c *ContrastSet) ComputeContrastSetID() string {
	type canonicalContrast struct {
		SchemaVersion    int              `json:"schema_version"`
		AnchorEpisodeID  string           `json:"anchor_episode_id"`
		VariantEpisodeID string           `json:"variant_episode_id"`
		Relation         ContrastRelation `json:"relation"`
		Intervention     InterventionInfo `json:"intervention"`
		Expected         ExpectedDiff     `json:"expected"`
		LabelSource      LabelSource      `json:"label_source"`
		Review           ReviewInfo       `json:"review"`
	}
	cc := canonicalContrast{
		SchemaVersion:    c.SchemaVersion,
		AnchorEpisodeID:  c.AnchorEpisodeID,
		VariantEpisodeID: c.VariantEpisodeID,
		Relation:         c.Relation,
		Intervention:     c.Intervention,
		Expected:         c.Expected,
		LabelSource:      c.LabelSource,
		Review:           c.Review,
	}
	data, _ := json.Marshal(cc)
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func (c *ContrastSet) VerifyContrastSetID() error {
	expected := c.ComputeContrastSetID()
	if c.ContrastSetID != expected {
		return fmt.Errorf("%w: contrast_set_id %s != computed %s", ErrHashMismatch, c.ContrastSetID, expected)
	}
	return nil
}

type DatasetManifest struct {
	SchemaVersion     int            `json:"schema_version"`
	ManifestID        string         `json:"manifest_id"`
	CreatedAt         time.Time      `json:"created_at"`
	PolicyBundleHash  string         `json:"policy_bundle_hash"`
	RedactionVersion  string         `json:"redaction_version"`
	EmbeddingModel    string         `json:"embedding_model"`
	EmbeddingVersion  string         `json:"embedding_version"`
	EmbeddingDims     int            `json:"embedding_dims"`
	TokenizerVersion  string         `json:"tokenizer_version"`
	EpisodeCount      int            `json:"episode_count"`
	ContrastSetCount  int            `json:"contrast_set_count"`
	EpisodeIDs        []string       `json:"episode_ids"`
	ContrastSetIDs    []string       `json:"contrast_set_ids"`
	SplitAssignments  map[string]SplitName `json:"split_assignments"`
	SourceHashes      []string       `json:"source_hashes"`
	License           string         `json:"license"`
}

func (m *DatasetManifest) Validate() error {
	if m.SchemaVersion != SupervisionSchemaVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrInvalidSchemaVersion, m.SchemaVersion, SupervisionSchemaVersion)
	}
	if strings.TrimSpace(m.ManifestID) == "" {
		return fmt.Errorf("%w: manifest_id", ErrMissingRequiredField)
	}
	if m.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_at", ErrMissingRequiredField)
	}
	if strings.TrimSpace(m.PolicyBundleHash) == "" {
		return fmt.Errorf("%w: policy_bundle_hash", ErrMissingRequiredField)
	}
	if strings.TrimSpace(m.RedactionVersion) == "" {
		return fmt.Errorf("%w: redaction_version", ErrMissingRequiredField)
	}
	if strings.TrimSpace(m.EmbeddingModel) == "" {
		return fmt.Errorf("%w: embedding_model", ErrMissingRequiredField)
	}
	if strings.TrimSpace(m.EmbeddingVersion) == "" {
		return fmt.Errorf("%w: embedding_version", ErrMissingRequiredField)
	}
	if m.EmbeddingDims <= 0 {
		return fmt.Errorf("%w: embedding_dims", ErrMissingRequiredField)
	}
	if strings.TrimSpace(m.TokenizerVersion) == "" {
		return fmt.Errorf("%w: tokenizer_version", ErrMissingRequiredField)
	}
	if m.EpisodeCount <= 0 {
		return fmt.Errorf("%w: episode_count", ErrMissingRequiredField)
	}
	if len(m.EpisodeIDs) != m.EpisodeCount {
		return fmt.Errorf("episode_ids length (%d) != episode_count (%d)", len(m.EpisodeIDs), m.EpisodeCount)
	}
	if len(m.ContrastSetIDs) != m.ContrastSetCount {
		return fmt.Errorf("contrast_set_ids length (%d) != contrast_set_count (%d)", len(m.ContrastSetIDs), m.ContrastSetCount)
	}
	if len(m.SplitAssignments) != m.EpisodeCount+m.ContrastSetCount {
		return fmt.Errorf("split_assignments length (%d) != total records (%d)", len(m.SplitAssignments), m.EpisodeCount+m.ContrastSetCount)
	}
	for id, split := range m.SplitAssignments {
		if split != SplitTrain && split != SplitValidation && split != SplitTest {
			return fmt.Errorf("invalid split for %s: %s", id, split)
		}
	}
	seen := make(map[string]bool)
	for _, id := range m.EpisodeIDs {
		if seen[id] {
			return fmt.Errorf("duplicate episode_id in manifest: %s", id)
		}
		seen[id] = true
	}
	for _, id := range m.ContrastSetIDs {
		if seen[id] {
			return fmt.Errorf("duplicate contrast_set_id in manifest: %s", id)
		}
		seen[id] = true
	}
	return nil
}

func (m *DatasetManifest) ComputeManifestID() string {
	type canonicalManifest struct {
		SchemaVersion     int            `json:"schema_version"`
		CreatedAt         time.Time      `json:"created_at"`
		PolicyBundleHash  string         `json:"policy_bundle_hash"`
		RedactionVersion  string         `json:"redaction_version"`
		EmbeddingModel    string         `json:"embedding_model"`
		EmbeddingVersion  string         `json:"embedding_version"`
		EmbeddingDims     int            `json:"embedding_dims"`
		TokenizerVersion  string         `json:"tokenizer_version"`
		EpisodeCount      int            `json:"episode_count"`
		ContrastSetCount  int            `json:"contrast_set_count"`
		EpisodeIDs        []string       `json:"episode_ids"`
		ContrastSetIDs    []string       `json:"contrast_set_ids"`
		SplitAssignments  map[string]SplitName `json:"split_assignments"`
		SourceHashes      []string       `json:"source_hashes"`
		License           string         `json:"license"`
	}
	
	sortedEpisodes := make([]string, len(m.EpisodeIDs))
	copy(sortedEpisodes, m.EpisodeIDs)
	sort.Strings(sortedEpisodes)
	
	sortedContrasts := make([]string, len(m.ContrastSetIDs))
	copy(sortedContrasts, m.ContrastSetIDs)
	sort.Strings(sortedContrasts)
	
	sortedSources := make([]string, len(m.SourceHashes))
	copy(sortedSources, m.SourceHashes)
	sort.Strings(sortedSources)
	
	sortedSplits := make([]string, 0, len(m.SplitAssignments))
	for k, v := range m.SplitAssignments {
		sortedSplits = append(sortedSplits, k+":"+string(v))
	}
	sort.Strings(sortedSplits)
	
	cm := canonicalManifest{
		SchemaVersion:    m.SchemaVersion,
		CreatedAt:        m.CreatedAt,
		PolicyBundleHash: m.PolicyBundleHash,
		RedactionVersion: m.RedactionVersion,
		EmbeddingModel:   m.EmbeddingModel,
		EmbeddingVersion: m.EmbeddingVersion,
		EmbeddingDims:    m.EmbeddingDims,
		TokenizerVersion: m.TokenizerVersion,
		EpisodeCount:     m.EpisodeCount,
		ContrastSetCount: m.ContrastSetCount,
		EpisodeIDs:       sortedEpisodes,
		ContrastSetIDs:   sortedContrasts,
		SplitAssignments: m.SplitAssignments,
		SourceHashes:     sortedSources,
		License:          m.License,
	}
	data, _ := json.Marshal(cm)
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func (m *DatasetManifest) VerifyManifestID() error {
	expected := m.ComputeManifestID()
	if m.ManifestID != expected {
		return fmt.Errorf("%w: manifest_id %s != computed %s", ErrHashMismatch, m.ManifestID, expected)
	}
	return nil
}

type SplitLock struct {
	SchemaVersion   int                 `json:"schema_version"`
	ManifestID      string              `json:"manifest_id"`
	SplitAssignments map[string]SplitName `json:"split_assignments"`
	CreatedAt       time.Time           `json:"created_at"`
}

func NewSplitLock(manifest *DatasetManifest) *SplitLock {
	assignments := make(map[string]SplitName, len(manifest.SplitAssignments))
	for k, v := range manifest.SplitAssignments {
		assignments[k] = v
	}
	return &SplitLock{
		SchemaVersion:    SupervisionSchemaVersion,
		ManifestID:       manifest.ManifestID,
		SplitAssignments: assignments,
		CreatedAt:        time.Now().UTC(),
	}
}

func (s *SplitLock) VerifyAgainstManifest(m *DatasetManifest) error {
	if s.ManifestID != m.ManifestID {
		return fmt.Errorf("split lock manifest_id mismatch: %s != %s", s.ManifestID, m.ManifestID)
	}
	if len(s.SplitAssignments) != len(m.SplitAssignments) {
		return fmt.Errorf("split lock assignment count mismatch")
	}
	for k, v := range m.SplitAssignments {
		if s.SplitAssignments[k] != v {
			return fmt.Errorf("split assignment mismatch for %s: lock=%s manifest=%s", k, s.SplitAssignments[k], v)
		}
	}
	return nil
}

type QuarantineRecord struct {
	SchemaVersion  int            `json:"schema_version"`
	RecordID       string         `json:"record_id"`
	RecordKind     string         `json:"record_kind"`
	OriginalData   json.RawMessage `json:"original_data"`
	Reason         string         `json:"reason"`
	DetectedAt     time.Time      `json:"detected_at"`
	RedactionStatus RedactionStatus `json:"redaction_status"`
}

type DataCard struct {
	ManifestID       string            `json:"manifest_id"`
	GeneratedAt      time.Time         `json:"generated_at"`
	EpisodeCount     int               `json:"episode_count"`
	ContrastSetCount int               `json:"contrast_set_count"`
	SplitCounts      map[SplitName]int `json:"split_counts"`
	IntentCounts     map[Intent]int    `json:"intent_counts"`
	RiskCounts       map[RiskLevel]int `json:"risk_counts"`
	OutcomeCounts    map[OutcomeStatus]int `json:"outcome_counts"`
	PolicyRefCounts  map[string]int    `json:"policy_ref_counts"`
	SourceLicenses   map[string]int    `json:"source_licenses"`
	RedactionVersion string            `json:"redaction_version"`
	EmbeddingModel   string            `json:"embedding_model"`
	TokenizerVersion string            `json:"tokenizer_version"`
	Notes            string            `json:"notes,omitempty"`
}