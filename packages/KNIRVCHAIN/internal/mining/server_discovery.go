package mining

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"KNIRVCHAIN/internal/graph"
	"KNIRVCHAIN/internal/types"
)

// ServerDiscoveryService handles MCP server registration and discovery
type ServerDiscoveryService struct {
	registeredServers map[string]*RegisteredServer
	graphQueries      *graph.GraphQueries
	mutex             sync.RWMutex
}

// RegisteredServer represents a registered MCP server
type RegisteredServer struct {
	CapabilityNode *types.CapabilityNode `json:"capability_node"`
	LastSeen       int64                 `json:"last_seen"`
	Status         ServerStatus          `json:"status"`
	HealthScore    float64               `json:"health_score"`
}

// ServerStatus represents the status of a registered server
type ServerStatus string

const (
	ServerStatusActive   ServerStatus = "active"
	ServerStatusInactive ServerStatus = "inactive"
	ServerStatusUnhealthy ServerStatus = "unhealthy"
)

// NewServerDiscoveryService creates a new server discovery service
func NewServerDiscoveryService(graphQueries *graph.GraphQueries) *ServerDiscoveryService {
	return &ServerDiscoveryService{
		registeredServers: make(map[string]*RegisteredServer),
		graphQueries:      graphQueries,
	}
}

// RegisterServer registers an MCP server for discovery
func (sds *ServerDiscoveryService) RegisterServer(capabilityNodeID string) error {
	sds.mutex.Lock()
	defer sds.mutex.Unlock()

	// Get the capability node
	capabilityNode, err := sds.graphQueries.GetCapabilityNode(capabilityNodeID)
	if err != nil {
		return fmt.Errorf("capability node not found: %w", err)
	}

	// Check if already registered
	if _, exists := sds.registeredServers[capabilityNodeID]; exists {
		// Update last seen
		sds.registeredServers[capabilityNodeID].LastSeen = time.Now().Unix()
		sds.registeredServers[capabilityNodeID].Status = ServerStatusActive
		log.Printf("Server registration updated: %s", capabilityNodeID)
		return nil
	}

	// Register new server
	registeredServer := &RegisteredServer{
		CapabilityNode: capabilityNode,
		LastSeen:       time.Now().Unix(),
		Status:         ServerStatusActive,
		HealthScore:    1.0, // Start with perfect health
	}

	sds.registeredServers[capabilityNodeID] = registeredServer

	log.Printf("Server registered: %s (%s) at %s", capabilityNodeID, capabilityNode.CapabilityType, capabilityNode.MCPPointer.EndpointURI)
	return nil
}

// UnregisterServer unregisters an MCP server
func (sds *ServerDiscoveryService) UnregisterServer(capabilityNodeID string) error {
	sds.mutex.Lock()
	defer sds.mutex.Unlock()

	if _, exists := sds.registeredServers[capabilityNodeID]; !exists {
		return fmt.Errorf("server not registered: %s", capabilityNodeID)
	}

	delete(sds.registeredServers, capabilityNodeID)
	log.Printf("Server unregistered: %s", capabilityNodeID)
	return nil
}

// DiscoverServers discovers available servers by capability type
func (sds *ServerDiscoveryService) DiscoverServers(capabilityType types.CapabilityType, limit int) ([]*DiscoveryResult, error) {
	sds.mutex.RLock()
	defer sds.mutex.RUnlock()

	var results []*DiscoveryResult

	for _, registered := range sds.registeredServers {
		if registered.CapabilityNode.CapabilityType == capabilityType &&
		   registered.Status == ServerStatusActive {

			results = append(results, &DiscoveryResult{
				CapabilityNode: registered.CapabilityNode,
				LastSeen:       registered.LastSeen,
				HealthScore:    registered.HealthScore,
			})

			if limit > 0 && len(results) >= limit {
				break
			}
		}
	}

	return results, nil
}

// DiscoveryResult represents a discovered server
type DiscoveryResult struct {
	CapabilityNode *types.CapabilityNode `json:"capability_node"`
	LastSeen       int64                 `json:"last_seen"`
	HealthScore    float64               `json:"health_score"`
}

// GetServerByID gets a registered server by capability node ID
func (sds *ServerDiscoveryService) GetServerByID(capabilityNodeID string) (*RegisteredServer, error) {
	sds.mutex.RLock()
	defer sds.mutex.RUnlock()

	server, exists := sds.registeredServers[capabilityNodeID]
	if !exists {
		return nil, fmt.Errorf("server not found: %s", capabilityNodeID)
	}

	return server, nil
}

// UpdateServerHealth updates the health status of a registered server
func (sds *ServerDiscoveryService) UpdateServerHealth(capabilityNodeID string, healthScore float64) error {
	sds.mutex.Lock()
	defer sds.mutex.Unlock()

	server, exists := sds.registeredServers[capabilityNodeID]
	if !exists {
		return fmt.Errorf("server not registered: %s", capabilityNodeID)
	}

	server.HealthScore = healthScore
	server.LastSeen = time.Now().Unix()

	// Update status based on health score
	if healthScore >= 0.8 {
		server.Status = ServerStatusActive
	} else if healthScore >= 0.5 {
		server.Status = ServerStatusUnhealthy
	} else {
		server.Status = ServerStatusInactive
	}

	log.Printf("Server health updated: %s (score: %.2f, status: %s)", capabilityNodeID, healthScore, server.Status)
	return nil
}

// Heartbeat sends a heartbeat for a registered server
func (sds *ServerDiscoveryService) Heartbeat(capabilityNodeID string) error {
	return sds.UpdateServerHealth(capabilityNodeID, 1.0)
}

