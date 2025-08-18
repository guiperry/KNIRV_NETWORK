package desktop

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MCPServer implements Model Context Protocol for agent communication
type MCPServer struct {
	// Core components
	desktopHost *DesktopHost
	hrmEngine   *HRMEngine

	// MCP protocol state
	capabilities MCPCapabilities
	tools        map[string]*MCPTool
	resources    map[string]*MCPResource
	prompts      map[string]*MCPPrompt

	// Connection management
	clients  map[string]*MCPClient
	upgrader websocket.Upgrader

	// Synchronization
	mutex   sync.RWMutex
	running bool
}

// MCPCapabilities defines what the server can do
type MCPCapabilities struct {
	Logging      *MCPLoggingCapability   `json:"logging,omitempty"`
	Prompts      *MCPPromptsCapability   `json:"prompts,omitempty"`
	Resources    *MCPResourcesCapability `json:"resources,omitempty"`
	Tools        *MCPToolsCapability     `json:"tools,omitempty"`
	Experimental map[string]interface{}  `json:"experimental,omitempty"`
}

type MCPLoggingCapability struct {
	Level string `json:"level"`
}

type MCPPromptsCapability struct {
	ListChanged bool `json:"listChanged"`
}

type MCPResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`
	ListChanged bool `json:"listChanged"`
}

type MCPToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// MCPTool represents a tool available to agents
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     func(args map[string]interface{}) (*MCPToolResult, error)
}

type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}

// MCPResource represents a resource available to agents
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Handler     func() (*MCPResourceContent, error)
}

type MCPResourceContent struct {
	Contents []MCPContent `json:"contents"`
}

// MCPPrompt represents a prompt template
type MCPPrompt struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
	Handler     func(args map[string]interface{}) (*MCPPromptResult, error)
}

type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type MCPPromptResult struct {
	Description string       `json:"description,omitempty"`
	Messages    []MCPMessage `json:"messages"`
}

type MCPMessage struct {
	Role    string     `json:"role"`
	Content MCPContent `json:"content"`
}

// MCPClient represents a connected MCP client
type MCPClient struct {
	ID           string
	Connection   *websocket.Conn
	Capabilities MCPCapabilities
	LastPing     time.Time
}

// NewMCPServer creates a new MCP server
func NewMCPServer(desktopHost *DesktopHost, hrmEngine *HRMEngine) *MCPServer {
	server := &MCPServer{
		desktopHost: desktopHost,
		hrmEngine:   hrmEngine,
		capabilities: MCPCapabilities{
			Logging:   &MCPLoggingCapability{Level: "info"},
			Prompts:   &MCPPromptsCapability{ListChanged: true},
			Resources: &MCPResourcesCapability{Subscribe: true, ListChanged: true},
			Tools:     &MCPToolsCapability{ListChanged: true},
			Experimental: map[string]interface{}{
				"hrm_cognitive_processing": true,
				"qr_linkage_system":        true,
				"personality_adaptation":   true,
			},
		},
		tools:     make(map[string]*MCPTool),
		resources: make(map[string]*MCPResource),
		prompts:   make(map[string]*MCPPrompt),
		clients:   make(map[string]*MCPClient),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	// Register default tools, resources, and prompts
	server.registerDefaultTools()
	server.registerDefaultResources()
	server.registerDefaultPrompts()

	return server
}

// Start starts the MCP server
func (mcp *MCPServer) Start() error {
	mcp.mutex.Lock()
	defer mcp.mutex.Unlock()

	if mcp.running {
		return fmt.Errorf("MCP server already running")
	}

	mcp.running = true
	log.Printf("MCP server started with capabilities: %+v", mcp.capabilities)

	return nil
}

// Stop stops the MCP server
func (mcp *MCPServer) Stop() error {
	mcp.mutex.Lock()
	defer mcp.mutex.Unlock()

	if !mcp.running {
		return fmt.Errorf("MCP server not running")
	}

	// Close all client connections
	for _, client := range mcp.clients {
		client.Connection.Close()
	}

	mcp.running = false
	log.Printf("MCP server stopped")

	return nil
}

// HandleWebSocket handles MCP WebSocket connections
func (mcp *MCPServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := mcp.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("MCP WebSocket upgrade failed: %v", err)
		return
	}

	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &MCPClient{
		ID:         clientID,
		Connection: conn,
		LastPing:   time.Now(),
	}

	mcp.mutex.Lock()
	mcp.clients[clientID] = client
	mcp.mutex.Unlock()

	defer func() {
		mcp.mutex.Lock()
		delete(mcp.clients, clientID)
		mcp.mutex.Unlock()
		conn.Close()
	}()

	log.Printf("MCP client connected: %s", clientID)

	// Send initial capabilities
	mcp.sendCapabilities(client)

	// Handle messages
	for {
		var message map[string]interface{}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("MCP client %s disconnected: %v", clientID, err)
			break
		}

		mcp.handleMessage(client, message)
	}
}

// registerDefaultTools registers the default MCP tools
func (mcp *MCPServer) registerDefaultTools() {
	// HRM Cognitive Processing Tool
	mcp.tools["hrm_process"] = &MCPTool{
		Name:        "hrm_process",
		Description: "Process input through HRM cognitive engine",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sensory_data": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "Array of sensory input values",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Context for the processing task",
				},
				"task_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of cognitive task to perform",
				},
			},
			"required": []string{"sensory_data", "context", "task_type"},
		},
		Handler: mcp.handleHRMProcess,
	}

	// QR Code Generation Tool
	mcp.tools["generate_qr"] = &MCPTool{
		Name:        "generate_qr",
		Description: "Generate QR code for device linkage",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"target_assignment", "transaction_sign"},
					"description": "Type of QR code to generate",
				},
				"target_id": map[string]interface{}{
					"type":        "string",
					"description": "Target system ID (for target_assignment)",
				},
				"transaction_data": map[string]interface{}{
					"type":        "object",
					"description": "Transaction data (for transaction_sign)",
				},
			},
			"required": []string{"type"},
		},
		Handler: mcp.handleGenerateQR,
	}

	// System Status Tool
	mcp.tools["system_status"] = &MCPTool{
		Name:        "system_status",
		Description: "Get current system status and health",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: mcp.handleSystemStatus,
	}
}

// registerDefaultResources registers the default MCP resources
func (mcp *MCPServer) registerDefaultResources() {
	mcp.resources["hrm_model_info"] = &MCPResource{
		URI:         "knirv://hrm/model_info",
		Name:        "HRM Model Information",
		Description: "Information about the loaded HRM cognitive model",
		MimeType:    "application/json",
		Handler:     mcp.getHRMModelInfo,
	}

	mcp.resources["system_capabilities"] = &MCPResource{
		URI:         "knirv://system/capabilities",
		Name:        "System Capabilities",
		Description: "Available system capabilities and features",
		MimeType:    "application/json",
		Handler:     mcp.getSystemCapabilities,
	}
}

// registerDefaultPrompts registers the default MCP prompts
func (mcp *MCPServer) registerDefaultPrompts() {
	mcp.prompts["cognitive_analysis"] = &MCPPrompt{
		Name:        "cognitive_analysis",
		Description: "Analyze input using HRM cognitive processing",
		Arguments: []MCPPromptArgument{
			{Name: "input_data", Description: "Data to analyze", Required: true},
			{Name: "analysis_type", Description: "Type of analysis to perform", Required: false},
		},
		Handler: mcp.handleCognitiveAnalysisPrompt,
	}
}

// MCP message handlers
func (mcp *MCPServer) sendCapabilities(client *MCPClient) {
	message := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    mcp.capabilities,
			"serverInfo": map[string]interface{}{
				"name":    "KNIRV Desktop Host",
				"version": "1.0.0",
			},
		},
	}

	client.Connection.WriteJSON(message)
}

func (mcp *MCPServer) handleMessage(client *MCPClient, message map[string]interface{}) {
	method, ok := message["method"].(string)
	if !ok {
		return
	}

	switch method {
	case "tools/list":
		mcp.handleToolsList(client, message)
	case "tools/call":
		mcp.handleToolsCall(client, message)
	case "resources/list":
		mcp.handleResourcesList(client, message)
	case "resources/read":
		mcp.handleResourcesRead(client, message)
	case "prompts/list":
		mcp.handlePromptsList(client, message)
	case "prompts/get":
		mcp.handlePromptsGet(client, message)
	case "ping":
		mcp.handlePing(client, message)
	default:
		log.Printf("Unknown MCP method: %s", method)
	}
}

// Tool handlers
func (mcp *MCPServer) handleHRMProcess(args map[string]interface{}) (*MCPToolResult, error) {
	sensoryData, ok := args["sensory_data"].([]interface{})
	if !ok {
		return &MCPToolResult{IsError: true, Content: []MCPContent{{Type: "text", Text: "Invalid sensory_data"}}}, nil
	}

	context, _ := args["context"].(string)
	taskType, _ := args["task_type"].(string)

	// Convert sensory data to float32 slice
	sensoryFloat := make([]float32, len(sensoryData))
	for i, v := range sensoryData {
		if f, ok := v.(float64); ok {
			sensoryFloat[i] = float32(f)
		}
	}

	// Process through HRM
	input := &HRMInput{
		SensoryData: sensoryFloat,
		Context:     context,
		TaskType:    taskType,
	}

	output, err := mcp.hrmEngine.ProcessCognitiveInput(input)
	if err != nil {
		return &MCPToolResult{IsError: true, Content: []MCPContent{{Type: "text", Text: err.Error()}}}, nil
	}

	resultJSON, _ := json.Marshal(output)
	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (mcp *MCPServer) handleGenerateQR(args map[string]interface{}) (*MCPToolResult, error) {
	qrType, ok := args["type"].(string)
	if !ok {
		return &MCPToolResult{IsError: true, Content: []MCPContent{{Type: "text", Text: "Invalid QR type"}}}, nil
	}

	var qrCode *QRCode
	var err error

	switch qrType {
	case "target_assignment":
		targetID, _ := args["target_id"].(string)
		if targetID == "" {
			targetID = "default_target"
		}
		qrCode, err = mcp.desktopHost.GetQRLinkage().GenerateTargetAssignmentQR(targetID, []string{"agent_deployment"})
	case "transaction_sign":
		// Mock transaction data for now
		txData := &TransactionData{
			Hash:      "0x1234567890abcdef",
			Amount:    "1.0 ETH",
			Recipient: "0xabcdef1234567890",
			GasFee:    "0.001 ETH",
			Timestamp: time.Now().Unix(),
		}
		qrCode, err = mcp.desktopHost.GetQRLinkage().GenerateTransactionSignQR(txData)
	default:
		return &MCPToolResult{IsError: true, Content: []MCPContent{{Type: "text", Text: "Unknown QR type"}}}, nil
	}

	if err != nil {
		return &MCPToolResult{IsError: true, Content: []MCPContent{{Type: "text", Text: err.Error()}}}, nil
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"session_id": qrCode.SessionID,
		"expires_at": qrCode.ExpiresAt,
		"qr_data":    string(qrCode.Data),
	})

	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(resultJSON)}},
	}, nil
}

func (mcp *MCPServer) handleSystemStatus(args map[string]interface{}) (*MCPToolResult, error) {
	status := map[string]interface{}{
		"desktop_id":         mcp.desktopHost.GetDesktopID(),
		"hrm_initialized":    mcp.hrmEngine.IsInitialized(),
		"mobile_connections": len(mcp.desktopHost.mobileConnections),
		"agent_sessions":     len(mcp.desktopHost.agentSessions),
		"mcp_clients":        len(mcp.clients),
		"timestamp":          time.Now().Unix(),
	}

	resultJSON, _ := json.Marshal(status)
	return &MCPToolResult{
		Content: []MCPContent{{Type: "text", Text: string(resultJSON)}},
	}, nil
}

// Resource handlers
func (mcp *MCPServer) getHRMModelInfo() (*MCPResourceContent, error) {
	info := mcp.hrmEngine.GetModelInfo()
	infoJSON, _ := json.Marshal(info)

	return &MCPResourceContent{
		Contents: []MCPContent{{Type: "text", Text: string(infoJSON)}},
	}, nil
}

func (mcp *MCPServer) getSystemCapabilities() (*MCPResourceContent, error) {
	capabilities := map[string]interface{}{
		"hrm_processing":         true,
		"qr_linkage":             true,
		"mobile_integration":     true,
		"personality_adaptation": true,
		"secure_communication":   true,
		"wasm_runtime":           true,
	}

	capJSON, _ := json.Marshal(capabilities)
	return &MCPResourceContent{
		Contents: []MCPContent{{Type: "text", Text: string(capJSON)}},
	}, nil
}

// Prompt handlers
func (mcp *MCPServer) handleCognitiveAnalysisPrompt(args map[string]interface{}) (*MCPPromptResult, error) {
	inputData, _ := args["input_data"].(string)
	analysisType, _ := args["analysis_type"].(string)

	if analysisType == "" {
		analysisType = "general"
	}

	prompt := fmt.Sprintf(
		"Analyze the following input using HRM cognitive processing:\n\nInput: %s\nAnalysis Type: %s\n\nProvide a detailed cognitive analysis including reasoning patterns, confidence levels, and insights.",
		inputData,
		analysisType,
	)

	return &MCPPromptResult{
		Description: "Cognitive analysis prompt for HRM processing",
		Messages: []MCPMessage{
			{
				Role:    "user",
				Content: MCPContent{Type: "text", Text: prompt},
			},
		},
	}, nil
}

// Protocol message handlers
func (mcp *MCPServer) handleToolsList(client *MCPClient, message map[string]interface{}) {
	tools := make([]map[string]interface{}, 0, len(mcp.tools))
	for _, tool := range mcp.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result": map[string]interface{}{
			"tools": tools,
		},
	}

	client.Connection.WriteJSON(response)
}

func (mcp *MCPServer) handleToolsCall(client *MCPClient, message map[string]interface{}) {
	params, ok := message["params"].(map[string]interface{})
	if !ok {
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		return
	}

	args, _ := params["arguments"].(map[string]interface{})

	tool, exists := mcp.tools[name]
	if !exists {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Tool not found",
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	result, err := tool.Handler(args)
	if err != nil {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]interface{}{
				"code":    -32603,
				"message": err.Error(),
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result":  result,
	}

	client.Connection.WriteJSON(response)
}

func (mcp *MCPServer) handleResourcesList(client *MCPClient, message map[string]interface{}) {
	resources := make([]map[string]interface{}, 0, len(mcp.resources))
	for _, resource := range mcp.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         resource.URI,
			"name":        resource.Name,
			"description": resource.Description,
			"mimeType":    resource.MimeType,
		})
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result": map[string]interface{}{
			"resources": resources,
		},
	}

	client.Connection.WriteJSON(response)
}

func (mcp *MCPServer) handleResourcesRead(client *MCPClient, message map[string]interface{}) {
	params, ok := message["params"].(map[string]interface{})
	if !ok {
		return
	}

	uri, ok := params["uri"].(string)
	if !ok {
		return
	}

	var resource *MCPResource
	for _, r := range mcp.resources {
		if r.URI == uri {
			resource = r
			break
		}
	}

	if resource == nil {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Resource not found",
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	content, err := resource.Handler()
	if err != nil {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]interface{}{
				"code":    -32603,
				"message": err.Error(),
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result":  content,
	}

	client.Connection.WriteJSON(response)
}

func (mcp *MCPServer) handlePromptsList(client *MCPClient, message map[string]interface{}) {
	prompts := make([]map[string]interface{}, 0, len(mcp.prompts))
	for _, prompt := range mcp.prompts {
		prompts = append(prompts, map[string]interface{}{
			"name":        prompt.Name,
			"description": prompt.Description,
			"arguments":   prompt.Arguments,
		})
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result": map[string]interface{}{
			"prompts": prompts,
		},
	}

	client.Connection.WriteJSON(response)
}

func (mcp *MCPServer) handlePromptsGet(client *MCPClient, message map[string]interface{}) {
	params, ok := message["params"].(map[string]interface{})
	if !ok {
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		return
	}

	args, _ := params["arguments"].(map[string]interface{})

	prompt, exists := mcp.prompts[name]
	if !exists {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Prompt not found",
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	result, err := prompt.Handler(args)
	if err != nil {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      message["id"],
			"error": map[string]interface{}{
				"code":    -32603,
				"message": err.Error(),
			},
		}
		client.Connection.WriteJSON(response)
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result":  result,
	}

	client.Connection.WriteJSON(response)
}

func (mcp *MCPServer) handlePing(client *MCPClient, message map[string]interface{}) {
	client.LastPing = time.Now()

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      message["id"],
		"result":  map[string]interface{}{},
	}

	client.Connection.WriteJSON(response)
}
