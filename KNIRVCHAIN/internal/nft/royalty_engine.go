package nft

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"KNIRVCHAIN/internal/blockchain"
	"KNIRVCHAIN/internal/graph"
	"KNIRVCHAIN/internal/types"
)

// RoyaltyEngine handles royalty calculation and distribution
type RoyaltyEngine struct {
	graphQueries *graph.GraphQueries
	blockchain   *blockchain.BlockchainStruct
	mutex        sync.RWMutex
}

// NewRoyaltyEngine creates a new royalty engine
func NewRoyaltyEngine(graphQueries *graph.GraphQueries, bc *blockchain.BlockchainStruct) *RoyaltyEngine {
	return &RoyaltyEngine{
		graphQueries: graphQueries,
		blockchain:   bc,
	}
}

// RoyaltyDistribution represents a royalty distribution event
type RoyaltyDistribution struct {
	PropertyNodeID string                    `json:"property_node_id"`
	UsageAmount    uint64                    `json:"usage_amount"`
	TotalRoyalties uint64                    `json:"total_royalties"`
	Distributions  map[string]*RoyaltyShare  `json:"distributions"`
	DistributedAt  int64                     `json:"distributed_at"`
	TransactionID  string                    `json:"transaction_id,omitempty"`
}

// RoyaltyShare represents a share of royalties for a specific recipient
type RoyaltyShare struct {
	Recipient string `json:"recipient"`
	Amount    uint64 `json:"amount"`
	Share     uint8  `json:"share"` // percentage (0-100)
}

// CalculateRoyalties calculates royalties for property usage
func (re *RoyaltyEngine) CalculateRoyalties(propertyNodeID string, usageAmount uint64) (*RoyaltyDistribution, error) {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	// Get the property node
	propertyNode, err := re.graphQueries.GetNodeStore().GetPropertyNode(propertyNodeID)
	if err != nil {
		return nil, fmt.Errorf("property node not found: %w", err)
	}

	// Calculate total royalties (typically a percentage of usage value)
	royaltyRate := 0.05 // 5% royalty rate (configurable)
	totalRoyalties := uint64(float64(usageAmount) * royaltyRate)

	// Get royalty distribution from the property
	distribution := propertyNode.Royalties.CalculateDistribution(totalRoyalties)

	// Convert to RoyaltyShare format
	distributions := make(map[string]*RoyaltyShare)
	for recipient, amount := range distribution {
		share := re.getSharePercentage(propertyNode.Royalties, recipient)
		distributions[recipient] = &RoyaltyShare{
			Recipient: recipient,
			Amount:    amount,
			Share:     share,
		}
	}

	royaltyDist := &RoyaltyDistribution{
		PropertyNodeID: propertyNodeID,
		UsageAmount:    usageAmount,
		TotalRoyalties: totalRoyalties,
		Distributions:  distributions,
		DistributedAt:  time.Now().Unix(),
	}

	return royaltyDist, nil
}

// getSharePercentage gets the share percentage for a recipient
func (re *RoyaltyEngine) getSharePercentage(royalties types.RoyaltyStructure, recipient string) uint8 {
	// Check origin NIM share
	if recipient == "origin_nim" {
		return royalties.OriginNIMShare
	}

	// Check network share
	if recipient == "network" {
		return royalties.NetworkShare
	}

	// Check dependency shares
	if share, exists := royalties.DependencyShares[recipient]; exists {
		return share
	}

	return 0
}

// DistributeRoyalties distributes calculated royalties
func (re *RoyaltyEngine) DistributeRoyalties(distribution *RoyaltyDistribution) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	// Create distribution transactions for each recipient
	for recipient, share := range distribution.Distributions {
		if share.Amount == 0 {
			continue
		}

		err := re.createRoyaltyPayment(recipient, share.Amount, distribution)
		if err != nil {
			log.Printf("Failed to distribute royalty to %s: %v", recipient, share.Amount, err)
			// Continue with other distributions even if one fails
		}
	}

	log.Printf("Royalties distributed for property %s: %d total to %d recipients",
		distribution.PropertyNodeID, distribution.TotalRoyalties, len(distribution.Distributions))

	return nil
}

// createRoyaltyPayment creates a payment transaction for royalties
func (re *RoyaltyEngine) createRoyaltyPayment(recipient string, amount uint64, distribution *RoyaltyDistribution) error {
	// Get the property node to determine the payer
	propertyNode, err := re.graphQueries.GetNodeStore().GetPropertyNode(distribution.PropertyNodeID)
	if err != nil {
		return fmt.Errorf("failed to get property node: %w", err)
	}

	// The current owner pays the royalties
	payer := propertyNode.Ownership.CurrentOwner

	// Check if payer has sufficient balance
	payerBalance := re.blockchain.CalculateTotalCrypto(payer)
	if payerBalance < amount {
		return fmt.Errorf("insufficient balance for royalty payment: %d < %d", payerBalance, amount)
	}

	// Create royalty payment transaction
	royaltyData := map[string]interface{}{
		"property_node_id": distribution.PropertyNodeID,
		"recipient":        recipient,
		"amount":           amount,
		"usage_amount":     distribution.UsageAmount,
		"distributed_at":   distribution.DistributedAt,
	}

	// This would need to be implemented as a proper transaction type
	// For now, we'll simulate the distribution
	log.Printf("Royalty payment: %d NRN from %s to %s for property %s",
		amount, payer, recipient, distribution.PropertyNodeID)

	return nil
}

