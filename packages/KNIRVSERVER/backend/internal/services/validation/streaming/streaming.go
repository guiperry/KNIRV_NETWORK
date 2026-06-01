// Package streaming provides canonical types for financial tick data streaming.
// These types are promoted from the fintech plugin package to a platform-level
// package so they can be shared across validators and market data consumers.
package streaming

import (
	"backend_server/internal/services/plugins/fintech"
)

// Re-export canonical types from the fintech package.

// TickData is an alias for fintech.TickData.
type TickData = fintech.TickData

// BarData is an alias for fintech.BarData.
type BarData = fintech.BarData

// TickDataStream is an alias for fintech.TickDataStream.
type TickDataStream = fintech.TickDataStream

// TickStreamManager is an alias for fintech.TickStreamManager.
type TickStreamManager = fintech.TickStreamManager

// TickDataWriter is an alias for fintech.TickDataWriter.
type TickDataWriter = fintech.TickDataWriter

// TickDataFilter is an alias for fintech.TickDataFilter.
type TickDataFilter = fintech.TickDataFilter

// TickDataType is an alias for fintech.TickDataType.
type TickDataType = fintech.TickDataType

// Tick type constants re-exported for convenience.
const (
	TickTypeTrade = fintech.TickTypeTrade
	TickTypeQuote = fintech.TickTypeQuote
	TickTypeBid   = fintech.TickTypeBid
	TickTypeAsk   = fintech.TickTypeAsk
)

// NewTickStreamManager creates a new TickStreamManager.
func NewTickStreamManager() *TickStreamManager {
	return fintech.NewTickStreamManager()
}

// NewTickDataWriter creates a new TickDataWriter.
func NewTickDataWriter() *TickDataWriter {
	return fintech.NewTickDataWriter()
}
