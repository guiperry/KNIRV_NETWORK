package types

import "fmt"

// ChainID represents a blockchain network identifier
type ChainID int

const (
	ChainKnirvChain ChainID = iota
	ChainKnirvOracle
	ChainKnirvNexus
	ChainKnirvRouter
	ChainKnirvGraph
	ChainXion
	ChainCosmosHub
	ChainExternal
)

// String returns the string representation of the ChainID
func (c ChainID) String() string {
	switch c {
	case ChainKnirvChain:
		return "knirvchain"
	case ChainKnirvOracle:
		return "knirvoracle"
	case ChainKnirvNexus:
		return "knirvserver"
	case ChainKnirvRouter:
		return "knirvrouter"
	case ChainKnirvGraph:
		return "knirvgraph"
	case ChainXion:
		return "xion"
	case ChainCosmosHub:
		return "cosmos"
	case ChainExternal:
		return "external"
	default:
		return "unknown"
	}
}

// ChainIDFromString parses a string into a ChainID
func ChainIDFromString(s string) (ChainID, error) {
	switch s {
	case "knirvchain":
		return ChainKnirvChain, nil
	case "knirvoracle":
		return ChainKnirvOracle, nil
	case "knirvserverr":
		return ChainKnirvNexus, nil
	case "knirvrouter":
		return ChainKnirvRouter, nil
	case "knirvgraph":
		return ChainKnirvGraph, nil
	case "xion":
		return ChainXion, nil
	case "cosmos":
		return ChainCosmosHub, nil
	case "external":
		return ChainExternal, nil
	default:
		return 0, fmt.Errorf("unknown chain ID: %s", s)
	}
}

// IsKnirvChain returns true if this is a KNIRV network chain
func (c ChainID) IsKnirvChain() bool {
	return c >= ChainKnirvChain && c <= ChainKnirvGraph
}

// IsExternal returns true if this is an external chain (non-KNIRV)
func (c ChainID) IsExternal() bool {
	return c >= ChainXion
}
