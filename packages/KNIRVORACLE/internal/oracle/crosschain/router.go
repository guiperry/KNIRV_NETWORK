package crosschain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ibc"
	"go.uber.org/zap"
)

// Router handles cross-chain transfer routing and orchestration
type Router struct {
	ibcHandler    *ibc.Handler
	bridgeManager *BridgeManager
	transfers     map[string]*CrossChainTransfer
	transferQueue *TransferQueue
	logger        *zap.Logger
	governance    *governance.GovernanceSystem
	mu            sync.RWMutex
}

// NewRouter creates a new cross-chain router
func NewRouter(ibcHandler *ibc.Handler, bridgeManager *BridgeManager, governanceSystem *governance.GovernanceSystem, logger *zap.Logger) *Router {
	return &Router{
		ibcHandler:    ibcHandler,
		bridgeManager: bridgeManager,
		transfers:     make(map[string]*CrossChainTransfer),
		transferQueue: NewTransferQueue(),
		logger:        logger,
		governance:    governanceSystem,
	}
}

// InitiateTransfer initiates a cross-chain transfer
func (r *Router) InitiateTransfer(req *TransferRequest) (*TransferReceipt, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid transfer request: %w", err)
	}

	// Validate source bridge exists and is enabled
	if !r.bridgeManager.IsBridgeEnabled(req.SourceChain) {
		return nil, fmt.Errorf("source chain bridge is not enabled")
	}

	// Validate destination bridge exists and is enabled
	if !r.bridgeManager.IsBridgeEnabled(req.DestChain) {
		return nil, fmt.Errorf("destination chain bridge is not enabled")
	}

	// Validate transfer amount
	if err := r.bridgeManager.ValidateTransferAmount(req.DestChain, req.Amount); err != nil {
		return nil, fmt.Errorf("invalid transfer amount: %w", err)
	}

	// Calculate fee
	fee, err := r.bridgeManager.CalculateFee(req.DestChain, req.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fee: %w", err)
	}

	// Generate transfer ID
	transferID := generateTransferID(req)

	// Create transfer
	transfer := &CrossChainTransfer{
		TransferID:       transferID,
		SourceChain:      req.SourceChain,
		DestChain:        req.DestChain,
		Sender:           req.Sender,
		Recipient:        req.Recipient,
		Amount:           req.Amount,
		Denom:            req.Denom,
		TimeoutHeight:    req.TimeoutHeight,
		TimeoutTimestamp: req.TimeoutTimestamp,
		Memo:             req.Memo,
		Status:           StatusPending,
		FeeAmount:        fee,
		FeeDenom:         req.Denom,
		CreatedAt:        time.Now().Unix(),
	}

	// Store transfer
	r.mu.Lock()
	r.transfers[transferID] = transfer
	r.mu.Unlock()

	// Add to queue
	r.transferQueue.Enqueue(transfer)

	// Update status to source locked
	transfer.Status = StatusSourceLocked

	// Create IBC packet for transfer
	packetData, err := json.Marshal(transfer)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transfer: %w", err)
	}

	// Get destination channel
	destChannel, err := r.ibcHandler.GetChannelForChain(req.DestChain)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination channel: %w", err)
	}

	// Create IBC packet
	packet := &ibc.IBCPacket{
		SourcePort:       "transfer",
		SourceChannel:    destChannel.CounterpartyChannelID,
		DestPort:         destChannel.PortID,
		DestChannel:      destChannel.ChannelID,
		DestChainID:      req.DestChain,
		Data:             packetData,
		TimeoutHeight:    req.TimeoutHeight,
		TimeoutTimestamp: req.TimeoutTimestamp,
		CreatedAt:        time.Now(),
	}

	// Send packet via IBC
	if err := r.ibcHandler.SendPacket(packet); err != nil {
		transfer.Status = StatusFailed
		transfer.Error = err.Error()
		return nil, fmt.Errorf("failed to send IBC packet: %w", err)
	}

	// Update status to in transit
	transfer.Status = StatusInTransit

	// Generate transaction hash
	txHash := fmt.Sprintf("%X", sha256.Sum256(packetData))

	r.logger.Info("Initiated cross-chain transfer",
		zap.String("transfer_id", transferID),
		zap.String("source", req.SourceChain.String()),
		zap.String("dest", req.DestChain.String()),
		zap.Uint64("amount", req.Amount),
	)

	return &TransferReceipt{
		TransferID:      transferID,
		SourceChain:     req.SourceChain,
		DestChain:       req.DestChain,
		Status:          transfer.Status.String(),
		FeeAmount:       fee,
		FeeDenom:        req.Denom,
		TransactionHash: txHash,
		Timestamp:       transfer.CreatedAt,
	}, nil
}

