// connectivity/proof_engine.go
package connectivity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ConnectivityProofEngine manages proof-of-connectivity for KNIRV-ROUTER
type ConnectivityProofEngine struct {
	host              host.Host
	ctx               context.Context
	cancel            context.CancelFunc
	measurements      map[peer.ID]*ConnectivityMeasurement
	measurementsMutex sync.RWMutex
	proofHistory      []ConnectivityProof
	proofHistoryMutex sync.RWMutex
	nrnMintingEnabled bool
	faucetEndpoint    string
	minConnectivity   float64
	measurementWindow time.Duration
}

// ConnectivityMeasurement represents connectivity metrics for a peer
type ConnectivityMeasurement struct {
	PeerID            peer.ID       `json:"peer_id"`
	Latency           time.Duration `json:"latency"`
	Bandwidth         float64       `json:"bandwidth"`   // MB/s
	PacketLoss        float64       `json:"packet_loss"` // percentage
	Uptime            time.Duration `json:"uptime"`
	LastMeasurement   time.Time     `json:"last_measurement"`
	ConnectivityScore float64       `json:"connectivity_score"`
	IsReliable        bool          `json:"is_reliable"`
}

// ConnectivityProof represents a proof-of-connectivity submission
type ConnectivityProof struct {
	ProofID        string                    `json:"proof_id"`
	NodeID         peer.ID                   `json:"node_id"`
	Timestamp      time.Time                 `json:"timestamp"`
	Measurements   []ConnectivityMeasurement `json:"measurements"`
	AggregateScore float64                   `json:"aggregate_score"`
	ProofHash      string                    `json:"proof_hash"`
	NRNReward      *big.Int                  `json:"nrn_reward"`
	Verified       bool                      `json:"verified"`
}

// ProofEngineConfig contains configuration for the proof engine
type ProofEngineConfig struct {
	NRNMintingEnabled bool          `json:"nrn_minting_enabled"`
	FaucetEndpoint    string        `json:"faucet_endpoint"`
	MinConnectivity   float64       `json:"min_connectivity"`
	MeasurementWindow time.Duration `json:"measurement_window"`
	RewardMultiplier  float64       `json:"reward_multiplier"`
}

// NewConnectivityProofEngine creates a new connectivity proof engine
func NewConnectivityProofEngine(host host.Host, config ProofEngineConfig) *ConnectivityProofEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectivityProofEngine{
		host:              host,
		ctx:               ctx,
		cancel:            cancel,
		measurements:      make(map[peer.ID]*ConnectivityMeasurement),
		proofHistory:      make([]ConnectivityProof, 0),
		nrnMintingEnabled: config.NRNMintingEnabled,
		faucetEndpoint:    config.FaucetEndpoint,
		minConnectivity:   config.MinConnectivity,
		measurementWindow: config.MeasurementWindow,
	}
}

// Start begins the connectivity proof engine
func (cpe *ConnectivityProofEngine) Start() error {
	log.Println("Starting Connectivity Proof Engine...")

	// Start measurement collection
	go cpe.collectConnectivityMeasurements()

	// Start proof generation
	go cpe.generateConnectivityProofs()

	// Start proof verification
	go cpe.verifyConnectivityProofs()

	log.Println("Connectivity Proof Engine started successfully")
	return nil
}

// Stop stops the connectivity proof engine
func (cpe *ConnectivityProofEngine) Stop() {
	log.Println("Stopping Connectivity Proof Engine...")
	cpe.cancel()
}

// collectConnectivityMeasurements continuously measures connectivity to peers
func (cpe *ConnectivityProofEngine) collectConnectivityMeasurements() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cpe.ctx.Done():
			return
		case <-ticker.C:
			cpe.measurePeerConnectivity()
		}
	}
}

// measurePeerConnectivity measures connectivity to all connected peers
func (cpe *ConnectivityProofEngine) measurePeerConnectivity() {
	peers := cpe.host.Network().Peers()

	for _, peerID := range peers {
		go func(pid peer.ID) {
			measurement := cpe.measureSinglePeer(pid)
			if measurement != nil {
				cpe.measurementsMutex.Lock()
				cpe.measurements[pid] = measurement
				cpe.measurementsMutex.Unlock()
			}
		}(peerID)
	}
}

