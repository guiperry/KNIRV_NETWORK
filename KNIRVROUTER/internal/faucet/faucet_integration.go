// faucet_integration.go
package faucet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// FaucetIntegration handles NRN token minting and faucet requests
type FaucetIntegration struct {
	faucetEndpoint    string
	httpClient        *http.Client
	requestHistory    map[string]*FaucetRequest
	historyMutex      sync.RWMutex
	rateLimiter       map[peer.ID]time.Time
	rateLimiterMutex  sync.RWMutex
	maxRequestsPerDay int
	minInterval       time.Duration
}

// FaucetRequest represents a request to the faucet
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

// Getter methods for FaucetRequest to implement the interface
func (fr *FaucetRequest) GetRequestID() string    { return fr.RequestID }
func (fr *FaucetRequest) GetNodeID() peer.ID      { return fr.NodeID }
func (fr *FaucetRequest) GetAmount() *big.Int     { return fr.Amount }
func (fr *FaucetRequest) GetReason() string       { return fr.Reason }
func (fr *FaucetRequest) GetTimestamp() time.Time { return fr.Timestamp }
func (fr *FaucetRequest) GetStatus() string       { return fr.Status }
func (fr *FaucetRequest) GetTxHash() string       { return fr.TxHash }
func (fr *FaucetRequest) GetErrorMessage() string { return fr.ErrorMessage }

// FaucetResponse represents the response from the faucet
type FaucetResponse struct {
	Success   bool   `json:"success"`
	TxHash    string `json:"tx_hash,omitempty"`
	Message   string `json:"message"`
	Amount    string `json:"amount,omitempty"`
	Recipient string `json:"recipient,omitempty"`
}

// FaucetConfig contains configuration for faucet integration
type FaucetConfig struct {
	FaucetEndpoint    string        `json:"faucet_endpoint"`
	MaxRequestsPerDay int           `json:"max_requests_per_day"`
	MinInterval       time.Duration `json:"min_interval"`
	DefaultAmount     *big.Int      `json:"default_amount"`
	Timeout           time.Duration `json:"timeout"`
}

// NewFaucetIntegration creates a new faucet integration instance
func NewFaucetIntegration(config FaucetConfig) *FaucetIntegration {
	return &FaucetIntegration{
		faucetEndpoint:    config.FaucetEndpoint,
		httpClient:        &http.Client{Timeout: config.Timeout},
		requestHistory:    make(map[string]*FaucetRequest),
		rateLimiter:       make(map[peer.ID]time.Time),
		maxRequestsPerDay: config.MaxRequestsPerDay,
		minInterval:       config.MinInterval,
	}
}

// RequestNRNTokens requests NRN tokens from the faucet
func (fi *FaucetIntegration) RequestNRNTokens(nodeID peer.ID, amount *big.Int, reason string) (*FaucetRequest, error) {
	// Check rate limiting
	if !fi.checkRateLimit(nodeID) {
		return nil, fmt.Errorf("rate limit exceeded for node %s", nodeID.String())
	}

	// Create request
	request := &FaucetRequest{
		RequestID: fmt.Sprintf("faucet_req_%d", time.Now().UnixNano()),
		NodeID:    nodeID,
		Amount:    amount,
		Reason:    reason,
		Timestamp: time.Now(),
		Status:    "pending",
	}

	// Store request
	fi.historyMutex.Lock()
	fi.requestHistory[request.RequestID] = request
	fi.historyMutex.Unlock()

	// Update rate limiter
	fi.rateLimiterMutex.Lock()
	fi.rateLimiter[nodeID] = time.Now()
	fi.rateLimiterMutex.Unlock()

	// Submit request to faucet
	go fi.submitFaucetRequest(request)

	log.Printf("NRN faucet request created: ID=%s, NodeID=%s, Amount=%s, Reason=%s",
		request.RequestID, nodeID.String()[:8], amount.String(), reason)

	return request, nil
}

// checkRateLimit checks if the node can make a faucet request
func (fi *FaucetIntegration) checkRateLimit(nodeID peer.ID) bool {
	fi.rateLimiterMutex.RLock()
	lastRequest, exists := fi.rateLimiter[nodeID]
	fi.rateLimiterMutex.RUnlock()

	if !exists {
		return true
	}

	// Check minimum interval
	if time.Since(lastRequest) < fi.minInterval {
		return false
	}

	// Check daily limit
	dayStart := time.Now().Truncate(24 * time.Hour)
	requestsToday := fi.countRequestsSince(nodeID, dayStart)

	return requestsToday < fi.maxRequestsPerDay
}

// countRequestsSince counts requests from a node since a given time
func (fi *FaucetIntegration) countRequestsSince(nodeID peer.ID, since time.Time) int {
	fi.historyMutex.RLock()
	defer fi.historyMutex.RUnlock()

	count := 0
	for _, request := range fi.requestHistory {
		if request.NodeID == nodeID && request.Timestamp.After(since) {
			count++
		}
	}

	return count
}