// TrackUsage tracks usage of a property for royalty calculation
func (re *RoyaltyEngine) TrackUsage(propertyNodeID, userAddress string, usageType string, usageValue uint64) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	// Record usage event
	usageEvent := &UsageEvent{
		PropertyNodeID: propertyNodeID,
		UserAddress:    userAddress,
		UsageType:      usageType,
		UsageValue:     usageValue,
		Timestamp:      time.Now().Unix(),
	}

	// Store usage event (this would go to a database)
	err := re.storeUsageEvent(usageEvent)
	if err != nil {
		return fmt.Errorf("failed to store usage event: %w", err)
	}

	// Check if royalties should be distributed (e.g., accumulated threshold)
	shouldDistribute, err := re.shouldDistributeRoyalties(propertyNodeID)
	if err != nil {
		return fmt.Errorf("failed to check distribution threshold: %w", err)
	}

	if shouldDistribute {
		err = re.processAccumulatedRoyalties(propertyNodeID)
		if err != nil {
			return fmt.Errorf("failed to process accumulated royalties: %w", err)
		}
	}

	log.Printf("Usage tracked: property %s, user %s, type %s, value %d",
		propertyNodeID, userAddress, usageType, usageValue)

	return nil
}

// UsageEvent represents a usage event for royalty calculation
type UsageEvent struct {
	PropertyNodeID string `json:"property_node_id"`
	UserAddress    string `json:"user_address"`
	UsageType      string `json:"usage_type"`
	UsageValue     uint64 `json:"usage_value"`
	Timestamp      int64  `json:"timestamp"`
}

// storeUsageEvent stores a usage event
func (re *RoyaltyEngine) storeUsageEvent(event *UsageEvent) error {
	// This would store the event in a database
	// For now, just log it
	log.Printf("Usage event stored: %+v", event)
	return nil
}

// shouldDistributeRoyalties checks if accumulated royalties should be distributed
func (re *RoyaltyEngine) shouldDistributeRoyalties(propertyNodeID string) (bool, error) {
	// This would check accumulated usage against distribution thresholds
	// For now, distribute on every usage (for demonstration)
	return true, nil
}

// processAccumulatedRoyalties processes and distributes accumulated royalties
func (re *RoyaltyEngine) processAccumulatedRoyalties(propertyNodeID string) error {
	// Get accumulated usage for the property
	accumulatedUsage, err := re.getAccumulatedUsage(propertyNodeID)
	if err != nil {
		return fmt.Errorf("failed to get accumulated usage: %w", err)
	}

	if accumulatedUsage == 0 {
		return nil // Nothing to distribute
	}

	// Calculate royalties
	distribution, err := re.CalculateRoyalties(propertyNodeID, accumulatedUsage)
	if err != nil {
		return fmt.Errorf("failed to calculate royalties: %w", err)
	}

	// Distribute royalties
	err = re.DistributeRoyalties(distribution)
	if err != nil {
		return fmt.Errorf("failed to distribute royalties: %w", err)
	}

	// Reset accumulated usage
	err = re.resetAccumulatedUsage(propertyNodeID)
	if err != nil {
		log.Printf("Warning: failed to reset accumulated usage: %v", err)
	}

	return nil
}

// getAccumulatedUsage gets the accumulated usage for a property
func (re *RoyaltyEngine) getAccumulatedUsage(propertyNodeID string) (uint64, error) {
	// This would query accumulated usage from database
	// For now, return a mock value
	return 1000, nil
}

// resetAccumulatedUsage resets the accumulated usage counter
func (re *RoyaltyEngine) resetAccumulatedUsage(propertyNodeID string) error {
	// This would reset the accumulator in database
	log.Printf("Accumulated usage reset for property: %s", propertyNodeID)
	return nil
}

// GetRoyaltyHistory gets the royalty distribution history for a property
func (re *RoyaltyEngine) GetRoyaltyHistory(propertyNodeID string, limit int) ([]*RoyaltyDistribution, error) {
	// This would query royalty distribution history from database
	// For now, return empty slice
	return []*RoyaltyDistribution{}, nil
}

