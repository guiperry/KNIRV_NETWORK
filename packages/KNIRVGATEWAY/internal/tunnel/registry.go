package tunnel

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RegistryManager manages node registrations and resource mappings
type RegistryManager struct {
	nodes             map[string]*NodeInfo
	chainIdToPeerId   map[string]string
	tunneledResources map[string]string // resourceId -> devId
	mu                sync.RWMutex
	logger            *zap.Logger
}

// NewRegistryManager creates a new registry manager
func NewRegistryManager(logger *zap.Logger) *RegistryManager {
	return &RegistryManager{
		nodes:             make(map[string]*NodeInfo),
		chainIdToPeerId:   make(map[string]string),
		tunneledResources: make(map[string]string),
		logger:            logger,
	}
}

// RegisterNodeViaControlSocket registers a node via control socket
func (rm *RegistryManager) RegisterNodeViaControlSocket(devId, chainId, internalIp string, internalP2pPort int, nodeType, controlSocketId, serverPublicHost string, publicRelayPort int) *NodeInfo {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	nodeInfo := &NodeInfo{
		DevID:           devId,
		ChainID:         chainId,
		InternalIP:      internalIp,
		InternalP2PPort: internalP2pPort,
		Type:            nodeType,
		LastSeen:        time.Now(),
		ControlSocketID: controlSocketId,
		IsTunneled:      true,
		PublicRelayURL:  fmt.Sprintf("tcp://%s:%d/p2p_tunnel/%s", serverPublicHost, publicRelayPort, devId),
	}

	rm.nodes[devId] = nodeInfo
	if chainId != "" {
		rm.chainIdToPeerId[chainId] = devId
	}

	rm.logger.Info("Node registered via control socket",
		zap.String("devId", devId),
		zap.String("chainId", chainId),
		zap.String("type", nodeType))

	return nodeInfo
}

// RegisterNodeViaAPI registers a node via HTTP API
func (rm *RegistryManager) RegisterNodeViaAPI(devId, chainId, publicIp string, publicP2pPort int, nodeType string) *NodeInfo {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	nodeInfo := &NodeInfo{
		DevID:         devId,
		ChainID:       chainId,
		PublicIP:      publicIp,
		PublicP2PPort: publicP2pPort,
		Type:          nodeType,
		LastSeen:      time.Now(),
		IsTunneled:    false,
	}

	rm.nodes[devId] = nodeInfo
	if chainId != "" {
		rm.chainIdToPeerId[chainId] = devId
	}

	rm.logger.Info("Node registered via API",
		zap.String("devId", devId),
		zap.String("chainId", chainId),
		zap.String("type", nodeType))

	return nodeInfo
}

// MapTunneledResource maps a resource ID to a dev ID
func (rm *RegistryManager) MapTunneledResource(resourceId, devId string) string {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.tunneledResources[resourceId] = devId
	return resourceId
}

// GetPeerIdForTunneledResource gets the dev ID for a tunneled resource
func (rm *RegistryManager) GetPeerIdForTunneledResource(resourceId string) string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.tunneledResources[resourceId]
}

// GetNodeByPeerId gets a node by peer ID
func (rm *RegistryManager) GetNodeByPeerId(devId string) *NodeInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.nodes[devId]
}

// GetNodeByChainId gets a node by chain ID
func (rm *RegistryManager) GetNodeByChainId(chainId string) *NodeInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	devId := rm.chainIdToPeerId[chainId]
	if devId == "" {
		return nil
	}

	return rm.nodes[devId]
}

// GetAllNodes gets all registered nodes
func (rm *RegistryManager) GetAllNodes() []*NodeInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	nodes := make([]*NodeInfo, 0, len(rm.nodes))
	for _, node := range rm.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// DeregisterNodeByControlSocket deregisters a node by control socket ID
func (rm *RegistryManager) DeregisterNodeByControlSocket(controlSocketId string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for devId, nodeInfo := range rm.nodes {
		if nodeInfo.ControlSocketID == controlSocketId {
			delete(rm.nodes, devId)
			if nodeInfo.ChainID != "" {
				delete(rm.chainIdToPeerId, nodeInfo.ChainID)
			}
			rm.logger.Info("Node deregistered",
				zap.String("devId", devId),
				zap.String("controlSocketId", controlSocketId))
			break
		}
	}
}

// UpdateNodeLastSeen updates the last seen time for a node
func (rm *RegistryManager) UpdateNodeLastSeen(devId string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if node, exists := rm.nodes[devId]; exists {
		node.LastSeen = time.Now()
	}
}

// RegisterBootnode registers a bootnode for DVE installation
func (rm *RegistryManager) RegisterBootnode(nodeID, chainID, ip string, port int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	nodeInfo := &NodeInfo{
		DevID:          nodeID,
		ChainID:        chainID,
		PublicIP:       ip,
		PublicP2PPort:  port,
		Type:           "bootnode",
		LastSeen:       time.Now(),
		IsTunneled:     false,
		IsBootnode:     true,
	}

	rm.nodes[nodeID] = nodeInfo
	if chainID != "" {
		rm.chainIdToPeerId[chainID] = nodeID
	}

	rm.logger.Info("Bootnode registered",
		zap.String("nodeId", nodeID),
		zap.String("chainId", chainID),
		zap.String("ip", ip),
		zap.Int("port", port))

	return nil
}

// GetBootnodes returns all registered bootnodes
func (rm *RegistryManager) GetBootnodes() ([]*NodeInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	bootnodes := make([]*NodeInfo, 0)
	for _, node := range rm.nodes {
		if node.IsBootnode {
			bootnodes = append(bootnodes, node)
		}
	}

	return bootnodes, nil
}

// GetBootnodeByChainID returns a bootnode by chain ID
func (rm *RegistryManager) GetBootnodeByChainID(chainID string) (*NodeInfo, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	devID := rm.chainIdToPeerId[chainID]
	if devID == "" {
		return nil, fmt.Errorf("bootnode not found for chain ID: %s", chainID)
	}

	node, ok := rm.nodes[devID]
	if !ok || !node.IsBootnode {
		return nil, fmt.Errorf("bootnode not found for chain ID: %s", chainID)
	}

	return node, nil
}
