package blockchain

import (
	"bytes"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// EventBundleNFT is the KNIRVCHAIN-side commit for chain_refactor.md's core
// requirement: KNIRVCHAIN mints one NFT per CLI decision/error event,
// bundling the Skills, Capabilities (MCPs), Assets, Context, and Credentials
// that event actually used. Unlike the pre-existing Skill/Capability/
// Property mining pipelines (internal/mining), which require a heavyweight
// pre-existing ErrorNode/ContextNode/IdeaNode, this bundle is built directly
// from the lightweight resource references the CLI already captures at
// decision time (dve.ResourceRef) — minting one bundle per decision has to
// be cheap enough to happen inline in the commit path (chain_refactor.md
// Open Decision #4).
const (
	eventBundleMintSchema     = "knirv.event-bundle.v1"
	eventBundleReceiptSchema  = "event_bundle_receipt.v1"
	eventBundleKeyPrefix      = "event_bundle:bundle:"
	eventBundleEventKeyPrefix = "event_bundle:event:"

	// eventBundleBurnAddress is the sink NRN is transferred to when burned to
	// mint a bundle. It has no corresponding wallet — nothing ever spends
	// from it — so a transfer into it is an effective burn while still using
	// the Transaction Chain's ordinary balance ledger (no separate burn
	// bookkeeping needed there).
	eventBundleBurnAddress = "0x000000000000000000000000000000000000dead"
)

// eventBundleMintCostNRN returns the flat NRN cost of minting one event
// bundle. KNIRVCHAIN stays sovereign — it burns NRN to mint bundles but does
// not itself mint or hold the NRN economy (that's KNIRVORACLE's Faucet and
// the Transaction Chain's temporary pool, see KNIRV_NETWORK's
// pkg/embedded/transactionchain). Overridable for deployments that want a
// different price; there is no existing per-bundle pricing model to inherit
// from (internal/mining's calculateNRNCost prices capability invocations, a
// different, heavier-weight mint path).
func eventBundleMintCostNRN() uint64 {
	if raw := strings.TrimSpace(os.Getenv("KNIRV_EVENT_BUNDLE_MINT_COST_NRN")); raw != "" {
		if cost, err := strconv.ParseUint(raw, 10, 64); err == nil {
			return cost
		}
	}
	return 10
}

// transactionChainBaseURL resolves the embedded Transaction Chain's local
// HTTP API (KNIRVSERVER's pkg/embedded/transactionchain), where the
// temporary NRN pool actually lives.
func transactionChainBaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("KNIRV_TRANSACTION_CHAIN_URL")); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	return "http://127.0.0.1:9190"
}

