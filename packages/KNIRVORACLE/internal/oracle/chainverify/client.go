// Package chainverify checks whether a transaction was actually accepted
// onto KNIRVCHAIN, and what it actually committed to, before KNIRVORACLE
// authorizes anything against it (see internal/oracle/actuarial, the
// settlement-payout path this backs).
package chainverify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client verifies transaction inclusion by calling KNIRVCHAIN's existing
// accumulator-proof endpoint: GET /proof/tx/{hash} (see
// KNIRV_NETWORK/packages/KNIRVCHAIN/internal/blockchain/blockchain_server.go,
// handleTxAccumProof / GenerateTxAccumProof). A 200 response is proof the
// transaction is included in a committed block, with the committed
// transaction itself in the response body; 404 means it never was —
// KNIRVCHAIN does not generate proofs for pending-pool-only transactions.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 10 * time.Second}}
}

// syndicateCommitmentPayload mirrors KNIRVCHAIN's SyndicateCommitment wire
// schema (internal/blockchain/transaction.go) byte-for-byte. Duplicated
// rather than imported for the same reason backend_server duplicates it
// (see its chain_adapter.go doc comment) — these are separate services that
// only communicate over the wire.
type syndicateCommitmentPayload struct {
	SchemaVersion  string `json:"schema_version"`
	EntityID       string `json:"entity_id"`
	CommitmentHash string `json:"commitment_hash"`
	Amount         uint64 `json:"amount,omitempty"`
}

// txAccumProofResponse decodes only the fields chainverify needs from
// KNIRVCHAIN's TxAccumProof response — the transaction itself, whose Type
// and Data (JSON-marshaled []byte, base64 on the wire and decoded back to
// raw bytes by encoding/json) carry the actual committed claim.
type txAccumProofResponse struct {
	Transaction struct {
		Type string `json:"type"`
		Data []byte `json:"data"`
	} `json:"transaction"`
}

// VerifiedCommitment is what a caller actually needs to authorize against:
// the syndicate transaction type and the commitment it carries, extracted
// and parsed from a chain-verified transaction — never taken from the
// caller's own say-so.
type VerifiedCommitment struct {
	TxType         string
	EntityID       string
	CommitmentHash string
	Amount         uint64
}

// VerifyCommitment fetches and parses the on-chain commitment for txHash.
// It returns (nil, nil) — not an error — when the transaction was never
// accepted onto the chain, so callers can distinguish "not found" from a
// transport/parse failure.
func (c *Client) VerifyCommitment(txHash string) (*VerifiedCommitment, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("chain verification URL not configured")
	}
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash is required")
	}
	resp, err := c.http.Get(c.baseURL + "/proof/tx/" + txHash)
	if err != nil {
		return nil, fmt.Errorf("chain proof request failed: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		return nil, nil
	default:
		return nil, fmt.Errorf("chain proof request returned status %d", resp.StatusCode)
	}

	var proof txAccumProofResponse
	if err := json.NewDecoder(resp.Body).Decode(&proof); err != nil {
		return nil, fmt.Errorf("decode chain proof response: %w", err)
	}
	if proof.Transaction.Type == "" {
		return nil, fmt.Errorf("chain proof response missing transaction type")
	}
	var payload syndicateCommitmentPayload
	if err := json.Unmarshal(proof.Transaction.Data, &payload); err != nil {
		return nil, fmt.Errorf("decode syndicate commitment payload: %w", err)
	}
	return &VerifiedCommitment{
		TxType:         proof.Transaction.Type,
		EntityID:       payload.EntityID,
		CommitmentHash: payload.CommitmentHash,
		Amount:         payload.Amount,
	}, nil
}
