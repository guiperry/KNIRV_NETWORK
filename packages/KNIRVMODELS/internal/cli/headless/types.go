package headless

// Request/response types for headless API

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

// StatusResponse represents the status response
type StatusResponse struct {
	DriverRunning bool   `json:"driver_running"`
	ServerReady   bool   `json:"server_ready"`
	DeviceIP      string `json:"device_ip,omitempty"`
	DeviceType    string `json:"device_type,omitempty"`
	Uptime        string `json:"uptime"`
}

// PipelineRequest represents a pipeline request
type PipelineRequest struct {
	Type string `json:"type"` // goat, arxiv, demo
}

// VerifyRequest represents a verification request
type VerifyRequest struct {
	Mode string `json:"mode"` // semantic, mathematical
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response  string  `json:"response"`
	Timestamp string  `json:"timestamp"`
}

// RulesRequest represents a rules management request
type RulesRequest struct {
	Action    string   `json:"action"` // list, add, delete
	Domain    string   `json:"domain,omitempty"`
	RuleType  string   `json:"rule_type,omitempty"`
	Conclusion string   `json:"conclusion,omitempty"`
	Index     int      `json:"index,omitempty"`
}

// RulesResponse represents a rules list response
type RulesResponse struct {
	Domain string                   `json:"domain,omitempty"`
	Rules  []map[string]interface{} `json:"rules,omitempty"`
	Message string                   `json:"message,omitempty"`
}

// DiscoveryResponse represents discovery results
type DiscoveryResponse struct {
	Devices      []map[string]interface{} `json:"devices"`
	SelectedIP   string                 `json:"selected_ip,omitempty"`
	SelectedType string                 `json:"selected_type,omitempty"`
	Output       string                 `json:"output"`
	Duration     float64                `json:"duration"`
}

// ProvisionResponse represents provisioning results
type ProvisionResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Message string `json:"message,omitempty"`
}

// GenericResponse is a generic success/error response
type GenericResponse struct {
	Message string      `json:"message"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