// GetRoyaltyEarnings gets total royalty earnings for an address
func (re *RoyaltyEngine) GetRoyaltyEarnings(address string) (*RoyaltyEarnings, error) {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	earnings := &RoyaltyEarnings{
		Address:         address,
		TotalEarned:     0,
		PendingEarnings: 0,
		Claims:          make(map[string]*RoyaltyClaim),
	}

	// Get all properties owned by this address
	properties, err := re.graphQueries.FindPropertyNodesByOwner(address, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get properties for address: %w", err)
	}

	for _, property := range properties {
		// Calculate pending royalties for this property
		accumulatedUsage, err := re.getAccumulatedUsage(property.ID)
		if err != nil {
			continue
		}

		if accumulatedUsage > 0 {
			distribution, err := re.CalculateRoyalties(property.ID, accumulatedUsage)
			if err != nil {
				continue
			}

			// Add to pending earnings
			if share, exists := distribution.Distributions[address]; exists {
				earnings.PendingEarnings += share.Amount
				earnings.Claims[property.ID] = &RoyaltyClaim{
					PropertyNodeID: property.ID,
					Amount:         share.Amount,
					CanClaim:       true,
				}
			}
		}
	}

	return earnings, nil
}

// RoyaltyEarnings represents royalty earnings for an address
type RoyaltyEarnings struct {
	Address         string                    `json:"address"`
	TotalEarned     uint64                    `json:"total_earned"`
	PendingEarnings uint64                    `json:"pending_earnings"`
	Claims          map[string]*RoyaltyClaim  `json:"claims"`
}

// RoyaltyClaim represents a claimable royalty amount
type RoyaltyClaim struct {
	PropertyNodeID string `json:"property_node_id"`
	Amount         uint64 `json:"amount"`
	CanClaim       bool   `json:"can_claim"`
}

// ClaimRoyalties allows an address to claim pending royalties
func (re *RoyaltyEngine) ClaimRoyalties(address string, propertyNodeIDs []string) error {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	earnings, err := re.GetRoyaltyEarnings(address)
	if err != nil {
		return fmt.Errorf("failed to get royalty earnings: %w", err)
	}

	totalClaim := uint64(0)
	validClaims := make([]string, 0)

	for _, propertyID := range propertyNodeIDs {
		if claim, exists := earnings.Claims[propertyID]; exists && claim.CanClaim {
			totalClaim += claim.Amount
			validClaims = append(validClaims, propertyID)
		}
	}

	if totalClaim == 0 {
		return fmt.Errorf("no valid claims found")
	}

	// Process the claim
	err = re.processRoyaltyClaim(address, validClaims, totalClaim)
	if err != nil {
		return fmt.Errorf("failed to process royalty claim: %w", err)
	}

	log.Printf("Royalties claimed: %d NRN by %s from %d properties", totalClaim, address, len(validClaims))
	return nil
}

// processRoyaltyClaim processes a royalty claim
func (re *RoyaltyEngine) processRoyaltyClaim(address string, propertyNodeIDs []string, totalAmount uint64) error {
	// This would create appropriate transactions to transfer royalties
	// For now, simulate the claim
	log.Printf("Royalty claim processed: %d NRN to %s", totalAmount, address)
	return nil
}

// GetRoyaltyAnalytics gets analytics about royalty distributions
func (re *RoyaltyEngine) GetRoyaltyAnalytics(timeRange string) (*RoyaltyAnalytics, error) {
	re.mutex.RLock()
	defer re.mutex.RUnlock()

	analytics := &RoyaltyAnalytics{
		TimeRange:       timeRange,
		TotalDistributed: 0,
		TotalEarned:     0,
		TopEarners:      make([]*EarnerStats, 0),
		PropertyStats:   make(map[string]*PropertyRoyaltyStats),
	}

	// This would aggregate analytics from royalty distribution history
	// For now, return mock data

	analytics.TotalDistributed = 10000
	analytics.TotalEarned = 9500 // 95% of distributed amount

	return analytics, nil
}

// RoyaltyAnalytics represents royalty distribution analytics
type RoyaltyAnalytics struct {
	TimeRange       string                            `json:"time_range"`
	TotalDistributed uint64                           `json:"total_distributed"`
	TotalEarned     uint64                           `json:"total_earned"`
	TopEarners      []*EarnerStats                    `json:"top_earners"`
	PropertyStats   map[string]*PropertyRoyaltyStats `json:"property_stats"`
}

// EarnerStats represents statistics for a royalty earner
type EarnerStats struct {
	Address    string `json:"address"`
	TotalEarned uint64 `json:"total_earned"`
	Properties int    `json:"properties"`
}

// PropertyRoyaltyStats represents royalty statistics for a property
type PropertyRoyaltyStats struct {
	PropertyNodeID   string `json:"property_node_id"`
	TotalDistributed uint64 `json:"total_distributed"`
	UsageCount       uint64 `json:"usage_count"`
}

// StartRoyaltyProcessor starts background royalty processing
func (re *RoyaltyEngine) StartRoyaltyProcessor(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				re.processPendingRoyalties()
			}
		}
	}()
}

// processPendingRoyalties processes any pending royalty distributions
func (re *RoyaltyEngine) processPendingRoyalties() {
	// Get all properties with accumulated usage
	// This is a simplified implementation
	log.Println("Processing pending royalties...")
}