package blockchain

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"KNIRVCHAIN/internal/utils"

	"github.com/gorilla/mux"
)

// Badge credential NFTs: KNIRVCHAIN's badge system, made real.
//
// KNIRVCHAIN's badge model (internal/agent: a skill-typed Badge attached to an
// Agent through a signed BadgeAttachment carrying a tx hash) never reached
// production — internal/agent is imported only by tests, it had no route, and
// its "on-chain record" was a sha256 labelled a mock tx hash. This file is that
// model on the production path: one NFT per issued skill-badge credential,
// attaching a skill badge to the agent (Agent Workspace) that holds it, recorded
// as a real protocol transaction.
//
// Issuer: KNIRVSERVER's badgecredential service, when an agent earns a badge
// (passes the 8 tests on the skill's KNIRVGRAPH error node), buys it, or is
// assigned it. Authorisation is the shared internal service token at this
// handler — the same custody model as validation-proof mints — so the chain
// records a protocol transaction (From = BLOCKCHAIN_ADDRESS) rather than a
// user-signed one: the credential is attested by the network, not self-issued.
//
// Source of truth is this server's LevelDB records plus the chain itself, not
// the ChromemDB badge collections, whose lookups are embedding-similarity
// queries and cannot answer "does this agent hold this badge" exactly.
const (
	badgeCredentialMintSchema    = "knirv.badge-credential.v1"
	badgeCredentialRevokeSchema  = "knirv.badge-credential-revoke.v1"
	badgeCredentialReceiptSchema = "badge_credential_receipt.v1"

	badgeCredentialTokenKeyPrefix = "badge_credential:token:"
	badgeCredentialCertKeyPrefix  = "badge_credential:cert:"
	badgeCredentialIDKeyPrefix    = "badge_credential:credential:"

	txTypeBadgeCredentialMint   = "protocol_badge_credential_mint"
	txTypeBadgeCredentialRevoke = "protocol_badge_credential_revoke"
)

var (
	badgeCredentialSafeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	sha256Hex             = regexp.MustCompile(`^[0-9a-f]{64}$`)
	badgeCredentialKinds  = map[string]string{"earned": "agent", "assigned": "agent", "purchased": "foundation_model"}
)

// BadgeCredentialNFT is the minted token. BadgeType is always "skill": the
// badge system's skill badges are exactly the credentials KNIRVSERVER issues.
type BadgeCredentialNFT struct {
	SchemaVersion string `json:"schema_version"`
	CredentialID  string `json:"credential_id"`
	BadgeID       string `json:"badge_id"`
	BadgeName     string `json:"badge_name"`
	BadgeType     string `json:"badge_type"`
	SkillID       string `json:"skill_id,omitempty"`
	ErrorNodeID   string `json:"error_node_id,omitempty"`
	// HolderAgentID is the agent the badge is attached to — the Agent
	// Workspace's DVE node ID (BadgeAttachment.AgentId in the badge model).
	HolderAgentID string `json:"holder_agent_id"`
	OwnerAddress  string `json:"owner_address,omitempty"`
	Kind          string `json:"kind"`
	ExecutionMode string `json:"execution_mode"`
	// SuiteHash pins an earned credential to the exact 8-test suite passed.
	SuiteHash       string    `json:"suite_hash,omitempty"`
	CertificateHash string    `json:"certificate_hash"`
	IssuedAt        time.Time `json:"issued_at"`
}

type badgeCredentialMintRequest struct {
	SchemaVersion   string    `json:"schema_version"`
	CredentialID    string    `json:"credential_id"`
	BadgeID         string    `json:"badge_id"`
	BadgeName       string    `json:"badge_name"`
	SkillID         string    `json:"skill_id,omitempty"`
	ErrorNodeID     string    `json:"error_node_id,omitempty"`
	HolderAgentID   string    `json:"holder_agent_id"`
	OwnerAddress    string    `json:"owner_address,omitempty"`
	Kind            string    `json:"kind"`
	ExecutionMode   string    `json:"execution_mode"`
	SuiteHash       string    `json:"suite_hash,omitempty"`
	CertificateHash string    `json:"certificate_hash"`
	IssuedAt        time.Time `json:"issued_at"`
}

type badgeCredentialRevokeRequest struct {
	SchemaVersion string `json:"schema_version"`
	CredentialID  string `json:"credential_id"`
	Reason        string `json:"reason"`
}

// badgeCredentialRevocation is the on-chain revocation payload.
type badgeCredentialRevocation struct {
	SchemaVersion string    `json:"schema_version"`
	CredentialID  string    `json:"credential_id"`
	TokenID       string    `json:"token_id"`
	Reason        string    `json:"reason"`
	RevokedAt     time.Time `json:"revoked_at"`
}

