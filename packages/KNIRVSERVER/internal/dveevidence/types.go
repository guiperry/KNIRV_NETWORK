package dveevidence

import "encoding/json"

const SchemaVersion = "dve.bundle.v1"

const (
	AlgorithmEd25519 = "ed25519"
)

type Signature struct {
	KeyID     string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type ToolRun struct {
	Tool        string   `json:"tool"`
	Version     string   `json:"version,omitempty"`
	Args        []string `json:"args,omitempty"`
	StartedAt   string   `json:"started_at,omitempty"`
	CompletedAt string   `json:"completed_at,omitempty"`
	ExitCode    int      `json:"exit_code,omitempty"`
	EventLogRef string   `json:"event_log_ref,omitempty"`
}

type PermissionDecision struct {
	ID           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	EventType    string `json:"event_type"`
	Action       string `json:"action"`
	Input        string `json:"input,omitempty"`
	PolicyHash   string `json:"policy_hash"`
	MatchedRule  string `json:"matched_rule,omitempty"`
	InputHash    string `json:"input_hash,omitempty"`
	DecisionHash string `json:"decision_hash"`
	ApproverID   string `json:"approver_id,omitempty"`
	Denied       bool   `json:"denied"`
}

type ArtifactRef struct {
	Name  string `json:"name"`
	Path  string `json:"path,omitempty"`
	Class string `json:"class"`
	Hash  string `json:"hash"`
	Size  int64  `json:"size,omitempty"`
}

type MemvidRef struct {
	MediaRef    string  `json:"media_ref"`
	IndexRef    string  `json:"index_ref,omitempty"`
	DurationSec float64 `json:"duration_sec,omitempty"`
}

type Bundle struct {
	SchemaVersion       string               `json:"schema_version"`
	SessionID           string               `json:"session_id"`
	DVEID               string               `json:"dve_id"`
	UserID              string               `json:"user_id"`
	ProjectID           string               `json:"project_id"`
	StartedAt           string               `json:"started_at"`
	CompletedAt         string               `json:"completed_at"`
	WorkspaceBaseHash   string               `json:"workspace_base_hash"`
	WorkspaceFinalHash  string               `json:"workspace_final_hash"`
	PolicyHash          string               `json:"policy_hash"`
	ToolRuns            []ToolRun            `json:"tool_runs"`
	PermissionDecisions []PermissionDecision `json:"permission_decisions"`
	Artifacts           []ArtifactRef        `json:"artifacts"`
	MemvidRefs          []MemvidRef          `json:"memvid_refs"`
	EventLogRoot        string               `json:"eventlog_root"`
	ArtifactMerkleRoot  string               `json:"artifact_merkle_root"`
	Signature           *Signature           `json:"signature,omitempty"`
}

type Event struct {
	Index     int             `json:"index"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	PrevHash  string          `json:"prev_hash,omitempty"`
	Hash      string          `json:"hash,omitempty"`
}

type Evidence struct {
	Events         []Event  `json:"events,omitempty"`
	ArtifactHashes []string `json:"artifact_hashes,omitempty"`
}