// burnNRNForBundleMint debits amount NRN from minterAddress's Transaction
// Chain wallet balance by transferring it to eventBundleBurnAddress. Returns
// an error (including insufficient funds) if the debit cannot be applied —
// callers must not mint the bundle in that case.
func burnNRNForBundleMint(minterAddress string, amount uint64) error {
	if amount == 0 {
		return nil
	}
	body, err := json.Marshal(map[string]any{
		"from":  minterAddress,
		"to":    eventBundleBurnAddress,
		"value": amount,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, transactionChainBaseURL()+"/transfer", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("burn NRN via transaction chain: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("transaction chain rejected NRN burn (%d): %s", resp.StatusCode, string(responseBody))
	}
	return nil
}

// eventBundleResourceRef is a lightweight reference to one Skill, Capability
// (MCP), or Asset involved in the event. It intentionally mirrors the CLI's
// dve.ResourceRef wire shape (KNIRVCHAIN cannot import the CLI's Go module —
// no cross-package imports per AGENTS.md/CLAUDE.md conventions — so this is
// an independent, structurally-identical type, not a shared one).
type eventBundleResourceRef struct {
	ID  string `json:"id"`
	Ref string `json:"ref,omitempty"`
}

// eventBundleCredentialRef references a credential used by the event. It is
// a name/provider/scope/key-ID pointer only — never a plaintext secret or
// ciphertext value, matching the CLI's dveartifact.CredentialEnvelope
// precedent (chain_refactor.md Design Invariant #2).
type eventBundleCredentialRef struct {
	Name     string   `json:"name"`
	Provider string   `json:"provider,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	KeyID    string   `json:"key_id,omitempty"`
}

// eventBundleContext captures the execution context the event occurred
// under (e.g. a usage-pattern signature), promoted to a real minted field
// instead of the previously stub-only ContextRecord persistence.
type eventBundleContext struct {
	ID               string `json:"id,omitempty"`
	PatternSignature string `json:"pattern_signature,omitempty"`
}

// eventBundlePolicy snapshots the policy the event was evaluated under
// (chain_refactor.md Open Decision #4: "Add Policy to the bundle schema"),
// mirroring the CLI's dveevidence.Policy shape.
type eventBundlePolicy struct {
	AllowedCommands    []string `json:"allowed_commands,omitempty"`
	DeniedCommands     []string `json:"denied_commands,omitempty"`
	AllowedCredentials []string `json:"allowed_credentials,omitempty"`
	AllowNetwork       bool     `json:"allow_network"`
	PolicyHash         string   `json:"policy_hash,omitempty"`
}

// EventBundleNFT is the full minted record.
type EventBundleNFT struct {
	SchemaVersion string                     `json:"schema_version"`
	EventID       string                     `json:"event_id"`
	SessionID     string                     `json:"session_id"`
	ProjectID     string                     `json:"project_id"`
	EventKind     string                     `json:"event_kind"` // "decision" | "error"
	Skills        []eventBundleResourceRef   `json:"skills,omitempty"`
	Capabilities  []eventBundleResourceRef   `json:"capabilities,omitempty"`
	Assets        []eventBundleResourceRef   `json:"assets,omitempty"`
	Context       *eventBundleContext        `json:"context,omitempty"`
	Credentials   []eventBundleCredentialRef `json:"credentials,omitempty"`
	Policy        eventBundlePolicy          `json:"policy"`
	MinterAddress string                     `json:"minter_address"`
	CreatedAt     time.Time                  `json:"created_at"`
}

type eventBundleMintRequest struct {
	SchemaVersion string                     `json:"schema_version"`
	EventID       string                     `json:"event_id"`
	SessionID     string                     `json:"session_id"`
	ProjectID     string                     `json:"project_id"`
	EventKind     string                     `json:"event_kind"`
	Skills        []eventBundleResourceRef   `json:"skills,omitempty"`
	Capabilities  []eventBundleResourceRef   `json:"capabilities,omitempty"`
	Assets        []eventBundleResourceRef   `json:"assets,omitempty"`
	Context       *eventBundleContext        `json:"context,omitempty"`
	Credentials   []eventBundleCredentialRef `json:"credentials,omitempty"`
	Policy        eventBundlePolicy          `json:"policy"`
	MinterAddress string                     `json:"minter_address"`
}

type eventBundleRecord struct {
	EventID       string    `json:"event_id"`
	ProjectID     string    `json:"project_id"`
	SessionID     string    `json:"session_id"`
	BundleHash    string    `json:"bundle_hash"`
	TransactionID string    `json:"transaction_id"`
	Bundle        EventBundleNFT `json:"bundle"`
	AcceptedAt    time.Time `json:"accepted_at"`
}

type eventBundleReceipt struct {
	SchemaVersion string    `json:"schema_version"`
	EventID       string    `json:"event_id"`
	BundleHash    string    `json:"bundle_hash"`
	TokenID       string    `json:"token_id"`
	TransactionID string    `json:"transaction_id"`
	BlockID       string    `json:"block_id,omitempty"`
	Final         bool      `json:"final"`
	FinalizedAt   time.Time `json:"finalized_at,omitempty"`
}

var eventKindValues = map[string]struct{}{"decision": {}, "error": {}}

func validateEventBundleMint(request eventBundleMintRequest) error {
	if request.SchemaVersion != eventBundleMintSchema {
		return fmt.Errorf("unsupported event bundle schema")
	}
	if !validationProofSafeID.MatchString(request.EventID) ||
		!validationProofSafeID.MatchString(request.SessionID) ||
		!validationProofSafeID.MatchString(request.ProjectID) {
		return fmt.Errorf("invalid event, session, or project id")
	}
	if _, ok := eventKindValues[request.EventKind]; !ok {
		return fmt.Errorf("invalid event_kind")
	}
	if request.MinterAddress == "" {
		return fmt.Errorf("minter_address is required")
	}
	for _, list := range [][]eventBundleResourceRef{request.Skills, request.Capabilities, request.Assets} {
		for _, ref := range list {
			if strings.TrimSpace(ref.ID) == "" {
				return fmt.Errorf("resource reference id must not be empty")
			}
		}
	}
	for _, cred := range request.Credentials {
		if strings.TrimSpace(cred.Name) == "" || strings.TrimSpace(cred.KeyID) == "" {
			return fmt.Errorf("credential reference must have a name and key id")
		}
	}
	if len(request.Skills) == 0 && len(request.Capabilities) == 0 && len(request.Assets) == 0 &&
		request.Context == nil && len(request.Credentials) == 0 {
		return fmt.Errorf("event bundle must reference at least one resource or context")
	}
	return nil
}

func bundleHash(bundle EventBundleNFT) (string, error) {
	canonical, err := json.Marshal(bundle)
	if err != nil {
		return "", err
	}
	return sha256Digest(canonical), nil
}

// handleEventBundleMint mints one EventBundleNFT transaction. Auth mirrors
// handleValidationProofMint's internal-token pattern — this is a
// service-to-service endpoint (CLI → KNIRVGATEWAY proxy → here), never
// directly exposed to end users.
func (bcs *BlockchainServer) handleEventBundleMint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "event_bundle_mint") {
		return
	}
	expectedToken := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	providedToken := strings.TrimSpace(r.Header.Get("X-KNIRV-Internal-Token"))
	if expectedToken == "" {
		http.Error(w, "internal event-bundle minting is not configured", http.StatusServiceUnavailable)
		return
	}
	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(providedToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request eventBundleMintRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid event bundle: "+err.Error(), http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "event bundle must contain one JSON value", http.StatusBadRequest)
		return
	}
	if err := validateEventBundleMint(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bcs.validationProofMu.Lock()
	defer bcs.validationProofMu.Unlock()

	if existing, exists, err := bcs.eventBundleRecordByEventID(request.EventID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		bcs.writeEventBundleReceipt(w, existing)
		return
	}

	bundle := EventBundleNFT{
		SchemaVersion: eventBundleMintSchema,
		EventID:       request.EventID,
		SessionID:     request.SessionID,
		ProjectID:     request.ProjectID,
		EventKind:     request.EventKind,
		Skills:        request.Skills,
		Capabilities:  request.Capabilities,
		Assets:        request.Assets,
		Context:       request.Context,
		Credentials:   request.Credentials,
		Policy:        request.Policy,
		MinterAddress: request.MinterAddress,
		CreatedAt:     time.Now().UTC(),
	}
	hash, err := bundleHash(bundle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	envelope := struct {
		SchemaVersion string         `json:"schema_version"`
		Bundle        EventBundleNFT `json:"bundle"`
	}{SchemaVersion: eventBundleMintSchema, Bundle: bundle}
	data, err := json.Marshal(envelope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mintCost := eventBundleMintCostNRN()
	if err := burnNRNForBundleMint(request.MinterAddress, mintCost); err != nil {
		http.Error(w, "burn NRN for bundle mint: "+err.Error(), http.StatusPaymentRequired)
		return
	}

	transaction := NewTransaction(request.MinterAddress, hash, 0, data)
	transaction.Type = TransactionTypeEventBundleMint
	if err := bcs.BlockchainPtr.AddTransactionToTransactionPool(transaction); err != nil {
		http.Error(w, "add event bundle transaction: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	record := &eventBundleRecord{
		EventID: request.EventID, ProjectID: request.ProjectID, SessionID: request.SessionID,
		BundleHash: hash, TransactionID: transaction.TransactionHash, Bundle: bundle,
		AcceptedAt: time.Now().UTC(),
	}
	if err := bcs.persistEventBundleRecord(record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bcs.BlockchainPtr.BroadcastTransaction(transaction)
	bcs.writeEventBundleReceipt(w, record)
}

// handleEventBundleGet serves GET /api/v1/event-bundles/{event_id} so a
// verifier (e.g. KNIRVSERVER's native_verifier) can independently re-fetch
// and re-hash a bundle by ID before certifying a commit that references it
// (chain_refactor.md Open Decision #2). No internal-auth token is required
// — this is a read of already-minted, non-sensitive data — but it is only
// ever reachable through the same gateway proxy path as the mint endpoint.
func (bcs *BlockchainServer) handleEventBundleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	if !validationProofSafeID.MatchString(eventID) {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	record, exists, err := bcs.eventBundleRecordByEventID(eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "event bundle not found", http.StatusNotFound)
		return
	}
	bcs.writeEventBundleReceipt(w, record)
}

func (bc *BlockchainStruct) validateEventBundleMintsInBlock(block *Block) error {
	seenEvents := make(map[string]string) // eventID -> bundleHash
	for _, existingBlock := range bc.Blocks {
		for _, transaction := range existingBlock.Transactions {
			bundle, ok := decodeEventBundleMint(transaction.Data)
			if !ok {
				continue
			}
			seenEvents[bundle.EventID] = transaction.To
		}
	}
	for _, transaction := range block.Transactions {
		bundle, ok := decodeEventBundleMint(transaction.Data)
		if !ok {
			continue
		}
		hash, err := bundleHash(bundle)
		if err != nil {
			return fmt.Errorf("recompute event bundle hash: %w", err)
		}
		if transaction.To != hash {
			return fmt.Errorf("event bundle transaction hash mismatch")
		}
		if existing, seen := seenEvents[bundle.EventID]; seen && existing != hash {
			return fmt.Errorf("event %s already has a different minted bundle", bundle.EventID)
		}
		seenEvents[bundle.EventID] = hash
	}
	return nil
}

func decodeEventBundleMint(data []byte) (EventBundleNFT, bool) {
	var envelope struct {
		SchemaVersion string         `json:"schema_version"`
		Bundle        EventBundleNFT `json:"bundle"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || envelope.SchemaVersion != eventBundleMintSchema {
		return EventBundleNFT{}, false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return EventBundleNFT{}, false
	}
	return envelope.Bundle, true
}

func (bcs *BlockchainServer) eventBundleRecordByEventID(eventID string) (*eventBundleRecord, bool, error) {
	key := eventBundleEventKeyPrefix + eventID
	exists, err := bcs.db.KeyExists(key)
	if err != nil || !exists {
		return nil, false, err
	}
	raw, err := bcs.db.GetBytes(key)
	if err != nil {
		return nil, false, err
	}
	var record eventBundleRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func (bcs *BlockchainServer) persistEventBundleRecord(record *eventBundleRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	eventKey := eventBundleEventKeyPrefix + record.EventID
	hashKey := eventBundleKeyPrefix + strings.TrimPrefix(record.BundleHash, "sha256:")
	if err := bcs.db.PutBytes(eventKey, raw); err != nil {
		return err
	}
	return bcs.db.PutBytes(hashKey, raw)
}

func (bcs *BlockchainServer) writeEventBundleReceipt(w http.ResponseWriter, record *eventBundleRecord) {
	receipt := eventBundleReceipt{
		SchemaVersion: eventBundleReceiptSchema,
		EventID:       record.EventID,
		BundleHash:    record.BundleHash,
		TokenID:       record.BundleHash, // the bundle hash doubles as the token id: content-addressed, one bundle per event
		TransactionID: record.TransactionID,
	}
	bcs.BlockchainPtr.Lock()
	for _, block := range bcs.BlockchainPtr.Blocks {
		for _, transaction := range block.Transactions {
			if transaction.TransactionHash == record.TransactionID {
				receipt.Final = true
				receipt.BlockID = hex.EncodeToString(block.BlockHash)
				receipt.FinalizedAt = time.Unix(block.Timestamp, 0).UTC()
				break
			}
		}
		if receipt.Final {
			break
		}
	}
	bcs.BlockchainPtr.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if receipt.Final {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
	_ = json.NewEncoder(w).Encode(receipt)
}
