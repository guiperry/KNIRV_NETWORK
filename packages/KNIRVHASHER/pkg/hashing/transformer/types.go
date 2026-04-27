package transformer

type WASMType string

const (
	WASMTypeRule       WASMType = "rule"
	WASMTypeResolution WASMType = "resolution"
	WASMTypePatch      WASMType = "patch"
)

type PolicyBadgeInquiry struct {
	Name            string   `json:"name"`
	BadgeType       string   `json:"badge_type"`
	ValuesSignals   []string `json:"values_signals"`
	OntologySignals []string `json:"ontology_signals"`
	DVEContext      string   `json:"dve_context"`
}

type DVEErrorInquiry struct {
	DVESessionID string            `json:"dve_session_id"`
	ErrorType    string            `json:"error_type"`
	ErrorMessage string            `json:"error_message"`
	ErrorContext string            `json:"error_context"`
	StackTrace   string            `json:"stack_trace,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type SystemPatchInquiry struct {
	ComponentID string `json:"component_id"`
	ErrorCode   string `json:"error_code"`
	SystemState string `json:"system_state"`
}

type WASMDecision struct {
	WASMType            WASMType `json:"wasm_type"`
	WASMPath            string   `json:"wasm_path"`
	WASMHash            string   `json:"wasm_hash"`
	Rationale           string   `json:"rationale"`
	BidirectionalVerified bool   `json:"bidirectional_verified"`
	WazeroExecPassed    bool     `json:"wazero_exec_passed"`
	HashNetworkVerified bool     `json:"hash_network_verified,omitempty"`
	VerifierConfidence  float32  `json:"verifier_confidence,omitempty"`
	TurnCount           int      `json:"turn_count"`
	ForwardVerifierMsg  string   `json:"forward_verifier_msg,omitempty"`
	BackwardVerifierMsg string   `json:"backward_verifier_msg,omitempty"`
}