type badgeCredentialRecord struct {
	TokenID             string             `json:"token_id"`
	TransactionID       string             `json:"transaction_id"`
	NFT                 BadgeCredentialNFT `json:"nft"`
	AcceptedAt          time.Time          `json:"accepted_at"`
	RevokedAt           *time.Time         `json:"revoked_at,omitempty"`
	RevokeReason        string             `json:"revoke_reason,omitempty"`
	RevokeTransactionID string             `json:"revoke_transaction_id,omitempty"`
}

type badgeCredentialReceipt struct {
	SchemaVersion       string     `json:"schema_version"`
	CredentialID        string     `json:"credential_id"`
	BadgeID             string     `json:"badge_id"`
	HolderAgentID       string     `json:"holder_agent_id"`
	CertificateHash     string     `json:"certificate_hash"`
	TokenID             string     `json:"token_id"`
	TransactionID       string     `json:"transaction_id"`
	BlockID             string     `json:"block_id,omitempty"`
	Final               bool       `json:"final"`
	FinalizedAt         *time.Time `json:"finalized_at,omitempty"`
	Revoked             bool       `json:"revoked"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty"`
	RevokeTransactionID string     `json:"revoke_transaction_id,omitempty"`
}

func validateBadgeCredentialMint(r badgeCredentialMintRequest) error {
	if r.SchemaVersion != badgeCredentialMintSchema {
		return fmt.Errorf("unsupported badge credential schema")
	}
	for label, id := range map[string]string{"credential_id": r.CredentialID, "badge_id": r.BadgeID, "holder_agent_id": r.HolderAgentID} {
		if !badgeCredentialSafeID.MatchString(id) {
			return fmt.Errorf("invalid %s", label)
		}
	}
	if strings.TrimSpace(r.BadgeName) == "" {
		return fmt.Errorf("badge_name is required")
	}
	mode, ok := badgeCredentialKinds[r.Kind]
	if !ok {
		return fmt.Errorf("unsupported credential kind %q", r.Kind)
	}
	if r.ExecutionMode != mode {
		return fmt.Errorf("a %s credential executes as %q, not %q", r.Kind, mode, r.ExecutionMode)
	}
	if r.Kind == "earned" && !sha256Hex.MatchString(r.SuiteHash) {
		return fmt.Errorf("an earned credential must pin the suite_hash it passed")
	}
	if !sha256Hex.MatchString(r.CertificateHash) {
		return fmt.Errorf("certificate_hash must be a sha256 hex digest")
	}
	if r.IssuedAt.IsZero() {
		return fmt.Errorf("issued_at is required")
	}
	return nil
}

func badgeCredentialTokenID(nft BadgeCredentialNFT) (string, error) {
	raw, err := json.Marshal(nft)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func requireInternalToken(w http.ResponseWriter, r *http.Request, what string) bool {
	expected := strings.TrimSpace(os.Getenv("KNIRV_INTERNAL_AUTH_TOKEN"))
	if expected == "" {
		http.Error(w, "internal "+what+" is not configured", http.StatusServiceUnavailable)
		return false
	}
	provided := strings.TrimSpace(r.Header.Get("X-KNIRV-Internal-Token"))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func decodeSingleJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("body must contain one JSON value")
	}
	return nil
}

