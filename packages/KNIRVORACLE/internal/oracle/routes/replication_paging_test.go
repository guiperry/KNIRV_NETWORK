package routes

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/guiperry/knirv-sdk-go/replication"
	signing "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/governance"
	"github.com/knirvcorp/knirvoracle/internal/oracle/ledger"
	"github.com/knirvcorp/knirvoracle/internal/oracle/mmr"
	"go.uber.org/zap"
)

func TestReplicationCheckpointsRejectsUnboundedAllocation(t *testing.T) {
	router := NewOracleRoutes(newTestOracleWithMode(t, oracle.ModeActive), zap.NewNop())
	for _, limit := range []string{"-1", "0", "999999999", "not-a-number", "1"} {
		t.Run(limit, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/oracle/v3/replication/checkpoints?chain=unknown&limit="+limit, nil)
			router.handleReplicationCheckpoints(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestReplicationClientRebuildsSignedRoots(t *testing.T) {
	o := newTestOracleWithMode(t, oracle.ModeActive)
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	stake := o.GetGovernanceSystem().GetNetworkParameters().MinValidatorStake
	if _, err := o.GetGovernanceSystem().RegisterValidator(&governance.ValidatorRegistration{Address: key.Address, PubKey: []byte{2, 4}, StakedAmount: new(big.Int).Add(stake, big.NewInt(1)), CommissionRate: 0.1}); err != nil {
		t.Fatal(err)
	}
	raw, err := hex.DecodeString(key.PrivateKeyHex())
	if err != nil {
		t.Fatal(err)
	}
	signer, err := signing.NewLocalSigner(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.Replication().ProduceHeadIfChanged(); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewOracleRoutes(o, zap.NewNop()).RegisterRoutes(mux)
	get := func(path string, authenticated bool, out any) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if authenticated {
			pull, err := replication.SignPull(signer, replication.PullRequest{ChainID: o.ChainID(), Nonce: fmt.Sprintf("acceptance-%s", path), Timestamp: time.Now().Unix()})
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(pull)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set(replicationAuthHeader, base64.StdEncoding.EncodeToString(body))
		}
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, res.Code, res.Body.String())
		}
		if err := json.Unmarshal(res.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
	var signed replication.SignedHead
	get("/oracle/v3/replication/head", false, &signed)
	head, err := replication.VerifyHead(signed, signed.Head.OwnerAddress, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var leaves struct {
		Leaves []string `json:"leaves"`
	}
	get("/oracle/v3/replication/leaves?from=0&limit=1024", false, &leaves)
	tree := mmr.New()
	for _, leaf := range leaves.Leaves {
		b, err := hex.DecodeString(leaf)
		if err != nil || len(b) != 32 {
			t.Fatalf("invalid leaf %q", leaf)
		}
		var hash mmr.Hash
		copy(hash[:], b)
		tree.AddRaw(hash)
	}
	root := tree.BagRoot()
	if tree.Size() != head.MMRSize || hex.EncodeToString(root[:]) != head.MMRRoot {
		t.Fatalf("MMR mismatch: size=%d root=%x head=%+v", tree.Size(), root, head)
	}
	var state struct {
		Records []ledger.Record `json:"records"`
	}
	get("/oracle/v3/replication/state?from=0&limit=256", true, &state)
	if uint64(len(state.Records)) != head.StateLogSize {
		t.Fatalf("state length=%d head=%d", len(state.Records), head.StateLogSize)
	}
	replayed, err := ledger.Replay(state.Records)
	if err != nil {
		t.Fatal(err)
	}
	stateRoot, err := ledger.StateRoot(replayed)
	if err != nil {
		t.Fatal(err)
	}
	if stateRoot != head.StateRoot {
		t.Fatalf("state root=%s signed=%s", stateRoot, head.StateRoot)
	}
}
