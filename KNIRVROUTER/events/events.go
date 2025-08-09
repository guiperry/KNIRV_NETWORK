// events/events.go
package events

import (
	"KNIRVROUTER_GO_Verifyer/blockchain"
	"KNIRVROUTER_GO_Verifyer/types"
)

// Define event types for blockchain, consensus, peer updates, etc.
type BlockAddedEvent struct {
	Block *blockchain.Block // Assuming you have Block defined in blockchain package
}

type TransactionAddedEvent struct {
	Transaction *types.Transaction
}