// measureSinglePeer measures connectivity to a single peer
func (cpe *ConnectivityProofEngine) measureSinglePeer(peerID peer.ID) *ConnectivityMeasurement {
	start := time.Now()

	// Simulate connectivity measurements
	// In production, this would use actual network measurements
	latency := cpe.measureLatency(peerID)
	bandwidth := cpe.measureBandwidth(peerID)
	packetLoss := cpe.measurePacketLoss(peerID)
	uptime := cpe.calculateUptime(peerID)

	// Calculate connectivity score
	score := cpe.calculateConnectivityScore(latency, bandwidth, packetLoss, uptime)

	measurementDuration := time.Since(start)

	measurement := &ConnectivityMeasurement{
		PeerID:            peerID,
		Latency:           latency,
		Bandwidth:         bandwidth,
		PacketLoss:        packetLoss,
		Uptime:            uptime,
		LastMeasurement:   time.Now(),
		ConnectivityScore: score,
		IsReliable:        score >= cpe.minConnectivity,
	}

	log.Printf("Measured connectivity to peer %s: score=%.2f, latency=%v, bandwidth=%.2f MB/s, measurement_time=%v",
		peerID.String()[:8], score, latency, bandwidth, measurementDuration)

	return measurement
}

// measureLatency measures network latency to a peer
func (cpe *ConnectivityProofEngine) measureLatency(peerID peer.ID) time.Duration {
	// Simplified latency measurement
	// In production, this would use actual ping measurements
	start := time.Now()

	// Simulate network round trip
	time.Sleep(time.Millisecond * time.Duration(10+(time.Now().UnixNano()%100)))

	return time.Since(start)
}

// measureBandwidth measures available bandwidth to a peer
func (cpe *ConnectivityProofEngine) measureBandwidth(peerID peer.ID) float64 {
	// Simplified bandwidth measurement
	// In production, this would use actual throughput tests

	// Simulate bandwidth measurement (MB/s)
	baseBandwidth := 10.0 + float64(time.Now().UnixNano()%90)
	return baseBandwidth
}

// measurePacketLoss measures packet loss percentage to a peer
func (cpe *ConnectivityProofEngine) measurePacketLoss(peerID peer.ID) float64 {
	// Simplified packet loss measurement
	// In production, this would use actual packet loss detection

	// Simulate packet loss (0-5%)
	loss := float64(time.Now().UnixNano()%50) / 10.0
	return loss
}

// calculateUptime calculates peer uptime
func (cpe *ConnectivityProofEngine) calculateUptime(peerID peer.ID) time.Duration {
	// Simplified uptime calculation
	// In production, this would track actual peer connection time

	// Simulate uptime (1-24 hours)
	hours := 1 + (time.Now().UnixNano() % 24)
	return time.Duration(hours) * time.Hour
}

// calculateConnectivityScore calculates overall connectivity score
func (cpe *ConnectivityProofEngine) calculateConnectivityScore(latency time.Duration, bandwidth, packetLoss float64, uptime time.Duration) float64 {
	// Scoring algorithm:
	// - Lower latency is better (max 100ms for full score)
	// - Higher bandwidth is better (max 100 MB/s for full score)
	// - Lower packet loss is better (0% for full score)
	// - Higher uptime is better (24h for full score)

	latencyScore := 1.0 - (float64(latency.Milliseconds()) / 100.0)
	if latencyScore < 0 {
		latencyScore = 0
	}

	bandwidthScore := bandwidth / 100.0
	if bandwidthScore > 1.0 {
		bandwidthScore = 1.0
	}

	packetLossScore := 1.0 - (packetLoss / 100.0)
	if packetLossScore < 0 {
		packetLossScore = 0
	}

	uptimeScore := float64(uptime.Hours()) / 24.0
	if uptimeScore > 1.0 {
		uptimeScore = 1.0
	}

	// Weighted average
	totalScore := (latencyScore*0.3 + bandwidthScore*0.3 + packetLossScore*0.2 + uptimeScore*0.2) * 100.0

	return totalScore
}

