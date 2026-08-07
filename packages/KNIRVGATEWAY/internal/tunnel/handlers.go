package tunnel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler handles HTTP API requests for the tunnel registry
type Handler struct {
	registryManager *RegistryManager
	tunnelManager   *TunnelManager
	config          *Config
	logger          *zap.Logger
	startTime       time.Time
}

// Config holds configuration for the tunnel handler
type Config struct {
	HTTPAPIPort         int
	ControlListenerPort int
	PublicRelayPort     int
	STUNPort            int
	ServerPublicHost    string
	RelayServerPeerID   string
}

// NewHandler creates a new tunnel handler
func NewHandler(registryManager *RegistryManager, tunnelManager *TunnelManager, config *Config, logger *zap.Logger) *Handler {
	return &Handler{
		registryManager: registryManager,
		tunnelManager:   tunnelManager,
		config:          config,
		logger:          logger,
		startTime:       time.Now(),
	}
}

// RegisterRoutes registers the tunnel registry routes
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Status endpoint
	r.HandleFunc("/status", h.handleStatus).Methods("GET")

	// Registry routes
	r.HandleFunc("/api/registry/register", h.handleRegister).Methods("POST")
	r.HandleFunc("/api/registry/nodes", h.handleGetNodes).Methods("GET")
	r.HandleFunc("/api/registry/node/dev/{devId}", h.handleGetNodeByDevID).Methods("GET")
	r.HandleFunc("/api/registry/node/chain/{chainId}", h.handleGetNodeByChainID).Methods("GET")

	// URI routes
	r.HandleFunc("/api/uri/generate", h.handleGenerateURI).Methods("POST")
	r.HandleFunc("/api/uri/resolve", h.handleResolveURI).Methods("GET")
}

