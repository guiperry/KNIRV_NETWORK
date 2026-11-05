package routes

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"go.uber.org/zap"
)

func TestReplicationRoutesHeadAndPullAuth(t *testing.T) {
	instance := newTestOracleWithMode(t, oracle.ModeActive)
	router := NewOracleRoutes(instance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	// A fresh active Oracle produced a head at startup.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oracle/v3/replication/head", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("head status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var signed replication.SignedHead
	if err := json.Unmarshal(rec.Body.Bytes(), &signed); err != nil {
		t.Fatalf("decode head: %v", err)
	}
	if signed.Head.Seq != 1 {
		t.Fatalf("first head seq = %d, want 1", signed.Head.Seq)
	}
	if signed.Head.StateRoot != instance.StateRoot() {
		t.Fatalf("head state root = %s, oracle root = %s", signed.Head.StateRoot, instance.StateRoot())
	}

	// R5 is not public.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oracle/v3/replication/state", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated state status = %d, want 401", rec.Code)
	}

	// Register the pull signer as a validator, then pull with a signed request.
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("validator key: %v", err)
	}
	minStake := instance.GetGovernanceSystem().GetNetworkParameters().MinValidatorStake
	if _, err := instance.GetGovernanceSystem().RegisterValidator(&governance.ValidatorRegistration{
		Address:        key.Address,
		PubKey:         []byte{0x02, 0x04},
		StakedAmount:   new(big.Int).Add(minStake, big.NewInt(1)),
		CommissionRate: 0.1,
	}); err != nil {
		t.Fatalf("register validator: %v", err)
	}
	rawKey, err := hex.DecodeString(key.PrivateKeyHex())
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}
	signer, err := knirvsigning.NewLocalSigner(rawKey)
	if err != nil {
		t.Fatalf("local signer: %v", err)
	}
	pull, err := replication.SignPull(signer, replication.PullRequest{
		ChainID: instance.ChainID(), HeadSeqSeen: 1, Nonce: "test-nonce-1", Timestamp: time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("sign pull: %v", err)
	}
	body, err := json.Marshal(pull)
	if err != nil {
		t.Fatalf("encode pull: %v", err)
	}

	authorized := httptest.NewRequest(http.MethodGet, "/oracle/v3/replication/state?from=0&limit=10", nil)
	authorized.Header.Set(replicationAuthHeader, base64.StdEncoding.EncodeToString(body))
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, authorized)
	if rec.Code != http.StatusOK {
		t.Fatalf("authorized state status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var state struct {
		Records []map[string]any `json:"records"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if len(state.Records) == 0 || state.Records[0]["kind"] != "genesis" {
		t.Fatalf("state records = %+v, want genesis first", state.Records)
	}

	// The same nonce cannot be replayed.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, authorized.Clone(authorized.Context()))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed pull status = %d, want 401", rec.Code)
	}

	// Head paging.
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oracle/v3/replication/heads?after_seq=0", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("heads status = %d", rec.Code)
	}
	var heads struct {
		Heads []replication.SignedHead `json:"heads"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &heads); err != nil {
		t.Fatalf("decode heads: %v", err)
	}
	if len(heads.Heads) != 1 {
		t.Fatalf("heads = %d, want 1", len(heads.Heads))
	}
}

func TestReplicationRejectsNonValidatorPull(t *testing.T) {
	instance := newTestOracleWithMode(t, oracle.ModeActive)
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(key.PrivateKeyHex())
	if err != nil {
		t.Fatal(err)
	}
	signer, err := knirvsigning.NewLocalSigner(raw)
	if err != nil {
		t.Fatal(err)
	}
	pull, err := replication.SignPull(signer, replication.PullRequest{ChainID: instance.ChainID(), Nonce: "unregistered-pull", Timestamp: time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(pull)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/oracle/v3/replication/state", nil)
	req.Header.Set(replicationAuthHeader, base64.StdEncoding.EncodeToString(body))
	res := httptest.NewRecorder()
	NewOracleRoutes(instance, zap.NewNop()).handleReplicationState(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("non-validator status=%d body=%s", res.Code, res.Body.String())
	}
}
