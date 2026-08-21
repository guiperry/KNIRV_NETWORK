package knirvoracle

import "fmt"

// FailoverClient tries each candidate Client's SubmitSettlementPayout in
// order, returning the first success. This is how a non-root node reaches
// KNIRVORACLE at all: it has no local oracle subprocess of its own, only
// KNIRVGATEWAY's public "/oracle/*" proxy (gateway.knirv.network for
// mainnet, testnet-gateway.knirv.network for testnet — see
// packages/KNIRVGATEWAY/internal/server/server.go's oracle proxy wiring).
// A root node's own local Client (from Manager.GetClient()) should be
// listed first — it's a direct in-process Unix-socket call, strictly
// faster than a public round trip, so there's no reason to prefer the
// gateway path just because it's also available.
type FailoverClient struct {
	candidates []*Client
}

func NewFailoverClient(candidates ...*Client) *FailoverClient {
	return &FailoverClient{candidates: candidates}
}

func (f *FailoverClient) SubmitSettlementPayout(req *SettlementPayoutRequest) (*SettlementPayoutResponse, error) {
	if len(f.candidates) == 0 {
		return nil, fmt.Errorf("no oracle candidates configured")
	}
	var lastErr error
	for _, c := range f.candidates {
		resp, err := c.SubmitSettlementPayout(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all oracle candidates failed: %w", lastErr)
}
