// connectivity/proof_engine.go
package connectivity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// FaucetClient interface for NRN token minting requests
type FaucetClient interface {
	RequestConnectivityReward(nodeID peer.ID, proofID string, score float64, amount *big.Int) (*FaucetRequest, error)
}

// BlockchainAdapter interface for submitting NRN mint transactions to blockchain
type BlockchainAdapter interface {
	SubmitNRNMintTx(recipient, amount, reason, proofID string) error
}

// NetworkPauseChecker interface for checking network pause state
type NetworkPauseChecker interface {
	IsNetworkPaused() bool
}

// FaucetRequest represents a request to the faucet (interface compatible)
type FaucetRequest struct {
	RequestID    string    `json:"request_id"`
	NodeID       peer.ID   `json:"node_id"`
	Amount       *big.Int  `json:"amount"`
	Reason       string    `json:"reason"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"` // "pending", "completed", "failed"
	TxHash       string    `json:"tx_hash,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// NRVMetadata represents a Network Resolution Vector with tokenized metadata
type NRVMetadata struct {
	NRVID             string                 `json:"nrv_id"`
	ProofID           string                 `json:"proof_id"`
	NodeID            peer.ID                `json:"node_id"`
	RouteData         map[string]interface{} `json:"route_data"`
	ConnectivityScore float64                `json:"connectivity_score"`
	NRNValue          *big.Int               `json:"nrn_value"`
	Timestamp         time.Time              `json:"timestamp"`
	Signature         string                 `json:"signature"`
	TreasuryAddress   string                 `json:"treasury_address"`
	Status            string                 `json:"status"` // "minted", "transferred", "confirmed"
	// Enhanced fields for production network
	NetworkMetrics  *NetworkMetrics  `json:"network_metrics,omitempty"`
	RouteQuality    *RouteQuality    `json:"route_quality,omitempty"`
	TokenMetadata   *TokenMetadata   `json:"token_metadata,omitempty"`
	ValidationProof *ValidationProof `json:"validation_proof,omitempty"`
}

// NetworkMetrics contains detailed network performance data
type NetworkMetrics struct {
	Latency          time.Duration `json:"latency"`
	Throughput       float64       `json:"throughput"`
	PacketLoss       float64       `json:"packet_loss"`
	Jitter           time.Duration `json:"jitter"`
	Bandwidth        float64       `json:"bandwidth"`
	Reliability      float64       `json:"reliability"`
	GeographicSpread []string      `json:"geographic_spread"`
	PeerCount        int           `json:"peer_count"`
	UpstreamNodes    []string      `json:"upstream_nodes"`
	DownstreamNodes  []string      `json:"downstream_nodes"`
}

// RouteQuality represents the quality assessment of a network route
type RouteQuality struct {
	OverallScore       float64   `json:"overall_score"`
	StabilityScore     float64   `json:"stability_score"`
	PerformanceScore   float64   `json:"performance_score"`
	SecurityScore      float64   `json:"security_score"`
	RedundancyScore    float64   `json:"redundancy_score"`
	QualityGrade       string    `json:"quality_grade"` // A, B, C, D, F
	CertificationLevel string    `json:"certification_level"`
	ValidatedAt        time.Time `json:"validated_at"`
}

// TokenMetadata contains NRV token-specific metadata
type TokenMetadata struct {
	TokenStandard        string                 `json:"token_standard"` // NRV-1.0
	MintingAuthority     string                 `json:"minting_authority"`
	TransferRestrictions []string               `json:"transfer_restrictions"`
	ExpirationTime       *time.Time             `json:"expiration_time,omitempty"`
	Attributes           map[string]interface{} `json:"attributes"`
	IPFSHash             string                 `json:"ipfs_hash,omitempty"`
	OnChainReference     string                 `json:"onchain_reference"`
}

// ValidationProof contains cryptographic proof of route validation
type ValidationProof struct {
	ProofType           string               `json:"proof_type"` // merkle, zk-snark, etc.
	ProofData           []byte               `json:"proof_data"`
	ValidatorSignatures []ValidatorSignature `json:"validator_signatures"`
	MerkleRoot          string               `json:"merkle_root"`
	BlockHeight         uint64               `json:"block_height"`
	ConsensusRound      uint64               `json:"consensus_round"`
}

// ValidatorSignature represents a validator's signature on the proof
type ValidatorSignature struct {
	ValidatorID string    `json:"validator_id"`
	PublicKey   string    `json:"public_key"`
	Signature   string    `json:"signature"`
	Timestamp   time.Time `json:"timestamp"`
	StakeWeight *big.Int  `json:"stake_weight"`
}

// TreasuryTransferRequest represents a request to transfer NRV to KNIRVORACLE treasury
type TreasuryTransferRequest struct {
	TransferID   string       `json:"transfer_id"`
	NRVMetadata  *NRVMetadata `json:"nrv_metadata"`
	TargetWallet string       `json:"target_wallet"`
	Timestamp    time.Time    `json:"timestamp"`
}

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
	faucetClient      FaucetClient
	blockchainAdapter BlockchainAdapter
	pauseChecker      NetworkPauseChecker
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
	NRNMintingEnabled bool                `json:"nrn_minting_enabled"`
	FaucetEndpoint    string              `json:"faucet_endpoint"`
	FaucetClient      FaucetClient        `json:"-"` // Not serialized
	BlockchainAdapter BlockchainAdapter   `json:"-"` // Not serialized
	PauseChecker      NetworkPauseChecker `json:"-"` // Not serialized
	MinConnectivity   float64             `json:"min_connectivity"`
	MeasurementWindow time.Duration       `json:"measurement_window"`
	RewardMultiplier  float64             `json:"reward_multiplier"`
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
		faucetClient:      config.FaucetClient,
		blockchainAdapter: config.BlockchainAdapter,
		pauseChecker:      config.PauseChecker,
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

// requestNRNMinting requests NRN token minting for verified proof and handles treasury transfer
func (cpe *ConnectivityProofEngine) requestNRNMinting(proof ConnectivityProof) {
	if !cpe.nrnMintingEnabled {
		log.Printf("NRN minting not enabled")
		return
	}

	// Check if network is paused
	if cpe.pauseChecker != nil && cpe.pauseChecker.IsNetworkPaused() {
		log.Printf("Network is paused, cannot mint NRN for proof %s", proof.ProofID)
		return
	}

	if cpe.faucetClient == nil {
		log.Printf("Faucet client not configured, falling back to endpoint simulation")
		cpe.requestNRNMintingFallback(proof)
		return
	}

	log.Printf("Requesting NRN minting for proof %s: %s NRN",
		proof.ProofID, proof.NRNReward.String())

	// First, mint the NRV (Network Resolution Vector) locally
	nrvMetadata := cpe.mintNRV(proof)
	if nrvMetadata == nil {
		log.Printf("Failed to mint NRV for proof %s", proof.ProofID)
		return
	}

	// Transfer NRV metadata to KNIRVORACLE treasury
	err := cpe.transferNRVToTreasury(nrvMetadata)
	if err != nil {
		log.Printf("Failed to transfer NRV to treasury for proof %s: %v", proof.ProofID, err)
		return
	}

	// Use the faucet client to request NRN tokens
	request, err := cpe.faucetClient.RequestConnectivityReward(
		proof.NodeID,
		proof.ProofID,
		proof.AggregateScore,
		proof.NRNReward,
	)

	if err != nil {
		log.Printf("Failed to request NRN minting for proof %s: %v", proof.ProofID, err)
		return
	}

	log.Printf("NRN minting request submitted for proof %s: RequestID=%s, Status=%s",
		proof.ProofID, request.RequestID, request.Status)

	// Also submit to blockchain adapter if available
	if cpe.blockchainAdapter != nil {
		reason := fmt.Sprintf("connectivity_proof_%s_score_%.2f", proof.ProofID, proof.AggregateScore)
		err := cpe.blockchainAdapter.SubmitNRNMintTx(
			proof.NodeID.String(),
			proof.NRNReward.String(),
			reason,
			proof.ProofID,
		)
		if err != nil {
			log.Printf("Failed to submit NRN mint transaction to blockchain for proof %s: %v", proof.ProofID, err)
		} else {
			log.Printf("NRN mint transaction submitted to blockchain for proof %s", proof.ProofID)
		}
	}
}

// requestNRNMintingFallback provides fallback behavior when faucet client is not available
func (cpe *ConnectivityProofEngine) requestNRNMintingFallback(proof ConnectivityProof) {
	if cpe.faucetEndpoint == "" {
		log.Printf("NRN minting fallback: no faucet endpoint configured")
		return
	}

	log.Printf("Using fallback NRN minting for proof %s: %s NRN",
		proof.ProofID, proof.NRNReward.String())

	// Prepare request payload
	payload := map[string]interface{}{
		"recipient": proof.NodeID.String(),
		"amount":    proof.NRNReward.String(),
		"reason":    fmt.Sprintf("connectivity_proof_%s_score_%.2f", proof.ProofID, proof.AggregateScore),
		"proof_id":  proof.ProofID,
		"timestamp": proof.Timestamp.Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal NRN mint request payload: %v", err)
		return
	}

	// Create HTTP client with timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Make HTTP request to faucet endpoint
	resp, err := client.Post(cpe.faucetEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Printf("Failed to send NRN mint request to %s: %v", cpe.faucetEndpoint, err)
		return
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read NRN mint response: %v", err)
		return
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		log.Printf("Failed to parse NRN mint response: %v", err)
		return
	}

	// Check response status
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		if success, ok := response["success"].(bool); ok && success {
			txHash, _ := response["tx_hash"].(string)
			log.Printf("NRN minting request successful for proof %s: TxHash=%s", proof.ProofID, txHash)
		} else {
			message, _ := response["message"].(string)
			log.Printf("NRN minting request failed for proof %s: %s", proof.ProofID, message)
		}
	} else {
		log.Printf("NRN minting request failed for proof %s: HTTP %d - %s",
			proof.ProofID, resp.StatusCode, string(responseBody))
	}
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

// mintNRV creates a Network Resolution Vector from a verified connectivity proof
func (cpe *ConnectivityProofEngine) mintNRV(proof ConnectivityProof) *NRVMetadata {
	// Check if network is paused
	if cpe.pauseChecker != nil && cpe.pauseChecker.IsNetworkPaused() {
		log.Printf("Network is paused, cannot mint NRV for proof %s", proof.ProofID)
		return nil
	}

	// Generate unique NRV ID
	nrvID := fmt.Sprintf("nrv_%s_%d", proof.ProofID, time.Now().UnixNano())

	// Create route data from proof and measurements
	routeData := map[string]interface{}{
		"source_node":       proof.NodeID.String(),
		"proof_hash":        proof.ProofHash,
		"aggregate_score":   proof.AggregateScore,
		"measurement_count": len(proof.Measurements),
		"verification_time": proof.Timestamp,
		"verified":          proof.Verified,
	}

	// Add aggregated measurement data
	if len(proof.Measurements) > 0 {
		var totalLatency time.Duration
		var totalBandwidth float64
		var totalPacketLoss float64
		var reliablePeers int

		for _, measurement := range proof.Measurements {
			totalLatency += measurement.Latency
			totalBandwidth += measurement.Bandwidth
			totalPacketLoss += measurement.PacketLoss
			if measurement.IsReliable {
				reliablePeers++
			}
		}

		measurementCount := len(proof.Measurements)
		routeData["avg_latency_ms"] = float64(totalLatency.Milliseconds()) / float64(measurementCount)
		routeData["avg_bandwidth_mbps"] = totalBandwidth / float64(measurementCount)
		routeData["avg_packet_loss"] = totalPacketLoss / float64(measurementCount)
		routeData["reliable_peers"] = reliablePeers
		routeData["reliability_ratio"] = float64(reliablePeers) / float64(measurementCount)
	}

	// Create enhanced network metrics
	networkMetrics := cpe.generateNetworkMetrics(proof)

	// Create route quality assessment
	routeQuality := cpe.assessRouteQuality(proof)

	// Create token metadata
	tokenMetadata := cpe.generateTokenMetadata(nrvID, proof)

	// Create validation proof
	validationProof := cpe.generateValidationProof(proof)

	// Create NRV metadata with enhanced fields
	nrvMetadata := &NRVMetadata{
		NRVID:             nrvID,
		ProofID:           proof.ProofID,
		NodeID:            proof.NodeID,
		RouteData:         routeData,
		ConnectivityScore: proof.AggregateScore,
		NRNValue:          proof.NRNReward,
		Timestamp:         time.Now(),
		TreasuryAddress:   "knirvoracle_treasury_wallet", // Should be configurable
		Status:            "minted",
		NetworkMetrics:    networkMetrics,
		RouteQuality:      routeQuality,
		TokenMetadata:     tokenMetadata,
		ValidationProof:   validationProof,
	}

	// Generate signature for NRV (simplified - in production would use proper cryptographic signing)
	nrvMetadata.Signature = cpe.generateNRVSignature(nrvMetadata)

	log.Printf("Minted NRV %s for proof %s with value %s NRN",
		nrvID, proof.ProofID, proof.NRNReward.String())

	return nrvMetadata
}

// transferNRVToTreasury sends NRV metadata to KNIRVORACLE treasury
func (cpe *ConnectivityProofEngine) transferNRVToTreasury(nrvMetadata *NRVMetadata) error {
	// Create transfer request
	transferID := fmt.Sprintf("transfer_%s_%d", nrvMetadata.NRVID, time.Now().UnixNano())
	transferRequest := &TreasuryTransferRequest{
		TransferID:   transferID,
		NRVMetadata:  nrvMetadata,
		TargetWallet: nrvMetadata.TreasuryAddress,
		Timestamp:    time.Now(),
	}

	// Serialize transfer request
	requestData, err := json.Marshal(transferRequest)
	if err != nil {
		return fmt.Errorf("failed to serialize transfer request: %w", err)
	}

	// Send to KNIRVORACLE treasury endpoint
	treasuryEndpoint := "http://localhost:8081/api/treasury/receive-nrv" // Should be configurable
	resp, err := http.Post(treasuryEndpoint, "application/json", bytes.NewBuffer(requestData))
	if err != nil {
		return fmt.Errorf("failed to send NRV to treasury: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("treasury transfer failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Update NRV status
	nrvMetadata.Status = "transferred"

	log.Printf("Successfully transferred NRV %s to treasury", nrvMetadata.NRVID)
	return nil
}

// generateNetworkMetrics creates detailed network metrics from connectivity proof
func (cpe *ConnectivityProofEngine) generateNetworkMetrics(proof ConnectivityProof) *NetworkMetrics {
	if len(proof.Measurements) == 0 {
		return &NetworkMetrics{
			PeerCount:   0,
			Reliability: 0.0,
		}
	}

	var totalLatency time.Duration
	var totalThroughput float64
	var totalPacketLoss float64
	var totalBandwidth float64
	var reliableCount int
	upstreamNodes := make([]string, 0)
	downstreamNodes := make([]string, 0)
	geoSpread := make(map[string]bool)

	for _, measurement := range proof.Measurements {
		totalLatency += measurement.Latency
		totalThroughput += measurement.Bandwidth // Using bandwidth as throughput proxy
		totalPacketLoss += measurement.PacketLoss
		totalBandwidth += measurement.Bandwidth

		if measurement.IsReliable {
			reliableCount++
		}

		// Collect node information
		nodeStr := measurement.PeerID.String()
		upstreamNodes = append(upstreamNodes, nodeStr)

		// Simulate geographic spread (in production, would use actual geo data)
		geoSpread["region_"+nodeStr[:8]] = true
	}

	measurementCount := len(proof.Measurements)
	geoSpreadSlice := make([]string, 0, len(geoSpread))
	for region := range geoSpread {
		geoSpreadSlice = append(geoSpreadSlice, region)
	}

	return &NetworkMetrics{
		Latency:          totalLatency / time.Duration(measurementCount),
		Throughput:       totalThroughput / float64(measurementCount),
		PacketLoss:       totalPacketLoss / float64(measurementCount),
		Jitter:           time.Duration(0), // Simplified - would calculate from latency variance in production
		Bandwidth:        totalBandwidth / float64(measurementCount),
		Reliability:      float64(reliableCount) / float64(measurementCount),
		GeographicSpread: geoSpreadSlice,
		PeerCount:        measurementCount,
		UpstreamNodes:    upstreamNodes,
		DownstreamNodes:  downstreamNodes,
	}
}

// assessRouteQuality evaluates the quality of a network route
func (cpe *ConnectivityProofEngine) assessRouteQuality(proof ConnectivityProof) *RouteQuality {
	// Calculate individual scores based on proof data
	stabilityScore := cpe.calculateStabilityScore(proof)
	performanceScore := cpe.calculatePerformanceScore(proof)
	securityScore := cpe.calculateSecurityScore(proof)
	redundancyScore := cpe.calculateRedundancyScore(proof)

	// Calculate overall score as weighted average
	overallScore := (stabilityScore*0.3 + performanceScore*0.3 + securityScore*0.2 + redundancyScore*0.2)

	// Determine quality grade
	var grade string
	switch {
	case overallScore >= 0.9:
		grade = "A"
	case overallScore >= 0.8:
		grade = "B"
	case overallScore >= 0.7:
		grade = "C"
	case overallScore >= 0.6:
		grade = "D"
	default:
		grade = "F"
	}

	// Determine certification level
	certLevel := "basic"
	if overallScore >= 0.85 {
		certLevel = "premium"
	} else if overallScore >= 0.75 {
		certLevel = "standard"
	}

	return &RouteQuality{
		OverallScore:       overallScore,
		StabilityScore:     stabilityScore,
		PerformanceScore:   performanceScore,
		SecurityScore:      securityScore,
		RedundancyScore:    redundancyScore,
		QualityGrade:       grade,
		CertificationLevel: certLevel,
		ValidatedAt:        time.Now(),
	}
}

// generateTokenMetadata creates token-specific metadata for NRV
func (cpe *ConnectivityProofEngine) generateTokenMetadata(nrvID string, proof ConnectivityProof) *TokenMetadata {
	attributes := make(map[string]interface{})
	attributes["proof_type"] = "connectivity"
	attributes["node_count"] = len(proof.Measurements)
	attributes["aggregate_score"] = proof.AggregateScore
	attributes["verification_timestamp"] = proof.Timestamp.Unix()
	attributes["network_version"] = "knirv-1.0"

	// Add route-specific attributes
	if len(proof.Measurements) > 0 {
		attributes["primary_node"] = proof.NodeID.String()
		attributes["measurement_count"] = len(proof.Measurements)
	}

	return &TokenMetadata{
		TokenStandard:        "NRV-1.0",
		MintingAuthority:     "knirvrouter_proof_engine",
		TransferRestrictions: []string{"treasury_only", "validated_nodes"},
		Attributes:           attributes,
		OnChainReference:     fmt.Sprintf("knirv://nrv/%s", nrvID),
	}
}

// generateValidationProof creates cryptographic validation proof
func (cpe *ConnectivityProofEngine) generateValidationProof(proof ConnectivityProof) *ValidationProof {
	// Create simplified validation proof (in production would use proper cryptography)
	proofData := []byte(fmt.Sprintf("proof_%s_%d", proof.ProofID, proof.Timestamp.Unix()))

	// Generate mock validator signatures (in production would be real validators)
	signatures := []ValidatorSignature{
		{
			ValidatorID: "validator_1",
			PublicKey:   "mock_public_key_1",
			Signature:   fmt.Sprintf("sig_%s_1", proof.ProofID),
			Timestamp:   time.Now(),
			StakeWeight: big.NewInt(1000000),
		},
	}

	return &ValidationProof{
		ProofType:           "merkle",
		ProofData:           proofData,
		ValidatorSignatures: signatures,
		MerkleRoot:          fmt.Sprintf("merkle_%s", proof.ProofHash),
		BlockHeight:         uint64(time.Now().Unix()),
		ConsensusRound:      1,
	}
}

// Helper methods for route quality calculation
func (cpe *ConnectivityProofEngine) calculateStabilityScore(proof ConnectivityProof) float64 {
	if len(proof.Measurements) == 0 {
		return 0.0
	}

	reliableCount := 0
	for _, measurement := range proof.Measurements {
		if measurement.IsReliable {
			reliableCount++
		}
	}

	return float64(reliableCount) / float64(len(proof.Measurements))
}

func (cpe *ConnectivityProofEngine) calculatePerformanceScore(proof ConnectivityProof) float64 {
	if len(proof.Measurements) == 0 {
		return 0.0
	}

	var totalLatency time.Duration
	var totalPacketLoss float64

	for _, measurement := range proof.Measurements {
		totalLatency += measurement.Latency
		totalPacketLoss += measurement.PacketLoss
	}

	avgLatency := totalLatency / time.Duration(len(proof.Measurements))
	avgPacketLoss := totalPacketLoss / float64(len(proof.Measurements))

	// Score based on latency (lower is better) and packet loss (lower is better)
	latencyScore := 1.0 - math.Min(float64(avgLatency.Milliseconds())/1000.0, 1.0)
	packetLossScore := 1.0 - math.Min(avgPacketLoss, 1.0)

	return (latencyScore + packetLossScore) / 2.0
}

func (cpe *ConnectivityProofEngine) calculateSecurityScore(proof ConnectivityProof) float64 {
	// Simplified security score based on proof verification
	if proof.Verified {
		return 0.8 // Base security score for verified proofs
	}
	return 0.2
}

func (cpe *ConnectivityProofEngine) calculateRedundancyScore(proof ConnectivityProof) float64 {
	// Score based on number of measurements (more redundancy is better)
	measurementCount := len(proof.Measurements)
	if measurementCount == 0 {
		return 0.0
	}

	// Normalize to 0-1 scale, with diminishing returns
	return math.Min(float64(measurementCount)/10.0, 1.0)
}

// generateNRVSignature creates a signature for NRV metadata (simplified implementation)
func (cpe *ConnectivityProofEngine) generateNRVSignature(nrvMetadata *NRVMetadata) string {
	// Create hash of NRV data
	data := fmt.Sprintf("%s:%s:%s:%f:%s:%d",
		nrvMetadata.NRVID,
		nrvMetadata.ProofID,
		nrvMetadata.NodeID.String(),
		nrvMetadata.ConnectivityScore,
		nrvMetadata.NRNValue.String(),
		nrvMetadata.Timestamp.Unix(),
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
