package interfaces

import (
	"KNIRVCHAIN_GO_Verifyer/types"
)

// TransactionEvent represents an event containing a transaction and optionally a block
type TransactionEvent struct {
	Transaction *types.Transaction
	Block       interface{} // Use interface{} to avoid circular imports, will be cast to *blockchain.Block when needed
}

// BlockEvent represents an event containing a block
type BlockEvent struct {
	Block interface{} // Use interface{} to avoid circular imports, will be cast to *blockchain.Block when needed
}

// Listener defines the interface for components that listen to blockchain events
type Listener interface {
	OnTransactionReceived(TransactionEvent)
	OnBlockReceived(TransactionEvent)
}

// Broadcaster defines the interface for broadcasting blockchain events
type Broadcaster interface {
	BroadcastTransaction(TransactionEvent) error
	RegisterListener(Listener)
}