// handleStatus returns the status of the tunnel registry service
func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	uptimeMs := now.Sub(h.startTime).Milliseconds()
	uptimeDays := int(uptimeMs / (1000 * 60 * 60 * 24))
	uptimeHours := int((uptimeMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
	uptimeMinutes := int((uptimeMs % (1000 * 60 * 60)) / (1000 * 60))
	uptimeSeconds := int((uptimeMs % (1000 * 60)) / 1000)
	uptimeString := fmt.Sprintf("%dd %dh %dm %ds", uptimeDays, uptimeHours, uptimeMinutes, uptimeSeconds)

	// Count active connections
	connectionRegistryMu.RLock()
	activeConnections := 0
	registeredConnections := len(connectionRegistry)
	for _, info := range connectionRegistry {
		if now.Sub(info.LastSeen) < 60*time.Minute {
			activeConnections++
		}
	}
	connectionRegistryMu.RUnlock()

	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := StatusResponse{
		Status:                "online",
		Version:               "1.0.0",
		Uptime:                uptimeString,
		UptimeMs:              uptimeMs,
		RegisteredConnections: registeredConnections,
		ActiveConnections:     activeConnections,
		MemoryUsage: MemoryUsage{
			RSS:       m.Sys,
			HeapTotal: m.HeapSys,
			HeapUsed:  m.HeapAlloc,
		},
		HTTPAPIPort:         h.config.HTTPAPIPort,
		ControlListenerPort: h.config.ControlListenerPort,
		PublicRelayPort:     h.config.PublicRelayPort,
		Timestamp:           now.Format(time.RFC3339),
	}

	// Check if request is from a browser
	acceptHeader := r.Header.Get("Accept")
	isHTMLRequest := strings.Contains(acceptHeader, "text/html")

	if isHTMLRequest {
		h.handleStatusHTML(w, status)
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// handleStatusHTML returns HTML status page
func (h *Handler) handleStatusHTML(w http.ResponseWriter, status StatusResponse) {
	connectionRegistryMu.RLock()
	defer connectionRegistryMu.RUnlock()

	var connectionRows strings.Builder
	for connectionID, info := range connectionRegistry {
		now := time.Now()
		isActive := now.Sub(info.LastSeen) < 60*time.Minute
		statusClass := "inactive"
		if isActive {
			statusClass = "active"
		}

		connectionRows.WriteString(fmt.Sprintf(`
		<tr>
			<td class="connectionid">%s</td>
			<td>%s</td>
			<td>%s</td>
			<td>%d</td>
			<td class="%s">%s</td>
			<td>%s</td>
		</tr>`,
			connectionID,
			info.Type,
			info.SourceIP,
			info.SourcePort,
			statusClass,
			map[bool]string{true: "Active", false: "Inactive"}[isActive],
			info.LastSeen.Format("2006-01-02 15:04:05"),
		))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>KNIRVORACLE Central Relay Status</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 1200px; margin: 0 auto; padding: 20px; }
		h1 { color: #2c3e50; border-bottom: 2px solid #eee; padding-bottom: 10px; }
		.status-card { background: #f8f9fa; border-radius: 8px; padding: 20px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		.status-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 15px; }
		.status-item { background: white; padding: 15px; border-radius: 6px; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
		.status-label { font-weight: bold; color: #6c757d; margin-bottom: 5px; font-size: 0.9rem; }
		.status-value { font-size: 1.1rem; color: #212529; }
		table { width: 100%%; border-collapse: collapse; margin-top: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
		th, td { padding: 12px 15px; text-align: left; border-bottom: 1px solid #e9ecef; }
		th { background-color: #f8f9fa; font-weight: bold; color: #495057; }
		tr:hover { background-color: #f8f9fa; }
		.active { color: #28a745; font-weight: bold; }
		.inactive { color: #dc3545; }
		.connectionid { font-family: monospace; word-break: break-all; }
		.refresh-button { background-color: #007bff; color: white; border: none; padding: 8px 16px; border-radius: 4px; cursor: pointer; font-size: 14px; margin-bottom: 20px; }
		.refresh-button:hover { background-color: #0069d9; }
		.memory-usage { font-size: 0.9rem; }
		@media (max-width: 768px) { .status-grid { grid-template-columns: 1fr; } table { font-size: 0.9rem; } th, td { padding: 8px 10px; } }
	</style>
</head>
<body>
	<h1>KNIRVORACLE Central Relay Status</h1>
	<button class="refresh-button" onclick="window.location.reload()">Refresh Status</button>
	<div class="status-card">
		<div class="status-grid">
			<div class="status-item"><div class="status-label">Status</div><div class="status-value">%s</div></div>
			<div class="status-item"><div class="status-label">Version</div><div class="status-value">%s</div></div>
			<div class="status-item"><div class="status-label">Uptime</div><div class="status-value">%s</div></div>
			<div class="status-item"><div class="status-label">Registered Connections</div><div class="status-value">%d</div></div>
			<div class="status-item"><div class="status-label">Active Connections</div><div class="status-value">%d</div></div>
			<div class="status-item"><div class="status-label">HTTP API Port</div><div class="status-value">%d</div></div>
			<div class="status-item"><div class="status-label">Control Listener Port</div><div class="status-value">%d</div></div>
			<div class="status-item"><div class="status-label">Public Relay Port</div><div class="status-value">%d</div></div>
			<div class="status-item"><div class="status-label">Last Updated</div><div class="status-value">%s</div></div>
		</div>
		<div class="status-item memory-usage">
			<div class="status-label">Memory Usage</div>
			<div class="status-value">RSS: %d MB | Heap Total: %d MB | Heap Used: %d MB</div>
		</div>
	</div>
	<h2>Active Connections</h2>
	<table>
		<thead><tr><th>Connection ID</th><th>Type</th><th>Source IP</th><th>Source Port</th><th>Status</th><th>Last Seen</th></tr></thead>
		<tbody>%s</tbody>
	</table>
	<script>setTimeout(() => { window.location.reload(); }, 30000);</script>
</body>
</html>`,
		status.Status,
		status.Version,
		status.Uptime,
		status.RegisteredConnections,
		status.ActiveConnections,
		status.HTTPAPIPort,
		status.ControlListenerPort,
		status.PublicRelayPort,
		time.Now().Format("2006-01-02 15:04:05"),
		status.MemoryUsage.RSS/1024/1024,
		status.MemoryUsage.HeapTotal/1024/1024,
		status.MemoryUsage.HeapUsed/1024/1024,
		connectionRows.String(),
	)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleRegister handles node registration via API
func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DevID         string `json:"devId"`
		ChainID       string `json:"chainId"`
		PublicIP      string `json:"publicIp"`
		PublicP2PPort int    `json:"publicP2pPort"`
		Type          string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DevID == "" || req.PublicIP == "" || req.PublicP2PPort == 0 {
		http.Error(w, "Missing required fields: devId, publicIp, publicP2pPort", http.StatusBadRequest)
		return
	}

	nodeInfo := h.registryManager.RegisterNodeViaAPI(req.DevID, req.ChainID, req.PublicIP, req.PublicP2PPort, req.Type)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Node registered successfully via API",
		"nodeInfo": nodeInfo,
	})
}

// handleGetNodes returns all registered nodes
func (h *Handler) handleGetNodes(w http.ResponseWriter, r *http.Request) {
	nodes := h.registryManager.GetAllNodes()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

// handleGetNodeByDevID returns a node by dev ID
func (h *Handler) handleGetNodeByDevID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	devId := vars["devId"]

	node := h.registryManager.GetNodeByPeerId(devId)
	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

// handleGetNodeByChainID returns a node by chain ID
func (h *Handler) handleGetNodeByChainID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainId := vars["chainId"]

	node := h.registryManager.GetNodeByChainId(chainId)
	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

// handleGenerateURI generates a URI for a resource
func (h *Handler) handleGenerateURI(w http.ResponseWriter, r *http.Request) {
	var req URIGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.DevID == "" {
		http.Error(w, "devId is required", http.StatusBadRequest)
		return
	}

	nodeInfo := h.registryManager.GetNodeByPeerId(req.DevID)
	if nodeInfo == nil {
		http.Error(w, fmt.Sprintf("Node with devId %s not registered", req.DevID), http.StatusNotFound)
		return
	}

	resourceType := req.ResourceType
	if resourceType == "" {
		resourceType = "dev"
	}

	// Generate unique resource ID (simplified - in production use UUID)
	resourceId := fmt.Sprintf("%s-%d", req.DevID, time.Now().UnixNano())

	// Map the resource to the node
	h.registryManager.MapTunneledResource(resourceId, req.DevID)

	var uri string
	if nodeInfo.IsTunneled {
		uri = fmt.Sprintf("knirv://%s/%s.%s", h.config.ServerPublicHost, resourceId, resourceType)
		if req.SubPath != "" {
			uri += "/" + strings.TrimPrefix(req.SubPath, "/")
		}
	} else {
		host := nodeInfo.PublicIP
		if host == "" {
			host = nodeInfo.DevID
		}
		uri = fmt.Sprintf("knirv://%s/%s.%s", host, resourceId, resourceType)
		if req.SubPath != "" {
			uri += "/" + strings.TrimPrefix(req.SubPath, "/")
		}
	}

	response := URIGenerateResponse{
		URI:          uri,
		ResourceID:   resourceId,
		ResourceType: resourceType,
		SubPath:      req.SubPath,
	}

	if !nodeInfo.IsTunneled {
		response.DirectInfo = &DirectConnectionInfo{
			IP:   nodeInfo.PublicIP,
			Port: nodeInfo.PublicP2PPort,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleResolveURI resolves a URI to connection details
func (h *Handler) handleResolveURI(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	if uri == "" {
		http.Error(w, "uri query parameter is required", http.StatusBadRequest)
		return
	}

	// Parse URI
	if !strings.HasPrefix(uri, "knirv://") {
		http.Error(w, "Invalid URI scheme", http.StatusBadRequest)
		return
	}

	pathPart := uri[len("knirv://"):]
	slashIndex := strings.Index(pathPart, "/")
	if slashIndex == -1 {
		http.Error(w, "Invalid URI format: missing path separator", http.StatusBadRequest)
		return
	}

	authority := pathPart[:slashIndex]
	remainingPath := pathPart[slashIndex+1:]

	// Extract identifier and resourceType
	dotIndex := strings.LastIndex(remainingPath, ".")
	if dotIndex == -1 {
		http.Error(w, "URI missing resource type", http.StatusBadRequest)
		return
	}

	identifier := remainingPath[:dotIndex]
	resourceType := remainingPath[dotIndex+1]

	// Check for tunneled resource
	targetPeerId := h.registryManager.GetPeerIdForTunneledResource(identifier)
	var connectionDetails interface{}

	if targetPeerId != "" {
		nodeInfo := h.registryManager.GetNodeByPeerId(targetPeerId)
		if nodeInfo != nil {
			if nodeInfo.IsTunneled {
				connectionDetails = map[string]interface{}{
					"connectionType":    "TUNNELED",
					"targetPeerId":      targetPeerId,
					"tunnelServerHost":  h.config.ServerPublicHost,
					"tunnelServerPort":  h.config.PublicRelayPort,
					"relayProtocolInfo": "Send targetPeerId as first line (JSON: {\"targetPeerId\":\"...\"}) after connecting to tunnel server.",
				}
			} else if nodeInfo.PublicIP != "" && nodeInfo.PublicP2PPort != 0 {
				multiaddress := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", nodeInfo.PublicIP, nodeInfo.PublicP2PPort, nodeInfo.DevID)
				connectionDetails = map[string]interface{}{
					"connectionType": "DIRECT_P2P",
					"targetPeerId":   nodeInfo.DevID,
					"multiaddress":   multiaddress,
				}
			}
		}
	}

	// Check for direct dev ID or chain ID
	if connectionDetails == nil {
		nodeInfo := h.registryManager.GetNodeByPeerId(identifier)
		if nodeInfo == nil {
			nodeInfo = h.registryManager.GetNodeByChainId(identifier)
		}

		if nodeInfo != nil {
			if nodeInfo.IsTunneled {
				if h.config.RelayServerPeerID == "" {
					http.Error(w, "gateway relay identity unavailable", http.StatusServiceUnavailable)
					return
				}
				relaySegment := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", h.config.ServerPublicHost, h.config.PublicRelayPort, h.config.RelayServerPeerID)
				fullMultiaddress := fmt.Sprintf("%s/p2p-circuit/p2p/%s", relaySegment, nodeInfo.DevID)
				connectionDetails = map[string]interface{}{
					"connectionType": "RELAYED_P2P_CIRCUIT",
					"targetPeerId":   nodeInfo.DevID,
					"multiaddress":   fullMultiaddress,
				}
			} else if nodeInfo.PublicIP != "" && nodeInfo.PublicP2PPort != 0 {
				multiaddress := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", nodeInfo.PublicIP, nodeInfo.PublicP2PPort, nodeInfo.DevID)
				connectionDetails = map[string]interface{}{
					"connectionType": "DIRECT_P2P",
					"targetPeerId":   nodeInfo.DevID,
					"multiaddress":   multiaddress,
				}
			}
		}
	}

	if connectionDetails == nil {
		http.Error(w, fmt.Sprintf("Resource with identifier '%s' not found", identifier), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"originalUri":        uri,
		"resolvedIdentifier": identifier,
		"resourceType":       resourceType,
		"authority":          authority,
	}

	// Merge connection details
	if connMap, ok := connectionDetails.(map[string]interface{}); ok {
		for k, v := range connMap {
			response[k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
