package transformer

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const ChainAssertionAttestationType = "assertion_attestation"

// ChainAssertionAttestation is intentionally wire-compatible with
// KNIRVCHAIN's AssertionAttestation. Packages communicate over HTTP rather
// than importing each other's internal Go modules.
type ChainAssertionAttestation struct {
	SchemaVersion    int32  `json:"schema_version"`
	AssertionKey     string `json:"assertion_key"`
	CommitmentTarget uint32 `json:"commitment_target"`
	ContextHash      uint32 `json:"context_hash"`
	BestSeed         string `json:"best_seed"`
}

func ChainAttestationFromProof(proof AttestationProof) ChainAssertionAttestation {
	return ChainAssertionAttestation{SchemaVersion: proof.SchemaVersion, AssertionKey: proof.AssertionKey, CommitmentTarget: proof.CommitmentTarget, ContextHash: proof.ContextHash, BestSeed: hex.EncodeToString(proof.BestSeed)}
}

// SubmitSignedChainTransaction submits an already-signed transaction to the
// chain. Signing remains with the node wallet; HASHER never handles a private
// key. The function deliberately refuses a mismatched transaction type.
func SubmitSignedChainTransaction(ctx context.Context, chainURL string, signedTransaction json.RawMessage) (string, error) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(signedTransaction, &envelope); err != nil {
		return "", fmt.Errorf("decode signed transaction: %w", err)
	}
	if envelope.Type != ChainAssertionAttestationType {
		return "", fmt.Errorf("unexpected transaction type %q", envelope.Type)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(chainURL, "/")+"/transaction", bytes.NewReader(signedTransaction))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("chain returned %d", resp.StatusCode)
	}
	var body struct {
		TxHash string `json:"tx_hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return body.TxHash, nil
}
