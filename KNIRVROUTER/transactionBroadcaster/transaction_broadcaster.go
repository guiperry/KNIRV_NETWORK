package transactionBroadcaster

import "KNIRVCHAIN_GO_Verifyer/types"

type TransactionAddedEvent struct {
	Transaction *types.Transaction
}

type TransactionAddedChan chan TransactionAddedEvent

type TransactionBroadcaster interface {
	BroadcastTransaction(TransactionAddedEvent)
}
