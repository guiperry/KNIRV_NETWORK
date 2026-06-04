package mcphardening

import (
	"fmt"
	"sync"
	"time"
)

type ToolCallStatus string

const (
	ToolCallStatusAllowed  ToolCallStatus = "allowed"
	ToolCallStatusDenied   ToolCallStatus = "denied"
	ToolCallStatusFlagged  ToolCallStatus = "flagged"
	ToolCallStatusBlocked  ToolCallStatus = "blocked"
)

type PoisoningLevel int

const (
	PoisoningLevelNone      PoisoningLevel = iota
	PoisoningLevelSuspicious
	PoisoningLevelConfirmed
	PoisoningLevelCritical
)

type ToolDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parameters  []string `json:"parameters"`
	Required    bool     `json:"required"`
	AllowedIPs  []string `json:"allowed_ips,omitempty"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	MaxArgs     int      `json:"max_args,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
}

type ToolCallRecord struct {
	ID           string            `json:"id"`
	AgentID      string            `json:"agent_id"`
	NodeID       string            `json:"node_id"`
	ToolName     string            `json:"tool_name"`
	Arguments    map[string]interface{} `json:"arguments"`
	Status       ToolCallStatus    `json:"status"`
	Reason       string            `json:"reason,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
	Duration     time.Duration     `json:"duration,omitempty"`
	TokenCount   int               `json:"token_count,omitempty"`
	PoisoningLevel PoisoningLevel  `json:"poisoning_level,omitempty"`
}

type PoisoningSignature struct {
	Pattern     string            `json:"pattern"`
	Confidence  float64           `json:"confidence"`
	Indicators  []string          `json:"indicators"`
	Severity    PoisoningLevel    `json:"severity"`
}

type MCPEndpoint struct {
	URL          string   `json:"url"`
	AllowedTools []string `json:"allowed_tools"`
	RateLimit    int      `json:"rate_limit"`
	AuthRequired bool     `json:"auth_required"`
}

type ToolCallValidator struct {
	mu           sync.RWMutex
	tools        map[string]*ToolDefinition
	allowedCalls map[string]int
	deniedCalls  map[string]int
	maxCallsPerMin int
}

type PoisoningDetector struct {
	mu           sync.RWMutex
	signatures   []PoisoningSignature
	suspiciousCalls map[string][]string
	threshold    float64
}

type MCPGateway struct {
	mu         sync.RWMutex
	endpoints  map[string]*MCPEndpoint
	auditLog   []*ToolCallRecord
	validator  *ToolCallValidator
	detector   *PoisoningDetector
}

func NewToolCallValidator() *ToolCallValidator {
	return &ToolCallValidator{
		tools:          make(map[string]*ToolDefinition),
		allowedCalls:   make(map[string]int),
		deniedCalls:    make(map[string]int),
		maxCallsPerMin: 100,
	}
}

func (tv *ToolCallValidator) RegisterTool(tool *ToolDefinition) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	tv.tools[tool.Name] = tool
}

func (tv *ToolCallValidator) GetTool(name string) (*ToolDefinition, bool) {
	tv.mu.RLock()
	defer tv.mu.RUnlock()
	tool, ok := tv.tools[name]
	return tool, ok
}

func (tv *ToolCallValidator) ListTools() []*ToolDefinition {
	tv.mu.RLock()
	defer tv.mu.RUnlock()
	result := make([]*ToolDefinition, 0, len(tv.tools))
	for _, tool := range tv.tools {
		result = append(result, tool)
	}
	return result
}

func (tv *ToolCallValidator) ValidateCall(agentID, toolName string, args map[string]interface{}) (ToolCallStatus, string) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	tv.allowedCalls[agentID]++

	tool, ok := tv.tools[toolName]
	if !ok {
		return ToolCallStatusFlagged, "unknown tool: " + toolName
	}

	if tool.Required && args == nil {
		return ToolCallStatusDenied, "required tool missing arguments"
	}

	if tool.MaxArgs > 0 && len(args) > tool.MaxArgs {
		return ToolCallStatusDenied, fmt.Sprintf("exceeded max %d arguments", tool.MaxArgs)
	}

	totalAgentCalls := tv.allowedCalls[agentID]
	if totalAgentCalls > tv.maxCallsPerMin {
		return ToolCallStatusBlocked, "rate limit exceeded"
	}

	return ToolCallStatusAllowed, ""
}

func (tv *ToolCallValidator) RecordDeniedCall(agentID string) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	tv.deniedCalls[agentID]++
}

func (tv *ToolCallValidator) SetMaxCallsPerMin(max int) {
	tv.mu.Lock()
	defer tv.mu.Unlock()
	tv.maxCallsPerMin = max
}

func (tv *ToolCallValidator) GetAgentCallStats(agentID string) map[string]int {
	tv.mu.RLock()
	defer tv.mu.RUnlock()
	return map[string]int{
		"allowed": tv.allowedCalls[agentID],
		"denied":  tv.deniedCalls[agentID],
	}
}

func NewPoisoningDetector() *PoisoningDetector {
	return &PoisoningDetector{
		signatures:      make([]PoisoningSignature, 0),
		suspiciousCalls: make(map[string][]string),
		threshold:       0.7,
	}
}

func (pd *PoisoningDetector) RegisterSignature(sig PoisoningSignature) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.signatures = append(pd.signatures, sig)
}

func (pd *PoisoningDetector) AnalyzeCall(agentID, toolName string, args map[string]interface{}) (PoisoningLevel, []string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	var indicators []string
	maxSeverity := PoisoningLevelNone

	for _, sig := range pd.signatures {
		if sig.Pattern == toolName || pd.matchesToolName(sig.Pattern, toolName) {
			matchConfidence := pd.calculateMatchConfidence(sig, args)
			if matchConfidence >= pd.threshold {
				indicators = append(indicators, sig.Indicators...)
				if pd.severityOrder(sig.Severity) > pd.severityOrder(maxSeverity) {
					maxSeverity = sig.Severity
				}
			}
		}
	}

	if len(indicators) > 0 {
		pd.suspiciousCalls[agentID] = append(pd.suspiciousCalls[agentID], toolName)
		return maxSeverity, indicators
	}

	return PoisoningLevelNone, nil
}

func (pd *PoisoningDetector) GetSuspiciousCallCount(agentID string) int {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	return len(pd.suspiciousCalls[agentID])
}

func (pd *PoisoningDetector) SetThreshold(t float64) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.threshold = t
}

func (pd *PoisoningDetector) calculateMatchConfidence(sig PoisoningSignature, args map[string]interface{}) float64 {
	return sig.Confidence
}

func (pd *PoisoningDetector) matchesToolName(pattern, toolName string) bool {
	return pattern == toolName
}

func (pd *PoisoningDetector) severityOrder(level PoisoningLevel) int {
	return int(level)
}

func NewMCPGateway() *MCPGateway {
	return &MCPGateway{
		endpoints: make(map[string]*MCPEndpoint),
		auditLog:  make([]*ToolCallRecord, 0),
		validator: NewToolCallValidator(),
		detector:  NewPoisoningDetector(),
	}
}

func (gw *MCPGateway) Validator() *ToolCallValidator {
	return gw.validator
}

func (gw *MCPGateway) Detector() *PoisoningDetector {
	return gw.detector
}

func (gw *MCPGateway) RegisterEndpoint(name string, ep *MCPEndpoint) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	gw.endpoints[name] = ep
}

func (gw *MCPGateway) GetEndpoint(name string) (*MCPEndpoint, bool) {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	ep, ok := gw.endpoints[name]
	return ep, ok
}

func (gw *MCPGateway) ListEndpoints() []*MCPEndpoint {
	gw.mu.RLock()
	defer gw.mu.RUnlock()
	result := make([]*MCPEndpoint, 0, len(gw.endpoints))
	for _, ep := range gw.endpoints {
		result = append(result, ep)
	}
	return result
}

func (gw *MCPGateway) ProcessToolCall(record *ToolCallRecord) ToolCallStatus {
	status, reason := gw.validator.ValidateCall(record.AgentID, record.ToolName, record.Arguments)
	record.Status = status
	record.Reason = reason
	record.Timestamp = time.Now().UTC()

	if status == ToolCallStatusAllowed || status == ToolCallStatusFlagged {
		poisonLevel, indicators := gw.detector.AnalyzeCall(record.AgentID, record.ToolName, record.Arguments)
		record.PoisoningLevel = poisonLevel
		if poisonLevel >= PoisoningLevelConfirmed {
			record.Status = ToolCallStatusBlocked
			record.Reason = fmt.Sprintf("poisoning detected: %v", indicators)
		}
	}

	gw.mu.Lock()
	gw.auditLog = append(gw.auditLog, record)
	gw.mu.Unlock()

	return record.Status
}

func (gw *MCPGateway) GetAuditLog(agentID string, limit int) []*ToolCallRecord {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	var result []*ToolCallRecord
	for _, rec := range gw.auditLog {
		if agentID == "" || rec.AgentID == agentID {
			result = append(result, rec)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

func (gw *MCPGateway) GetStatistics() map[string]interface{} {
	gw.mu.RLock()
	defer gw.mu.RUnlock()

	statusCounts := map[ToolCallStatus]int{}
	poisonCounts := map[PoisoningLevel]int{}
	for _, rec := range gw.auditLog {
		statusCounts[rec.Status]++
		poisonCounts[rec.PoisoningLevel]++
	}

	return map[string]interface{}{
		"total_calls":          len(gw.auditLog),
		"status_distribution":  statusCounts,
		"poisoning_distribution": poisonCounts,
		"registered_tools":     len(gw.validator.tools),
		"registered_endpoints": len(gw.endpoints),
		"registered_signatures": len(gw.detector.signatures),
	}
}
