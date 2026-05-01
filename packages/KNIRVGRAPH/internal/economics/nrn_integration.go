package economics

import (
	"log"
	"math/big"
	"sync"
	"time"
)

// NRNIntegration handles NRN token economics for KNIRVGRAPH.
// Rewards and bounties are tracked locally on the graph — no oracle calls.
type NRNIntegration struct {
	enabled         bool
	mutex           sync.RWMutex
	economicMetrics *EconomicMetrics
}

// EconomicMetrics tracks economic activity in KNIRVGRAPH
type EconomicMetrics struct {
	TotalNRVsCreated   int64     `json:"total_nrvs_created"`
	TotalSkillsInvoked int64     `json:"total_skills_invoked"`
	TotalNRNRewards    *big.Int  `json:"total_nrn_rewards"`
	TotalNRNBurned     *big.Int  `json:"total_nrn_burned"`
	ActiveErrorNodes   int64     `json:"active_error_nodes"`
	ActiveSkillNodes   int64     `json:"active_skill_nodes"`
	NetworkEfficiency  float64   `json:"network_efficiency"`
	LastUpdated        time.Time `json:"last_updated"`
}

// NewNRNIntegration creates a new local-only NRN integration instance.
// No oracle connection is established — bounties and rewards are tracked
// locally on the graph and committed via CommitSkill to KNIRVCHAIN.
func NewNRNIntegration() *NRNIntegration {
	return &NRNIntegration{
		enabled: true,
		economicMetrics: &EconomicMetrics{
			TotalNRNRewards: big.NewInt(0),
			TotalNRNBurned:  big.NewInt(0),
			LastUpdated:     time.Now(),
		},
	}
}

// Start initializes the NRN integration.
func (ni *NRNIntegration) Start() {
	log.Println("NRN Integration started (local-only mode)")
}

// GetEconomicMetrics returns current economic metrics
func (ni *NRNIntegration) GetEconomicMetrics() *EconomicMetrics {
	ni.mutex.RLock()
	defer ni.mutex.RUnlock()

	metrics := &EconomicMetrics{
		TotalNRVsCreated:   ni.economicMetrics.TotalNRVsCreated,
		TotalSkillsInvoked: ni.economicMetrics.TotalSkillsInvoked,
		TotalNRNRewards:    new(big.Int).Set(ni.economicMetrics.TotalNRNRewards),
		TotalNRNBurned:     new(big.Int).Set(ni.economicMetrics.TotalNRNBurned),
		ActiveErrorNodes:   ni.economicMetrics.ActiveErrorNodes,
		ActiveSkillNodes:   ni.economicMetrics.ActiveSkillNodes,
		NetworkEfficiency:  ni.economicMetrics.NetworkEfficiency,
		LastUpdated:        ni.economicMetrics.LastUpdated,
	}

	return metrics
}

// IsEnabled returns whether NRN integration is enabled
func (ni *NRNIntegration) IsEnabled() bool {
	ni.mutex.RLock()
	defer ni.mutex.RUnlock()
	return ni.enabled
}

// SetEnabled enables or disables NRN integration
func (ni *NRNIntegration) SetEnabled(enabled bool) {
	ni.mutex.Lock()
	defer ni.mutex.Unlock()
	ni.enabled = enabled
	log.Printf("NRN integration enabled: %v", enabled)
}

// Bounty tracks a reward associated with an ErrorNode
type Bounty struct {
	ErrorNodeID string   `json:"error_node_id"`
	Amount      *big.Int `json:"amount"`
	Reason      string   `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// AddBounty adds a bounty to an error node. Bounties are later redeemed
// when CommitSkill transfers ownership and triggers reward distribution.
func (ni *NRNIntegration) AddBounty(errorNodeID string, amount *big.Int, reason string) *Bounty {
	ni.mutex.Lock()
	defer ni.mutex.Unlock()

	ni.economicMetrics.TotalNRNRewards.Add(ni.economicMetrics.TotalNRNRewards, amount)
	ni.economicMetrics.ActiveErrorNodes++
	ni.economicMetrics.LastUpdated = time.Now()

	bounty := &Bounty{
		ErrorNodeID: errorNodeID,
		Amount:      new(big.Int).Set(amount),
		Reason:      reason,
		Timestamp:   time.Now(),
	}

	log.Printf("Bounty added: %s NRN for error node %s (%s)", amount.String(), errorNodeID, reason)
	return bounty
}
