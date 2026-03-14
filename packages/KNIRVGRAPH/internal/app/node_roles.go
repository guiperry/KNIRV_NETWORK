package app

// NodeType defines the role of a node in the KNIRVGRAPH network
type NodeType int

const (
	NODE_FULL NodeType = iota
	NODE_HUB
	NODE_LIGHT
	NODE_DVE
)

// NodeConfig holds configuration specific to a node's role
type NodeConfig struct {
	Type NodeType // The type of this node

	// General Node properties
	EnableFullState bool // For Full and Hub Nodes
	IsValidator     bool // For Full Nodes
	EnableDRQSync   bool // For Full and Hub Nodes

	// Hub Node specific properties
	HostsClustersThreshold int // Minimum clusters to host to be considered a hub
	MinPageRankThreshold   float64 // Minimum PageRank to be considered a hub

	// Light Node specific properties
	QueryOnly bool // Light nodes only query, do not store full state

	// DVE Node specific properties
	EnableSandbox   bool   // DVE Nodes have sandboxed execution
	NRNStake        uint64 // NRN tokens staked for DVE validation
	HasGPUOrTPU     bool   // Indicates specialized hardware for training
}

// GetDefaultNodeConfig returns a default configuration for a given node type
func GetDefaultNodeConfig(nodeType NodeType) NodeConfig {
	switch nodeType {
	case NODE_FULL:
		return NodeConfig{
			Type: NODE_FULL,
			EnableFullState: true,
			IsValidator: true,
			EnableDRQSync: true,
			QueryOnly: false,
			EnableSandbox: false,
			NRNStake: 0,
			HasGPUOrTPU: false,
		}
	case NODE_HUB:
		return NodeConfig{
			Type: NODE_HUB,
			EnableFullState: true,
			IsValidator: false, // Hub nodes might not be validators themselves, but coordinate
			EnableDRQSync: true,
			HostsClustersThreshold: 50,
			MinPageRankThreshold: 0.1, // Example threshold
			QueryOnly: false,
			EnableSandbox: false,
			NRNStake: 0,
			HasGPUOrTPU: false,
		}
	case NODE_LIGHT:
		return NodeConfig{
			Type: NODE_LIGHT,
			EnableFullState: false,
			IsValidator: false,
			EnableDRQSync: false,
			QueryOnly: true,
			EnableSandbox: false,
			NRNStake: 0,
			HasGPUOrTPU: false,
		}
	case NODE_DVE:
		return NodeConfig{
			Type: NODE_DVE,
			EnableFullState: false,
			IsValidator: false,
			EnableDRQSync: false,
			QueryOnly: false,
			EnableSandbox: true,
			NRNStake: 10000, // Example NRN stake
			HasGPUOrTPU: true,
		}
	default:
		return NodeConfig{Type: NODE_FULL} // Default to full node if unknown
	}
}