// ReceiveTransfer processes an incoming cross-chain transfer
func (r *Router) ReceiveTransfer(transfer *CrossChainTransfer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate transfer exists
	existingTransfer, exists := r.transfers[transfer.TransferID]
	if !exists {
		// New transfer from another chain
		r.transfers[transfer.TransferID] = transfer
		existingTransfer = transfer
	}

	// Update status to destination received
	existingTransfer.Status = StatusDestReceived

	// Validate proof if present
	if transfer.Proof != nil {
		if r.governance == nil {
			return fmt.Errorf("proof validation requires the registered governance validator set")
		}
		if err := ValidateProof(transfer.Proof, transfer, r.governance.ListActiveValidators()); err != nil {
			existingTransfer.Status = StatusFailed
			existingTransfer.Error = fmt.Sprintf("proof validation failed: %v", err)
			return fmt.Errorf("proof validation failed: %w", err)
		}
	}

	// Mark as completed
	existingTransfer.Status = StatusCompleted
	now := time.Now().Unix()
	existingTransfer.CompletedAt = &now

	r.logger.Info("Received cross-chain transfer",
		zap.String("transfer_id", transfer.TransferID),
		zap.String("recipient", transfer.Recipient),
		zap.Uint64("amount", transfer.Amount),
	)

	return nil
}

// GetTransfer retrieves a transfer by ID
func (r *Router) GetTransfer(transferID string) (*CrossChainTransfer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transfer, exists := r.transfers[transferID]
	if !exists {
		return nil, fmt.Errorf("transfer %s not found", transferID)
	}

	return transfer, nil
}

// ListTransfers returns all transfers
func (r *Router) ListTransfers() []*CrossChainTransfer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transfers := make([]*CrossChainTransfer, 0, len(r.transfers))
	for _, transfer := range r.transfers {
		transfers = append(transfers, transfer)
	}

	return transfers
}

// HandleTimeout handles transfer timeouts
func (r *Router) HandleTimeout(transferID string, currentHeight, currentTimestamp uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	transfer, exists := r.transfers[transferID]
	if !exists {
		return fmt.Errorf("transfer %s not found", transferID)
	}

	// Check if transfer is timed out
	if !transfer.IsTimedOut(currentHeight, currentTimestamp) {
		return fmt.Errorf("transfer has not timed out")
	}

	// Update status
	transfer.Status = StatusTimedOut
	now := time.Now().Unix()
	transfer.CompletedAt = &now

	r.logger.Warn("Transfer timed out",
		zap.String("transfer_id", transferID),
		zap.Uint64("current_height", currentHeight),
		zap.Uint64("timeout_height", transfer.TimeoutHeight),
	)

	return nil
}

// RefundTransfer refunds a failed or timed out transfer
func (r *Router) RefundTransfer(transferID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	transfer, exists := r.transfers[transferID]
	if !exists {
		return fmt.Errorf("transfer %s not found", transferID)
	}

	// Check if transfer can be refunded
	if !transfer.CanBeRefunded() {
		return fmt.Errorf("transfer cannot be refunded (status: %s)", transfer.Status.String())
	}

	// Update status
	transfer.Status = StatusRefunded
	now := time.Now().Unix()
	transfer.CompletedAt = &now

	r.logger.Info("Refunded transfer",
		zap.String("transfer_id", transferID),
		zap.String("sender", transfer.Sender),
		zap.Uint64("amount", transfer.Amount),
	)

	return nil
}

// MonitorTimeouts monitors for transfer timeouts
func (r *Router) MonitorTimeouts(currentHeight, currentTimestamp uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, transfer := range r.transfers {
		// Skip completed transfers
		if transfer.IsCompleted() {
			continue
		}

		// Check for timeout
		if transfer.IsTimedOut(currentHeight, currentTimestamp) {
			transfer.Status = StatusTimedOut
			now := time.Now().Unix()
			transfer.CompletedAt = &now

			r.logger.Warn("Transfer timed out during monitoring",
				zap.String("transfer_id", transfer.TransferID),
			)
		}
	}
}

// generateTransferID generates a unique transfer ID
func generateTransferID(req *TransferRequest) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d:%d",
		req.SourceChain.String(),
		req.DestChain.String(),
		req.Sender,
		req.Recipient,
		req.Amount,
		time.Now().UnixNano(),
	)
	return fmt.Sprintf("%X", sha256.Sum256([]byte(data)))
}
