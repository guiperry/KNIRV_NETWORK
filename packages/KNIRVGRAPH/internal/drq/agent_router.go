package drq

import (
	"math"
)

// AgentRouter assigns agents to clusters via DRQ policy
type AgentRouter struct {
	availableAgents map[string]*AgentProfile
	clusterMgr      *DRQClusterManager
	drqSync         *DRQSyncProtocol
	topology        NetworkTopologyInterface // Use the interface
}

// RouteAgent selects optimal cluster using DRQ Q-values
func (ar *AgentRouter) RouteAgent(
	agentID string,
) (string, error) {
	agent := ar.availableAgents[agentID]
	
	// Get all active clusters (stub)
	clusters := ar.clusterMgr.GetActiveClusters() 
	
	var bestCluster string
	var bestQValue float64 = math.Inf(-1)
	
	for clusterID, cluster := range clusters {
		// Construct state
		state := ErrorClusterState{
			ClusterID:       clusterID,
			ClusterDensity:  float64(len(cluster.Errors)),
			ComplexityScore: cluster.AvgComplexity,
			AgentAssignments: cluster.AgentCounts,
		}
		
		// Construct action
		action := DRQAction{
			Type:          ASSIGN_NEW_AGENT,
			AgentID:       agentID,
			TargetCluster: clusterID,
		}
		
		// Retrieve Q-value
		qValue := ar.drqSync.GetQValue(state, action)
		
		// Apply specialization bonus (stub)
		if ar.matchesSpecialization(agent, cluster) { 
			qValue += 50.0
		}
		
		// Apply reputation scaling
		qValue *= agent.ReputationScore
		
		if qValue > bestQValue {
			bestQValue = qValue
			bestCluster = clusterID
		}
	}
	
	// Assign agent
	agent.CurrentCluster = bestCluster
	cluster := clusters[bestCluster]
	// Ensure cluster can be modified (this will be handled when ClusterManager is fully implemented)
	if cluster.AgentCounts == nil {
		cluster.AgentCounts = make(map[string]int)
	}
	cluster.AgentCounts[agentID]++ 
	
	return bestCluster, nil
}

// matchesSpecialization is a stub for checking if an agent matches a cluster's specialization
func (ar *AgentRouter) matchesSpecialization(agent *AgentProfile, cluster *ErrorCluster) bool {
	// TODO: Implement actual specialization matching logic
	_ = agent
	_ = cluster
	return true // Assume match for stub
}