// CleanupInactiveServers removes servers that haven't been seen recently
func (sds *ServerDiscoveryService) CleanupInactiveServers(maxAge time.Duration) int {
	sds.mutex.Lock()
	defer sds.mutex.Unlock()

	cutoffTime := time.Now().Add(-maxAge).Unix()
	removed := 0

	for id, server := range sds.registeredServers {
		if server.LastSeen < cutoffTime {
			delete(sds.registeredServers, id)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("Cleaned up %d inactive servers", removed)
	}

	return removed
}

// GetServerStatistics returns statistics about registered servers
func (sds *ServerDiscoveryService) GetServerStatistics() (*ServerStatistics, error) {
	sds.mutex.RLock()
	defer sds.mutex.RUnlock()

	stats := &ServerStatistics{
		TotalServers:      len(sds.registeredServers),
		ServersByType:     make(map[string]int),
		ServersByStatus:   make(map[string]int),
		AverageHealthScore: 0.0,
	}

	totalHealth := 0.0

	for _, server := range sds.registeredServers {
		// Count by type
		typeKey := string(server.CapabilityNode.CapabilityType)
		stats.ServersByType[typeKey]++

		// Count by status
		statusKey := string(server.Status)
		stats.ServersByStatus[statusKey]++

		// Accumulate health scores
		totalHealth += server.HealthScore
	}

	if stats.TotalServers > 0 {
		stats.AverageHealthScore = totalHealth / float64(stats.TotalServers)
	}

	return stats, nil
}

// ServerStatistics represents statistics about registered servers
type ServerStatistics struct {
	TotalServers       int                `json:"total_servers"`
	ServersByType      map[string]int     `json:"servers_by_type"`
	ServersByStatus    map[string]int     `json:"servers_by_status"`
	AverageHealthScore float64            `json:"average_health_score"`
}

// StartHealthMonitoring starts background health monitoring
func (sds *ServerDiscoveryService) StartHealthMonitoring(ctx context.Context, checkInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sds.performHealthChecks(ctx)
			}
		}
	}()
}

// performHealthChecks performs health checks on all registered servers
func (sds *ServerDiscoveryService) performHealthChecks(ctx context.Context) {
	sds.mutex.RLock()
	serverIDs := make([]string, 0, len(sds.registeredServers))
	for id := range sds.registeredServers {
		serverIDs = append(serverIDs, id)
	}
	sds.mutex.RUnlock()

	for _, serverID := range serverIDs {
		select {
		case <-ctx.Done():
			return
		default:
			// Perform health check
			healthScore := sds.checkServerHealth(ctx, serverID)
			sds.UpdateServerHealth(serverID, healthScore)
		}
	}
}

// checkServerHealth performs a health check on a specific server
func (sds *ServerDiscoveryService) checkServerHealth(ctx context.Context, serverID string) float64 {
	// This would perform an actual health check (e.g., MCP ping, response time test)
	// For now, simulate a health check

	// Simulate some servers being unhealthy
	// In reality, this would test the MCP endpoint
	if len(serverID) % 3 == 0 { // Every third server is "unhealthy"
		return 0.3
	}

	return 0.95 // Most servers are healthy
}

// FindBestServer finds the best server for a given capability type
func (sds *ServerDiscoveryService) FindBestServer(capabilityType types.CapabilityType) (*DiscoveryResult, error) {
	servers, err := sds.DiscoverServers(capabilityType, 0)
	if err != nil {
		return nil, err
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found for capability type: %s", capabilityType)
	}

	// Find server with best health score
	var bestServer *DiscoveryResult
	bestScore := -1.0

	for _, server := range servers {
		if server.HealthScore > bestScore {
			bestScore = server.HealthScore
			bestServer = server
		}
	}

	return bestServer, nil
}

// BroadcastServerUpdate broadcasts server status updates
func (sds *ServerDiscoveryService) BroadcastServerUpdate(serverID string) error {
	// This would broadcast updates to interested parties
	// For now, just log
	log.Printf("Server update broadcasted: %s", serverID)
	return nil
}

// SearchServers performs advanced search on registered servers
func (sds *ServerDiscoveryService) SearchServers(query *ServerSearchQuery) ([]*DiscoveryResult, error) {
	sds.mutex.RLock()
	defer sds.mutex.RUnlock()

	var results []*DiscoveryResult

	for _, registered := range sds.registeredServers {
		if sds.matchesSearchQuery(registered, query) {
			results = append(results, &DiscoveryResult{
				CapabilityNode: registered.CapabilityNode,
				LastSeen:       registered.LastSeen,
				HealthScore:    registered.HealthScore,
			})

			if query.Limit > 0 && len(results) >= query.Limit {
				break
			}
		}
	}

	return results, nil
}

// ServerSearchQuery represents a search query for servers
type ServerSearchQuery struct {
	CapabilityType types.CapabilityType `json:"capability_type,omitempty"`
	MinHealthScore float64              `json:"min_health_score,omitempty"`
	MaxNRNCost     uint64               `json:"max_nrn_cost,omitempty"`
	Limit          int                  `json:"limit,omitempty"`
}

// matchesSearchQuery checks if a server matches the search query
func (sds *ServerDiscoveryService) matchesSearchQuery(server *RegisteredServer, query *ServerSearchQuery) bool {
	// Check capability type
	if query.CapabilityType != "" && server.CapabilityNode.CapabilityType != query.CapabilityType {
		return false
	}

	// Check minimum health score
	if query.MinHealthScore > 0 && server.HealthScore < query.MinHealthScore {
		return false
	}

	// Check maximum NRN cost
	if query.MaxNRNCost > 0 && server.CapabilityNode.NRNCost > query.MaxNRNCost {
		return false
	}

	return true
}