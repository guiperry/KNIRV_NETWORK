package compliance

import "time"

type FrameworkID string

const (
	FrameworkSOC2   FrameworkID = "soc2"
	FrameworkGDPR   FrameworkID = "gdpr"
	FrameworkHIPAA  FrameworkID = "hipaa"
	FrameworkPCI    FrameworkID = "pci-dss"
	FrameworkKNIRV  FrameworkID = "knirv-network"
)

type Control struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
}

type Framework struct {
	ID        FrameworkID `json:"id"`
	Name      string      `json:"name"`
	Version   string      `json:"version"`
	Controls  []Control   `json:"controls"`
}

type ControlStatus struct {
	ControlID     string    `json:"control_id"`
	FrameworkID   FrameworkID `json:"framework_id"`
	NodeID        string    `json:"node_id"`
	Status        string    `json:"status"`
	LastChecked   time.Time `json:"last_checked"`
	Evidence      string    `json:"evidence,omitempty"`
}

type FrameworkManager struct {
	frameworks map[FrameworkID]*Framework
	statuses   map[string]*ControlStatus
}

func NewFrameworkManager() *FrameworkManager {
	fm := &FrameworkManager{
		frameworks: make(map[FrameworkID]*Framework),
		statuses:   make(map[string]*ControlStatus),
	}
	fm.registerDefaults()
	return fm
}

func (fm *FrameworkManager) registerDefaults() {
	fm.frameworks[FrameworkKNIRV] = &Framework{
		ID:      FrameworkKNIRV,
		Name:    "KNIRV Network Security Framework",
		Version: "1.0.0",
		Controls: []Control{
			{ID: "KNIRV-01", Name: "DID Identity Binding", Description: "Every agent must have a valid DID", Category: "identity"},
			{ID: "KNIRV-02", Name: "Policy Enforcement", Description: "All actions evaluated against active policies", Category: "policy"},
			{ID: "KNIRV-03", Name: "Audit Logging", Description: "All governance events must be chain-hashed", Category: "audit"},
			{ID: "KNIRV-04", Name: "Revocation Checking", Description: "Revoked identities must be rejected", Category: "identity"},
			{ID: "KNIRV-05", Name: "Resource Guardrails", Description: "Resource usage must stay within configured limits", Category: "resource"},
			{ID: "KNIRV-06", Name: "Poisoning Detection", Description: "MCP tool calls must be scanned for poisoning", Category: "security"},
			{ID: "KNIRV-07", Name: "Circuit Breaking", Description: "Failure thresholds must trigger circuit breakers", Category: "reliability"},
		},
	}
}

func (fm *FrameworkManager) GetFramework(id FrameworkID) (*Framework, bool) {
	fw, ok := fm.frameworks[id]
	return fw, ok
}

func (fm *FrameworkManager) ListFrameworks() []*Framework {
	result := make([]*Framework, 0, len(fm.frameworks))
	for _, fw := range fm.frameworks {
		result = append(result, fw)
	}
	return result
}

func (fm *FrameworkManager) RegisterFramework(fw *Framework) {
	fm.frameworks[fw.ID] = fw
}

func (fm *FrameworkManager) UpdateControlStatus(status *ControlStatus) {
	key := string(status.FrameworkID) + ":" + status.ControlID + ":" + status.NodeID
	fm.statuses[key] = status
}

func (fm *FrameworkManager) GetControlStatus(fwID FrameworkID, controlID, nodeID string) (*ControlStatus, bool) {
	key := string(fwID) + ":" + controlID + ":" + nodeID
	s, ok := fm.statuses[key]
	return s, ok
}

func (fm *FrameworkManager) GetNodeCompliance(nodeID string) map[FrameworkID][]ControlStatus {
	result := make(map[FrameworkID][]ControlStatus)
	for _, s := range fm.statuses {
		if s.NodeID == nodeID {
			result[s.FrameworkID] = append(result[s.FrameworkID], *s)
		}
	}
	return result
}
