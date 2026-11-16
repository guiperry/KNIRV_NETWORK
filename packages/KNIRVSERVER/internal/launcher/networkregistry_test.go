package launcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
)

// workerRegisterContract is the unsigned POST /register contract of the
// REGISTRY.KNIRV.NETWORK worker (websites/REGISTRY.KNIRV.NETWORK/src/index.js,
// register()): a body carrying a txChainAddress is rejected with 400 because a
// tx-chain address requires a signed registry record; any other valid body is
// accepted with 201. The stub applies only that rule. It persists nothing, so the
// test checks what the launcher sends, not what a store holds.
func workerRegisterContract(t *testing.T) (*httptest.Server, *[][]byte) {
	t.Helper()
	var bodies [][]byte
	mux := http.NewServeMux()
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		raw, _ := io.ReadAll(io.LimitReader(r.Body, 64*1024))
		bodies = append(bodies, raw)
		var generic map[string]any
		if err := json.Unmarshal(raw, &generic); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if v, ok := generic["txChainAddress"]; ok && v != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"A tx-chain address requires a signed registry record."}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":201}`))
	})
	return httptest.NewServer(mux), &bodies
}

// TestUnsignedRegistrationOmitsTxChainAddress checks the launcher's outbound unsigned
// registration against the worker contract: the body must not carry a txChainAddress
// field, so the worker accepts it (201). A regression that reintroduces the field
// makes the contract stub return 400 and RegisterNode fail.
func TestUnsignedRegistrationOmitsTxChainAddress(t *testing.T) {
	srv, bodies := workerRegisterContract(t)
	defer srv.Close()

	reg := NodeRegistration{ChainID: "node-1", IP: "203.0.113.5", Port: 8084, Type: "bootnode"}
	if err := RegisterNode(context.Background(), srv.URL, reg); err != nil {
		t.Fatalf("RegisterNode rejected by worker contract: %v", err)
	}
	if len(*bodies) != 1 {
		t.Fatalf("got %d registration bodies, want 1", len(*bodies))
	}
	if strings.Contains(string((*bodies)[0]), "txChainAddress") {
		t.Fatalf("unsigned registration body carries txChainAddress: %s", (*bodies)[0])
	}
}

// TestUnsignedRegistrationStructHasNoTxChainField guards the wire shape: a
// NodeRegistration marshals to exactly the worker's unsigned fields.
func TestUnsignedRegistrationStructHasNoTxChainField(t *testing.T) {
	raw, err := json.Marshal(NodeRegistration{ChainID: "c", Port: 1, Type: "client", IP: "10.0.0.1", PeerID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for k := range got {
		switch k {
		case "chainID", "ip", "port", "type", "peerID":
		default:
			t.Fatalf("unexpected field %q in unsigned registration: %s", k, raw)
		}
	}
}

// TestSignedRootRecordAcceptedByWorkerVerifier covers the signed path the worker
// uses (POST /registry/register → KNIRVSDK verify_registry_record). The record is
// signed by the launcher's production signing function with the trusted root key
// (test-generated); the same Rust verifier the worker runs accepts it against the
// trusted root address and immutable root registry id.
func TestSignedRootRecordAcceptedByWorkerVerifier(t *testing.T) {
	root, _ := newTestRootSigner(t)
	regID := deriveRootRegistryID(root.PublicKey())
	txSigner, _ := newTestRootSigner(t)
	txAddr := txSigner.Address()

	record := knirvsigning.RegistryRecord{
		ChainID:        regID,
		TxChainAddress: txAddr,
		OwnerPubKey:    base64.StdEncoding.EncodeToString(root.PublicKey()),
		Nonce:          1,
		ExpiresAtUnix:  uint64(time.Now().Add(time.Hour).Unix()),
	}
	signed, err := signRootRegistryRecord(context.Background(), root, record, regID, uint64(time.Now().Unix()))
	if err != nil {
		t.Fatalf("sign root record: %v", err)
	}
	got, err := knirvsigning.VerifyRegistryRecord(signed, root.Address(), regID, time.Now())
	if err != nil {
		t.Fatalf("worker verifier rejected the signed root record: %v", err)
	}
	if got.TxChainAddress != txAddr {
		t.Fatalf("verified tx-chain address = %q, want %q", got.TxChainAddress, txAddr)
	}
}

// TestSignedRecordFromNonRootKeyRejectedByWorkerVerifier documents the trust limit
// that blocks a per-node signed tx-chain record: the worker accepts a signed record
// only when the signer is the configured root identity, so a non-root node's own
// key is refused (the worker maps this to 401).
func TestSignedRecordFromNonRootKeyRejectedByWorkerVerifier(t *testing.T) {
	root, _ := newTestRootSigner(t)
	node, _ := newTestRootSigner(t)
	regID := deriveRootRegistryID(root.PublicKey())
	record := knirvsigning.RegistryRecord{
		ChainID:        regID,
		TxChainAddress: node.Address(),
		OwnerPubKey:    base64.StdEncoding.EncodeToString(node.PublicKey()),
		Nonce:          1,
		ExpiresAtUnix:  uint64(time.Now().Add(time.Hour).Unix()),
	}
	signed, err := signRootRegistryRecord(context.Background(), node, record, regID, uint64(time.Now().Unix()))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := knirvsigning.VerifyRegistryRecord(signed, root.Address(), regID, time.Now()); err == nil {
		t.Fatal("worker verifier accepted a non-root signed record; trust model changed, revisit the owner decision")
	}
}

// TestNodeRegistryRegistrationAcceptedByWorkerContract drives the launcher's own
// registration builder (what startNodeRegistry sends) through the worker contract
// stub. A txChainAddress reintroduced into the builder makes the worker return 400.
func TestNodeRegistryRegistrationAcceptedByWorkerContract(t *testing.T) {
	t.Setenv(nodeRegistrationIDEnv, "node-launcher-1")
	srv, bodies := workerRegisterContract(t)
	defer srv.Close()

	app := &ServerApp{config: &Config{Host: "203.0.113.5", Port: 8084}}
	reg, ok := app.nodeRegistryRegistration()
	if !ok {
		t.Fatal("nodeRegistryRegistration: no registration built")
	}
	if err := RegisterNode(context.Background(), srv.URL, reg); err != nil {
		t.Fatalf("launcher registration rejected by worker contract: %v", err)
	}
	if len(*bodies) != 1 || strings.Contains(string((*bodies)[0]), "txChainAddress") {
		t.Fatalf("launcher body = %q, want one body without txChainAddress", *bodies)
	}
}

// TestLookupTxChainAddressUnsignedReturnsEmptyForWorkerNode: the worker's unsigned
// /lookup never returns a tx-chain address (none is stored through that path), so
// the unsigned lookup reports empty without error for a registered node.
func TestLookupTxChainAddressUnsignedReturnsEmptyForWorkerNode(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/lookup/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/lookup/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ip":"203.0.113.5","port":8084,"type":"bootnode","validated":false}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	addr, err := LookupTxChainAddress(context.Background(), srv.URL, "node-2")
	if err != nil {
		t.Fatalf("LookupTxChainAddress: %v", err)
	}
	if addr != "" {
		t.Fatalf("unsigned lookup returned %q, want empty", addr)
	}
}