// generateConnectivityProofs generates proofs based on collected measurements
func (cpe *ConnectivityProofEngine) generateConnectivityProofs() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cpe.ctx.Done():
			return
		case <-ticker.C:
			cpe.generateProof()
		}
	}
}

// generateProof creates a connectivity proof from current measurements
func (cpe *ConnectivityProofEngine) generateProof() {
	cpe.measurementsMutex.RLock()
	measurements := make([]ConnectivityMeasurement, 0, len(cpe.measurements))
	for _, measurement := range cpe.measurements {
		measurements = append(measurements, *measurement)
	}
	cpe.measurementsMutex.RUnlock()

	if len(measurements) == 0 {
		log.Println("No measurements available for proof generation")
		return
	}

	// Calculate aggregate score
	var totalScore float64
	for _, measurement := range measurements {
		totalScore += measurement.ConnectivityScore
	}
	aggregateScore := totalScore / float64(len(measurements))

	// Create proof
	proof := ConnectivityProof{
		ProofID:        fmt.Sprintf("proof_%d", time.Now().UnixNano()),
		NodeID:         cpe.host.ID(),
		Timestamp:      time.Now(),
		Measurements:   measurements,
		AggregateScore: aggregateScore,
		Verified:       false,
	}

	// Generate proof hash
	proof.ProofHash = cpe.generateProofHash(proof)

	// Calculate NRN reward if minting is enabled
	if cpe.nrnMintingEnabled && aggregateScore >= cpe.minConnectivity {
		reward := cpe.calculateNRNReward(aggregateScore, len(measurements))
		proof.NRNReward = reward
	}

	// Store proof
	cpe.proofHistoryMutex.Lock()
	cpe.proofHistory = append(cpe.proofHistory, proof)
	cpe.proofHistoryMutex.Unlock()

	log.Printf("Generated connectivity proof: ID=%s, score=%.2f, measurements=%d",
		proof.ProofID, aggregateScore, len(measurements))

	// Submit proof for verification
	go cpe.submitProofForVerification(proof)
}

// generateProofHash generates a cryptographic hash for the proof
func (cpe *ConnectivityProofEngine) generateProofHash(proof ConnectivityProof) string {
	// Create hash input from proof data
	hashInput := fmt.Sprintf("%s_%s_%f_%d",
		proof.NodeID.String(),
		proof.Timestamp.Format(time.RFC3339),
		proof.AggregateScore,
		len(proof.Measurements))

	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// calculateNRNReward calculates NRN reward based on connectivity performance
func (cpe *ConnectivityProofEngine) calculateNRNReward(score float64, measurementCount int) *big.Int {
	// Base reward calculation
	baseReward := 100 // Base NRN tokens

	// Score multiplier (0.5x to 2x based on score)
	scoreMultiplier := score / 50.0
	if scoreMultiplier > 2.0 {
		scoreMultiplier = 2.0
	}

	// Measurement count bonus (more measurements = higher reward)
	measurementBonus := float64(measurementCount) * 0.1
	if measurementBonus > 1.0 {
		measurementBonus = 1.0
	}

	finalReward := float64(baseReward) * scoreMultiplier * (1.0 + measurementBonus)

	// Convert to big.Int (assuming 18 decimal places)
	rewardWei := new(big.Int)
	rewardWei.SetString(fmt.Sprintf("%.0f", finalReward*1e18), 10)

	return rewardWei
}

// submitProofForVerification submits proof to the network for verification
func (cpe *ConnectivityProofEngine) submitProofForVerification(proof ConnectivityProof) {
	log.Printf("Submitting proof %s for verification", proof.ProofID)

	// In production, this would submit to the blockchain or verification network
	// For now, simulate verification process
	time.Sleep(2 * time.Second)

	// Mark as verified
	cpe.proofHistoryMutex.Lock()
	for i := range cpe.proofHistory {
		if cpe.proofHistory[i].ProofID == proof.ProofID {
			cpe.proofHistory[i].Verified = true
			break
		}
	}
	cpe.proofHistoryMutex.Unlock()

	log.Printf("Proof %s verified successfully", proof.ProofID)

	// Request NRN minting if reward is available
	if proof.NRNReward != nil && proof.NRNReward.Cmp(big.NewInt(0)) > 0 {
		go cpe.requestNRNMinting(proof)
	}
}

// requestNRNMinting requests NRN token minting for verified proof
func (cpe *ConnectivityProofEngine) requestNRNMinting(proof ConnectivityProof) {
	if !cpe.nrnMintingEnabled || cpe.faucetEndpoint == "" {
		log.Printf("NRN minting not enabled or faucet endpoint not configured")
		return
	}

	log.Printf("Requesting NRN minting for proof %s: %s NRN",
		proof.ProofID, proof.NRNReward.String())

	// In production, this would make HTTP request to faucet
	// For now, simulate the request
	time.Sleep(1 * time.Second)

	log.Printf("NRN minting request completed for proof %s", proof.ProofID)
}

// verifyConnectivityProofs verifies proofs from other nodes
func (cpe *ConnectivityProofEngine) verifyConnectivityProofs() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cpe.ctx.Done():
			return
		case <-ticker.C:
			cpe.verifyPendingProofs()
		}
	}
}