// submitFaucetRequest submits the request to the faucet endpoint
func (fi *FaucetIntegration) submitFaucetRequest(request *FaucetRequest) {
	log.Printf("Submitting faucet request %s to endpoint %s", request.RequestID, fi.faucetEndpoint)

	// Prepare request payload
	payload := map[string]interface{}{
		"recipient": request.NodeID.String(),
		"amount":    request.Amount.String(),
		"reason":    request.Reason,
		"timestamp": request.Timestamp.Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fi.updateRequestStatus(request.RequestID, "failed", "", fmt.Sprintf("Failed to marshal payload: %v", err))
		return
	}

	// Make HTTP request
	resp, err := fi.httpClient.Post(fi.faucetEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fi.updateRequestStatus(request.RequestID, "failed", "", fmt.Sprintf("HTTP request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fi.updateRequestStatus(request.RequestID, "failed", "", fmt.Sprintf("Failed to read response: %v", err))
		return
	}

	// Parse response
	var faucetResp FaucetResponse
	if err := json.Unmarshal(responseBody, &faucetResp); err != nil {
		fi.updateRequestStatus(request.RequestID, "failed", "", fmt.Sprintf("Failed to parse response: %v", err))
		return
	}

	// Update request status based on response
	if faucetResp.Success {
		fi.updateRequestStatus(request.RequestID, "completed", faucetResp.TxHash, "")
		log.Printf("Faucet request %s completed successfully: TxHash=%s", request.RequestID, faucetResp.TxHash)
	} else {
		fi.updateRequestStatus(request.RequestID, "failed", "", faucetResp.Message)
		log.Printf("Faucet request %s failed: %s", request.RequestID, faucetResp.Message)
	}
}

// updateRequestStatus updates the status of a faucet request
func (fi *FaucetIntegration) updateRequestStatus(requestID, status, txHash, errorMessage string) {
	fi.historyMutex.Lock()
	defer fi.historyMutex.Unlock()

	if request, exists := fi.requestHistory[requestID]; exists {
		request.Status = status
		request.TxHash = txHash
		request.ErrorMessage = errorMessage
	}
}

// GetRequestStatus returns the status of a faucet request
func (fi *FaucetIntegration) GetRequestStatus(requestID string) (*FaucetRequest, error) {
	fi.historyMutex.RLock()
	defer fi.historyMutex.RUnlock()

	request, exists := fi.requestHistory[requestID]
	if !exists {
		return nil, fmt.Errorf("request not found: %s", requestID)
	}

	// Return a copy to avoid race conditions
	requestCopy := *request
	return &requestCopy, nil
}

// GetRequestHistory returns the faucet request history for a node
func (fi *FaucetIntegration) GetRequestHistory(nodeID peer.ID) []*FaucetRequest {
	fi.historyMutex.RLock()
	defer fi.historyMutex.RUnlock()

	var history []*FaucetRequest
	for _, request := range fi.requestHistory {
		if request.NodeID == nodeID {
			requestCopy := *request
			history = append(history, &requestCopy)
		}
	}

	return history
}

// GetFaucetStats returns statistics about faucet usage
func (fi *FaucetIntegration) GetFaucetStats() map[string]interface{} {
	fi.historyMutex.RLock()
	defer fi.historyMutex.RUnlock()

	totalRequests := len(fi.requestHistory)
	completedRequests := 0
	failedRequests := 0
	pendingRequests := 0
	var totalAmount big.Int

	for _, request := range fi.requestHistory {
		switch request.Status {
		case "completed":
			completedRequests++
			totalAmount.Add(&totalAmount, request.Amount)
		case "failed":
			failedRequests++
		case "pending":
			pendingRequests++
		}
	}

	return map[string]interface{}{
		"total_requests":      totalRequests,
		"completed_requests":  completedRequests,
		"failed_requests":     failedRequests,
		"pending_requests":    pendingRequests,
		"total_amount_minted": totalAmount.String(),
		"success_rate":        float64(completedRequests) / float64(totalRequests) * 100,
		"faucet_endpoint":     fi.faucetEndpoint,
	}
}

// RequestConnectivityReward requests NRN tokens as a reward for connectivity proof
func (fi *FaucetIntegration) RequestConnectivityReward(nodeID peer.ID, proofID string, score float64, amount *big.Int) (*FaucetRequest, error) {
	reason := fmt.Sprintf("connectivity_proof_%s_score_%.2f", proofID, score)
	return fi.RequestNRNTokens(nodeID, amount, reason)
}

// RequestParticipationReward requests NRN tokens as a reward for network participation
func (fi *FaucetIntegration) RequestParticipationReward(nodeID peer.ID, participationType string, amount *big.Int) (*FaucetRequest, error) {
	reason := fmt.Sprintf("participation_reward_%s", participationType)
	return fi.RequestNRNTokens(nodeID, amount, reason)
}

// CleanupOldRequests removes old requests from history to prevent memory leaks
func (fi *FaucetIntegration) CleanupOldRequests(maxAge time.Duration) {
	fi.historyMutex.Lock()
	defer fi.historyMutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for requestID, request := range fi.requestHistory {
		if request.Timestamp.Before(cutoff) {
			delete(fi.requestHistory, requestID)
		}
	}

	// Also cleanup rate limiter
	fi.rateLimiterMutex.Lock()
	defer fi.rateLimiterMutex.Unlock()

	for nodeID, lastRequest := range fi.rateLimiter {
		if lastRequest.Before(cutoff) {
			delete(fi.rateLimiter, nodeID)
		}
	}
}

// StartCleanupRoutine starts a routine to periodically cleanup old requests
func (fi *FaucetIntegration) StartCleanupRoutine(ctx context.Context, interval, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fi.CleanupOldRequests(maxAge)
		}
	}
}