// handleBadgeCredentialMint serves POST /api/v1/badge-credentials/mint.
// Idempotent per certificate hash: a credential whose claims change (e.g. a
// purchased badge later earned) gets a new certificate and a new token; the
// credential's latest token supersedes the earlier one.
func (bcs *BlockchainServer) handleBadgeCredentialMint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if bcs.checkNetworkPauseAndRejectIfPaused(w, "badge_credential_mint") {
		return
	}
	if !requireInternalToken(w, r, "badge credential minting") {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var request badgeCredentialMintRequest
	if err := decodeSingleJSON(r, &request); err != nil {
		http.Error(w, "invalid badge credential: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateBadgeCredentialMint(request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bcs.validationProofMu.Lock()
	defer bcs.validationProofMu.Unlock()

	if existing, ok, err := bcs.badgeCredentialByCertificate(request.CertificateHash); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if ok {
		bcs.writeBadgeCredentialReceipt(w, existing)
		return
	}

	nft := BadgeCredentialNFT{
		SchemaVersion: badgeCredentialMintSchema, CredentialID: request.CredentialID,
		BadgeID: request.BadgeID, BadgeName: request.BadgeName, BadgeType: "skill",
		SkillID: request.SkillID, ErrorNodeID: request.ErrorNodeID,
		HolderAgentID: request.HolderAgentID, OwnerAddress: request.OwnerAddress,
		Kind: request.Kind, ExecutionMode: request.ExecutionMode, SuiteHash: request.SuiteHash,
		CertificateHash: request.CertificateHash, IssuedAt: request.IssuedAt.UTC(),
	}
	tokenID, err := badgeCredentialTokenID(nft)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(nft)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	transaction := NewTransaction(utils.BLOCKCHAIN_ADDRESS, tokenID, 0, data)
	transaction.Type = txTypeBadgeCredentialMint
	record := &badgeCredentialRecord{
		TokenID: tokenID, TransactionID: transaction.TransactionHash, NFT: nft, AcceptedAt: time.Now().UTC(),
	}
	// Submit, then persist (validation-proof order). If persisting fails the
	// issuer retries and resubmits an identical, content-addressed transaction;
	// persisting first could instead strand a record that is never submitted.
	bcs.BlockchainPtr.addVerifiedTxnToPoolAndSignal(transaction)
	if err := bcs.persistBadgeCredentialRecord(record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bcs.BlockchainPtr.BroadcastTransaction(transaction)
	bcs.writeBadgeCredentialReceipt(w, record)
}

// handleBadgeCredentialRevoke serves POST /api/v1/badge-credentials/revoke.
func (bcs *BlockchainServer) handleBadgeCredentialRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !requireInternalToken(w, r, "badge credential revocation") {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var request badgeCredentialRevokeRequest
	if err := decodeSingleJSON(r, &request); err != nil || request.SchemaVersion != badgeCredentialRevokeSchema || !badgeCredentialSafeID.MatchString(request.CredentialID) {
		http.Error(w, "invalid badge credential revocation", http.StatusBadRequest)
		return
	}

	bcs.validationProofMu.Lock()
	defer bcs.validationProofMu.Unlock()

	record, ok, err := bcs.badgeCredentialByID(request.CredentialID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "badge credential not found", http.StatusNotFound)
		return
	}
	if record.RevokedAt != nil {
		bcs.writeBadgeCredentialReceipt(w, record)
		return
	}
	now := time.Now().UTC()
	data, err := json.Marshal(badgeCredentialRevocation{
		SchemaVersion: badgeCredentialRevokeSchema, CredentialID: request.CredentialID,
		TokenID: record.TokenID, Reason: request.Reason, RevokedAt: now,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	transaction := NewTransaction(utils.BLOCKCHAIN_ADDRESS, record.TokenID, 0, data)
	transaction.Type = txTypeBadgeCredentialRevoke
	record.RevokedAt, record.RevokeReason, record.RevokeTransactionID = &now, request.Reason, transaction.TransactionHash
	bcs.BlockchainPtr.addVerifiedTxnToPoolAndSignal(transaction)
	if err := bcs.persistBadgeCredentialRecord(record); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bcs.BlockchainPtr.BroadcastTransaction(transaction)
	bcs.writeBadgeCredentialReceipt(w, record)
}

// handleBadgeCredentialGet serves GET /api/v1/badge-credentials/{credential_id}:
// the credential's current token, finality and revocation state. Public read,
// like GET /api/v1/event-bundles/{event_id}.
func (bcs *BlockchainServer) handleBadgeCredentialGet(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["credential_id"]
	if !badgeCredentialSafeID.MatchString(id) {
		http.Error(w, "invalid credential id", http.StatusBadRequest)
		return
	}
	record, ok, err := bcs.badgeCredentialByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "badge credential not found", http.StatusNotFound)
		return
	}
	bcs.writeBadgeCredentialReceipt(w, record)
}

func (bcs *BlockchainServer) getBadgeCredentialRecord(key string) (*badgeCredentialRecord, bool, error) {
	exists, err := bcs.db.KeyExists(key)
	if err != nil || !exists {
		return nil, false, err
	}
	raw, err := bcs.db.GetBytes(key)
	if err != nil {
		return nil, false, err
	}
	var record badgeCredentialRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func (bcs *BlockchainServer) badgeCredentialByToken(tokenID string) (*badgeCredentialRecord, bool, error) {
	return bcs.getBadgeCredentialRecord(badgeCredentialTokenKeyPrefix + strings.TrimPrefix(tokenID, "sha256:"))
}

func (bcs *BlockchainServer) badgeCredentialPointer(key string) (*badgeCredentialRecord, bool, error) {
	exists, err := bcs.db.KeyExists(key)
	if err != nil || !exists {
		return nil, false, err
	}
	tokenID, err := bcs.db.GetBytes(key)
	if err != nil {
		return nil, false, err
	}
	return bcs.badgeCredentialByToken(string(tokenID))
}

func (bcs *BlockchainServer) badgeCredentialByCertificate(certHash string) (*badgeCredentialRecord, bool, error) {
	return bcs.badgeCredentialPointer(badgeCredentialCertKeyPrefix + certHash)
}

func (bcs *BlockchainServer) badgeCredentialByID(credentialID string) (*badgeCredentialRecord, bool, error) {
	return bcs.badgeCredentialPointer(badgeCredentialIDKeyPrefix + credentialID)
}

func (bcs *BlockchainServer) persistBadgeCredentialRecord(record *badgeCredentialRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := bcs.db.PutBytes(badgeCredentialTokenKeyPrefix+strings.TrimPrefix(record.TokenID, "sha256:"), raw); err != nil {
		return err
	}
	if err := bcs.db.PutBytes(badgeCredentialCertKeyPrefix+record.NFT.CertificateHash, []byte(record.TokenID)); err != nil {
		return err
	}
	// The credential's latest token supersedes any earlier one (an upgrade
	// from purchased to earned re-mints under a new certificate).
	return bcs.db.PutBytes(badgeCredentialIDKeyPrefix+record.NFT.CredentialID, []byte(record.TokenID))
}

// transactionFinality reports whether txID is in a block yet.
func (bcs *BlockchainServer) transactionFinality(txID string) (string, *time.Time) {
	if bcs.BlockchainPtr == nil || txID == "" {
		return "", nil
	}
	bcs.BlockchainPtr.Lock()
	defer bcs.BlockchainPtr.Unlock()
	for _, block := range bcs.BlockchainPtr.Blocks {
		for _, transaction := range block.Transactions {
			if transaction.TransactionHash == txID {
				at := time.Unix(block.Timestamp, 0).UTC()
				return hex.EncodeToString(block.BlockHash), &at
			}
		}
	}
	return "", nil
}

func (bcs *BlockchainServer) writeBadgeCredentialReceipt(w http.ResponseWriter, record *badgeCredentialRecord) {
	receipt := badgeCredentialReceipt{
		SchemaVersion: badgeCredentialReceiptSchema, CredentialID: record.NFT.CredentialID,
		BadgeID: record.NFT.BadgeID, HolderAgentID: record.NFT.HolderAgentID,
		CertificateHash: record.NFT.CertificateHash, TokenID: record.TokenID,
		TransactionID: record.TransactionID, Revoked: record.RevokedAt != nil,
		RevokedAt: record.RevokedAt, RevokeTransactionID: record.RevokeTransactionID,
	}
	receipt.BlockID, receipt.FinalizedAt = bcs.transactionFinality(record.TransactionID)
	receipt.Final = receipt.BlockID != ""
	w.Header().Set("Content-Type", "application/json")
	if receipt.Final {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
	_ = json.NewEncoder(w).Encode(receipt)
}

func decodeBadgeCredentialMint(data []byte) (BadgeCredentialNFT, bool) {
	var nft BadgeCredentialNFT
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&nft) != nil || nft.SchemaVersion != badgeCredentialMintSchema {
		return BadgeCredentialNFT{}, false
	}
	return nft, true
}

func decodeBadgeCredentialRevoke(data []byte) (badgeCredentialRevocation, bool) {
	var rev badgeCredentialRevocation
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&rev) != nil || rev.SchemaVersion != badgeCredentialRevokeSchema {
		return badgeCredentialRevocation{}, false
	}
	return rev, true
}

// validateBadgeCredentialsInBlock enforces, for every node accepting the
// block: a mint's token ID is the hash of its own content and it is a
// protocol transaction; a revocation names a token minted earlier (or earlier
// in the same block) for the same credential.
func (bc *BlockchainStruct) validateBadgeCredentialsInBlock(block *Block) error {
	minted := make(map[string]string) // token id -> credential id
	for _, existing := range bc.Blocks {
		for _, tx := range existing.Transactions {
			if tx.Type == txTypeBadgeCredentialMint {
				if nft, ok := decodeBadgeCredentialMint(tx.Data); ok {
					minted[tx.To] = nft.CredentialID
				}
			}
		}
	}
	for _, tx := range block.Transactions {
		switch tx.Type {
		case txTypeBadgeCredentialMint:
			nft, ok := decodeBadgeCredentialMint(tx.Data)
			if !ok {
				return fmt.Errorf("malformed badge credential mint")
			}
			tokenID, err := badgeCredentialTokenID(nft)
			if err != nil {
				return err
			}
			if tx.From != utils.BLOCKCHAIN_ADDRESS || tx.To != tokenID {
				return fmt.Errorf("badge credential mint identity mismatch")
			}
			minted[tokenID] = nft.CredentialID
		case txTypeBadgeCredentialRevoke:
			rev, ok := decodeBadgeCredentialRevoke(tx.Data)
			if !ok {
				return fmt.Errorf("malformed badge credential revocation")
			}
			if tx.From != utils.BLOCKCHAIN_ADDRESS || tx.To != rev.TokenID {
				return fmt.Errorf("badge credential revocation identity mismatch")
			}
			if credential, ok := minted[rev.TokenID]; !ok || credential != rev.CredentialID {
				return fmt.Errorf("revocation of unminted badge credential %s", rev.CredentialID)
			}
		}
	}
	return nil
}
