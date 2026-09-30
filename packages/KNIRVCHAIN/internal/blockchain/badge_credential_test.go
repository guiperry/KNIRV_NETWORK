package blockchain

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"KNIRVCHAIN/internal/database"

	"github.com/gorilla/mux"
)

func newBadgeCredentialServer(t *testing.T) *BlockchainServer {
	t.Helper()
	db, err := database.NewLevelDB(filepath.Join(t.TempDir(), "chain.db"))
	if err != nil {
		t.Fatalf("open leveldb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &BlockchainServer{db: db, testMode: true, BlockchainPtr: &BlockchainStruct{}}
}

func badgeCredentialRouter(bcs *BlockchainServer) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/badge-credentials/mint", bcs.handleBadgeCredentialMint)
	r.HandleFunc("/api/v1/badge-credentials/revoke", bcs.handleBadgeCredentialRevoke)
	r.HandleFunc("/api/v1/badge-credentials/{credential_id}", bcs.handleBadgeCredentialGet).Methods(http.MethodGet)
	return r
}

const hexA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const hexB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func earnedMint(cert string) map[string]any {
	return map[string]any{
		"schema_version": badgeCredentialMintSchema, "credential_id": "cred-1",
		"badge_id": "badge-leak", "badge_name": "Leak Fixer", "skill_id": "skill-1",
		"error_node_id": "err-1", "holder_agent_id": "dve-abc123", "owner_address": "knirv1owner",
		"kind": "earned", "execution_mode": "agent", "suite_hash": hexA,
		"certificate_hash": cert, "issued_at": time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}
}

func post(t *testing.T, h http.Handler, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	if token != "" {
		req.Header.Set("X-KNIRV-Internal-Token", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestBadgeCredentialMintAuthAndValidation(t *testing.T) {
	bcs := newBadgeCredentialServer(t)
	h := badgeCredentialRouter(bcs)

	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "")
	if rec := post(t, h, "/api/v1/badge-credentials/mint", "x", earnedMint(hexB)); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured token must fail closed, got %d", rec.Code)
	}
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	if rec := post(t, h, "/api/v1/badge-credentials/mint", "wrong", earnedMint(hexB)); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token must be rejected, got %d", rec.Code)
	}

	bad := earnedMint(hexB)
	bad["execution_mode"] = "foundation_model"
	if rec := post(t, h, "/api/v1/badge-credentials/mint", "secret", bad); rec.Code != http.StatusBadRequest {
		t.Fatalf("an earned credential claiming foundation execution must be rejected, got %d", rec.Code)
	}
	bad = earnedMint(hexB)
	delete(bad, "suite_hash")
	if rec := post(t, h, "/api/v1/badge-credentials/mint", "secret", bad); rec.Code != http.StatusBadRequest {
		t.Fatalf("an earned credential without its suite hash must be rejected, got %d", rec.Code)
	}
	if len(bcs.BlockchainPtr.TransactionPool) != 0 {
		t.Fatal("rejected mints must not reach the pool")
	}
}

func TestBadgeCredentialMintRevokeAndFinality(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	bcs := newBadgeCredentialServer(t)
	h := badgeCredentialRouter(bcs)

	rec := post(t, h, "/api/v1/badge-credentials/mint", "secret", earnedMint(hexB))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("mint: %d %s", rec.Code, rec.Body.String())
	}
	var receipt badgeCredentialReceipt
	_ = json.Unmarshal(rec.Body.Bytes(), &receipt)
	if !strings.HasPrefix(receipt.TokenID, "sha256:") || receipt.Final || receipt.TransactionID == "" {
		t.Fatalf("unexpected pending receipt: %+v", receipt)
	}
	pool := bcs.BlockchainPtr.TransactionPool
	if len(pool) != 1 || pool[0].Type != txTypeBadgeCredentialMint || pool[0].To != receipt.TokenID {
		t.Fatalf("expected one mint transaction addressed to the token: %+v", pool)
	}
	// The mint is a protocol transaction the chain itself accepts as valid.
	if ok, err := pool[0].VerifySignature(); !ok || err != nil {
		t.Fatalf("mint transaction must pass protocol verification: %v %v", ok, err)
	}

	// Same certificate → replayed receipt, no second transaction.
	rec = post(t, h, "/api/v1/badge-credentials/mint", "secret", earnedMint(hexB))
	var replay badgeCredentialReceipt
	_ = json.Unmarshal(rec.Body.Bytes(), &replay)
	if replay.TokenID != receipt.TokenID || len(bcs.BlockchainPtr.TransactionPool) != 1 {
		t.Fatalf("a repeated mint must replay, got %+v with pool %d", replay, len(bcs.BlockchainPtr.TransactionPool))
	}

	// Once the transaction is in a block, the receipt is final.
	block := &Block{BlockHash: []byte{1, 2, 3}, Timestamp: time.Now().Unix(), Transactions: []*Transaction{pool[0]}}
	if err := bcs.BlockchainPtr.validateBadgeCredentialsInBlock(block); err != nil {
		t.Fatalf("block with a genuine mint must validate: %v", err)
	}
	bcs.BlockchainPtr.Blocks = append(bcs.BlockchainPtr.Blocks, block)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/api/v1/badge-credentials/cred-1", nil))
	var got badgeCredentialReceipt
	_ = json.Unmarshal(getRec.Body.Bytes(), &got)
	if getRec.Code != http.StatusOK || !got.Final || got.Revoked {
		t.Fatalf("expected a final, active credential: %d %+v", getRec.Code, got)
	}

	rec = post(t, h, "/api/v1/badge-credentials/revoke", "secret", map[string]any{
		"schema_version": badgeCredentialRevokeSchema, "credential_id": "cred-1", "reason": "audit",
	})
	var revoked badgeCredentialReceipt
	_ = json.Unmarshal(rec.Body.Bytes(), &revoked)
	if !revoked.Revoked || revoked.RevokeTransactionID == "" {
		t.Fatalf("expected a revoked receipt: %d %+v", rec.Code, revoked)
	}
	revokeTx := bcs.BlockchainPtr.TransactionPool[len(bcs.BlockchainPtr.TransactionPool)-1]
	if err := bcs.BlockchainPtr.validateBadgeCredentialsInBlock(&Block{Transactions: []*Transaction{revokeTx}}); err != nil {
		t.Fatalf("revoking a minted credential must validate: %v", err)
	}
}

