package drq

import (
	"KNIRVGRAPH/internal/dht"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)
const (
	QValueTopic = "q-value-updates"
)
// QValueUpdate represents a message for gossiping Q-value updates.
type QValueUpdate struct {
	NodeID    string  `json:"node_id"`
	StateKey  string  `json:"state_key"`
	ActionKey string  `json:"action_key"`
	NewQ      float64 `json:"new_q"`
}
// DRQSyncProtocol handles distributed Q-value propagation
type DRQSyncProtocol struct {
	nodeID          string
	dhtManager      dht.DHTManagerInterface
	localQTable     map[string]map[string]float64              // state → action → Q-value
	neighborQTables map[string]map[string]map[string]float64 // neighborID → state → action → Q-value
	neighborWeights map[string]float64                         // nodeID → weight
	learningRate    float64
	discountFactor  float64
	syncInterval    time.Duration
	mutex           sync.RWMutex
}
// NewDRQSyncProtocol creates a new DRQSyncProtocol.
func NewDRQSyncProtocol(nodeID string, dhtManager dht.DHTManagerInterface, learningRate, discountFactor float64, syncInterval time.Duration) *DRQSyncProtocol {
	return &DRQSyncProtocol{
		nodeID:          nodeID,
		dhtManager:      dhtManager,
		localQTable:     make(map[string]map[string]float64),
		neighborQTables: make(map[string]map[string]map[string]float64),
		neighborWeights: make(map[string]float64), // This should be populated based on network topology
		learningRate:    learningRate,
		discountFactor:  discountFactor,
		syncInterval:    syncInterval,
	}
}
// Start subscribes to the Q-value topic and starts the gossip handler.
func (d *DRQSyncProtocol) Start(ctx context.Context) error {
	ch, err := d.dhtManager.Subscribe(QValueTopic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to Q-value topic: %w", err)
	}
	go d.handleGossip(ctx, ch)
	return nil
}
// handleGossip processes incoming Q-value updates from the network.
func (d *DRQSyncProtocol) handleGossip(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}

			var update QValueUpdate
			if err := json.Unmarshal(data, &update); err != nil {
				log.Printf("Error unmarshaling Q-value update: %v", err)
				continue
			}

			if update.NodeID == d.nodeID {
				continue // Ignore messages from self
			}

			d.mutex.Lock()
			if _, ok := d.neighborQTables[update.NodeID]; !ok {
				d.neighborQTables[update.NodeID] = make(map[string]map[string]float64)
			}
			if _, ok := d.neighborQTables[update.NodeID][update.StateKey]; !ok {
				d.neighborQTables[update.NodeID][update.StateKey] = make(map[string]float64)
			}
			d.neighborQTables[update.NodeID][update.StateKey][update.ActionKey] = update.NewQ
			d.mutex.Unlock()
		}
	}
}
// SynchronizeQValues aggregates Q-values from network neighbors
func (d *DRQSyncProtocol) SynchronizeQValues(
	state ErrorClusterState,
	action DRQAction,
	reward float64,
	nextState ErrorClusterState,
) error {
	stateKey := state.ClusterID
	actionKey := action.Type.String()
	d.mutex.Lock()
	defer d.mutex.Unlock()
	// Initialize localQTable entry if not exists
	if _, ok := d.localQTable[stateKey]; !ok {
		d.localQTable[stateKey] = make(map[string]float64)
	}
	if _, ok := d.localQTable[stateKey][actionKey]; !ok {
		d.localQTable[stateKey][actionKey] = 0.0 // Initialize with a default value
	}
	// Fetch neighbor Q-values from our cache
	neighborQValues := d.fetchNeighborQValues(nextState)
	// Weighted aggregation
	aggregatedMaxQ := 0.0
	totalWeight := 0.0
	for nodeID, qMap := range neighborQValues {
		weight, ok := d.neighborWeights[nodeID]
		if !ok {
			weight = 1.0 // Default weight if not specified
		}
		maxQ := d.getMaxQ(qMap, nextState)
		aggregatedMaxQ += weight * maxQ
		totalWeight += weight
	}
	if totalWeight > 0 {
		aggregatedMaxQ /= totalWeight
	}
	// Local Q-update with distributed term
	currentQ := d.localQTable[stateKey][actionKey]
	targetQ := reward + d.discountFactor*aggregatedMaxQ
	newQ := (1-d.learningRate)*currentQ + d.learningRate*targetQ
	d.localQTable[stateKey][actionKey] = newQ
	// Gossip updated Q-value to neighbors
	return d.gossipQUpdate(stateKey, actionKey, newQ)
}
// fetchNeighborQValues fetches Q-values from the local cache of neighbor Q-tables.
func (d *DRQSyncProtocol) fetchNeighborQValues(nextState ErrorClusterState) map[string]map[string]float64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	stateKey := nextState.ClusterID
	result := make(map[string]map[string]float64)
	for neighborID, qTables := range d.neighborQTables {
		if qMap, ok := qTables[stateKey]; ok {
			result[neighborID] = qMap
		}
	}
	return result
}
// getMaxQ gets the maximum Q-value for a given state from a Q-table map.
func (d *DRQSyncProtocol) getMaxQ(qMap map[string]float64, nextState ErrorClusterState) float64 {
	// In a more sophisticated implementation, this could depend on the nextState.
	// For now, we just find the max Q-value in the map.
	_ = nextState // Mark as used to avoid compiler warning
	maxQ := -1.0  // Start with a value lower than any possible Q-value
	for _, q := range qMap {
		if q > maxQ {
			maxQ = q
		}
	}
	// If no Q-values were found, return 0.0
	if maxQ == -1.0 {
		return 0.0
	}
	return maxQ
}
// gossipQUpdate gossips the updated Q-value to the network.
func (d *DRQSyncProtocol) gossipQUpdate(stateKey, actionKey string, newQ float64) error {
	update := QValueUpdate{
		NodeID:    d.nodeID,
		StateKey:  stateKey,
		ActionKey: actionKey,
		NewQ:      newQ,
	}
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal Q-value update: %w", err)
	}
	return d.dhtManager.Publish(QValueTopic, data)
}
// GetQValue retrieves a Q-value for a given state and action.
func (d *DRQSyncProtocol) GetQValue(state ErrorClusterState, action DRQAction) float64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	stateKey := state.ClusterID
	actionKey := action.Type.String()
	if qMap, ok := d.localQTable[stateKey]; ok {
		if qValue, found := qMap[actionKey]; found {
			return qValue
		}
	}
	return 0.0 // Default Q-value if not found
}