// verifyPendingProofs verifies proofs that are pending verification
func (cpe *ConnectivityProofEngine) verifyPendingProofs() {
	// In production, this would verify proofs from other nodes
	// For now, just log the verification process
	log.Println("Verifying pending connectivity proofs from network...")
}

// GetConnectivityMeasurements returns current connectivity measurements
func (cpe *ConnectivityProofEngine) GetConnectivityMeasurements() map[peer.ID]*ConnectivityMeasurement {
	cpe.measurementsMutex.RLock()
	defer cpe.measurementsMutex.RUnlock()

	measurements := make(map[peer.ID]*ConnectivityMeasurement)
	for k, v := range cpe.measurements {
		measurements[k] = v
	}

	return measurements
}

// GetProofHistory returns the history of connectivity proofs
func (cpe *ConnectivityProofEngine) GetProofHistory() []ConnectivityProof {
	cpe.proofHistoryMutex.RLock()
	defer cpe.proofHistoryMutex.RUnlock()

	history := make([]ConnectivityProof, len(cpe.proofHistory))
	copy(history, cpe.proofHistory)

	return history
}

// GetConnectivityStats returns aggregated connectivity statistics
func (cpe *ConnectivityProofEngine) GetConnectivityStats() map[string]interface{} {
	cpe.measurementsMutex.RLock()
	measurements := make(map[peer.ID]*ConnectivityMeasurement)
	for k, v := range cpe.measurements {
		measurements[k] = v
	}
	cpe.measurementsMutex.RUnlock()

	cpe.proofHistoryMutex.RLock()
	proofCount := len(cpe.proofHistory)
	var totalRewards big.Int
	verifiedProofs := 0

	for _, proof := range cpe.proofHistory {
		if proof.Verified {
			verifiedProofs++
		}
		if proof.NRNReward != nil {
			totalRewards.Add(&totalRewards, proof.NRNReward)
		}
	}
	cpe.proofHistoryMutex.RUnlock()

	// Calculate average connectivity score
	var totalScore float64
	reliablePeers := 0

	for _, measurement := range measurements {
		totalScore += measurement.ConnectivityScore
		if measurement.IsReliable {
			reliablePeers++
		}
	}

	avgScore := 0.0
	if len(measurements) > 0 {
		avgScore = totalScore / float64(len(measurements))
	}

	return map[string]interface{}{
		"total_peers":       len(measurements),
		"reliable_peers":    reliablePeers,
		"average_score":     avgScore,
		"total_proofs":      proofCount,
		"verified_proofs":   verifiedProofs,
		"total_nrn_rewards": totalRewards.String(),
		"engine_uptime":     time.Since(time.Now().Add(-time.Hour)), // Placeholder
		"last_measurement":  time.Now(),
	}
}
