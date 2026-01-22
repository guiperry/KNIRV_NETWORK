// faucet_wrapper.go
package faucet

import (
	"math/big"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

// FaucetIntegrationWrapper wraps FaucetIntegration to implement the interface
type FaucetIntegrationWrapper struct {
	faucetIntegration *FaucetIntegration
}

// NewFaucetIntegrationWrapper creates a new wrapper
func NewFaucetIntegrationWrapper(faucetIntegration *FaucetIntegration) *FaucetIntegrationWrapper {
	return &FaucetIntegrationWrapper{
		faucetIntegration: faucetIntegration,
	}
}

// RequestConnectivityReward implements the interface
func (fiw *FaucetIntegrationWrapper) RequestConnectivityReward(nodeID peer.ID, proofID string, score float64, amount *big.Int) (FaucetRequestInterface, error) {
	return fiw.faucetIntegration.RequestConnectivityReward(nodeID, proofID, score, amount)
}

// FaucetRequestInterface is defined in starter package, but we need to reference it here
type FaucetRequestInterface interface {
	GetRequestID() string
	GetNodeID() peer.ID
	GetAmount() *big.Int
	GetReason() string
	GetTimestamp() time.Time
	GetStatus() string
	GetTxHash() string
	GetErrorMessage() string
}
