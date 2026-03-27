// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package nexus

import (
	"github.com/apache/arrow/go/v14/arrow"
)

// NewAgentMemorySchema defines the "Living Data" structure for an agent session.
func NewAgentMemorySchema() *arrow.Schema {
	return arrow.NewSchema(
		[]arrow.Field{
			{Name: "timestamp", Type: arrow.PrimitiveTypes.Int64},
			{Name: "agent_id", Type: arrow.BinaryTypes.String},
			{Name: "intent", Type: arrow.BinaryTypes.String},          // What the LLM said it would do
			{Name: "observed_action", Type: arrow.BinaryTypes.String}, // What eBPF actually saw
			{Name: "token_usage", Type: arrow.PrimitiveTypes.Int32},
			{Name: "context_relevance", Type: arrow.PrimitiveTypes.Float64},
			{Name: "verified", Type: arrow.FixedWidthTypes.Boolean}, // Whether intent matches action
		},
		nil,
	)
}