func TestBadgeCredentialBlockValidationRejectsTampering(t *testing.T) {
	t.Setenv("KNIRV_INTERNAL_AUTH_TOKEN", "secret")
	bcs := newBadgeCredentialServer(t)
	h := badgeCredentialRouter(bcs)
	post(t, h, "/api/v1/badge-credentials/mint", "secret", earnedMint(hexB))
	tx := *bcs.BlockchainPtr.TransactionPool[0]

	// Claims rewritten after minting no longer hash to the token id.
	var nft BadgeCredentialNFT
	_ = json.Unmarshal(tx.Data, &nft)
	nft.Kind, nft.ExecutionMode = "purchased", "foundation_model"
	tx.Data, _ = json.Marshal(nft)
	if err := bcs.BlockchainPtr.validateBadgeCredentialsInBlock(&Block{Transactions: []*Transaction{&tx}}); err == nil {
		t.Fatal("a mint whose content does not hash to its token must be rejected")
	}

	// A revocation for a token that was never minted is rejected.
	orphan := NewTransaction(bcs.BlockchainPtr.TransactionPool[0].From, "sha256:"+hexA, 0,
		[]byte(`{"schema_version":"`+badgeCredentialRevokeSchema+`","credential_id":"cred-x","token_id":"sha256:`+hexA+`","reason":"r","revoked_at":"2026-09-30T00:00:00Z"}`))
	orphan.Type = txTypeBadgeCredentialRevoke
	if err := bcs.BlockchainPtr.validateBadgeCredentialsInBlock(&Block{Transactions: []*Transaction{orphan}}); err == nil {
		t.Fatal("revoking an unminted credential must be rejected")
	}
}
