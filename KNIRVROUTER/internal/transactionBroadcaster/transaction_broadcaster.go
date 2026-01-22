package transactionBroadcaster

import "KNIRVROUTER/internal/types"

type TransactionAddedEvent struct {
	Transaction *types.Transaction
}

type TransactionAddedChan chan TransactionAddedEvent

type TransactionBroadcaster interface {
	BroadcastTransaction(TransactionAddedEvent)
}
