package validationchain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Mock Server ---

type mockValidationHandler struct {
	t                *testing.T
	policyRequests   []PolicyCommitRequest
	validationReqs   []CommitValidationResultRequest
	evidenceReqs     []EvidenceAnchorRequest
	dveURIRequests   []DVEURIRecord
	existingRecords  map[string]map[string]interface{}
}

func newMockHandler(t *testing.T) *mockValidationHandler {
	return &mockValidationHandler{
		t:               t,
		existingRecords: make(map[string]map[string]interface{}),
	}
}

func (h *mockValidationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method + " " + r.URL.Path {
	case "POST /validation/commit":
		var req CommitValidationResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.validationReqs = append(h.validationReqs, req)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": "val-tx-" + req.ValidationID,
		})

	case "POST /policy/commit":
		var req PolicyCommitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.policyRequests = append(h.policyRequests, req)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": "pol-tx-" + req.Name,
		})

	case "POST /evidence/anchor":
		var req EvidenceAnchorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.evidenceReqs = append(h.evidenceReqs, req)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": "ev-tx-" + req.EvidenceID,
		})

	case "POST /dve_uri/submit":
		var req DVEURIRecord
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.dveURIRequests = append(h.dveURIRequests, req)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": "dve-tx-" + req.DVEID,
		})

	case "GET /records/test-record-1":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         "test-record-1",
			"record_type": "validation",
			"payload":    map[string]interface{}{"result": "pass"},
		})

	case "GET /chain/height":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"height": uint64(42),
		})

	case "GET /health":
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		h.t.Logf("Unhandled request: %s %s", r.Method, r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
	}
}

// --- Tests ---

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:9999")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.baseURL != "http://localhost:9999" {
		t.Errorf("expected baseURL http://localhost:9999, got %s", client.baseURL)
	}
}

func TestClientHealth(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)
	if err := client.Health(); err != nil {
		t.Errorf("Health() returned error: %v", err)
	}
}

func TestClientHealth_Error(t *testing.T) {
	client := NewClient("http://localhost:1")
	if err := client.Health(); err == nil {
		t.Error("expected error for unreachable server")
	}
}

func TestClientCommitPolicy(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	req := PolicyCommitRequest{
		Name:     "test-policy",
		Type:     "guardrail_policy",
		TargetDVE: "dve-1",
		Priority: 5,
		Enabled:  true,
	}

	txHash, err := client.CommitPolicy(req)
	if err != nil {
		t.Fatalf("CommitPolicy() error: %v", err)
	}
	if txHash != "pol-tx-test-policy" {
		t.Errorf("expected tx hash pol-tx-test-policy, got %s", txHash)
	}
	if len(handler.policyRequests) != 1 {
		t.Fatalf("expected 1 policy request, got %d", len(handler.policyRequests))
	}
	if handler.policyRequests[0].Name != "test-policy" {
		t.Errorf("expected policy name test-policy, got %s", handler.policyRequests[0].Name)
	}
}

func TestClientCommitPolicy_WithRules(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	req := PolicyCommitRequest{
		Name:    "memory-limit",
		Type:    "guardrail_policy",
		Enabled: true,
		Rules: map[string]interface{}{
			"rule-1": map[string]interface{}{
				"type":       "guardrail",
				"condition":  map[string]interface{}{"max_value": 1024.0},
				"action":     "block",
				"parameters": map[string]interface{}{"guardrail_type": "memory"},
			},
		},
	}

	txHash, err := client.CommitPolicy(req)
	if err != nil {
		t.Fatalf("CommitPolicy() error: %v", err)
	}
	if txHash == "" {
		t.Error("expected non-empty transaction hash")
	}
	// Verify rules are passed through
	if len(handler.policyRequests) != 1 {
		t.Fatalf("expected 1 policy request, got %d", len(handler.policyRequests))
	}
	rules, ok := handler.policyRequests[0].Rules["rule-1"].(map[string]interface{})
	if !ok {
		t.Fatal("expected rule-1 in rules map")
	}
	if rules["action"] != "block" {
		t.Errorf("expected action block, got %v", rules["action"])
	}
}

func TestClientCommitValidationResult(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	req := CommitValidationResultRequest{
		ValidationID: "val-001",
		NodeID:       "node-1",
		ResultType:   "workflow_chain_commit",
		Payload:      map[string]interface{}{"evidence_id": "ev-001", "status": "verified"},
	}

	txHash, err := client.CommitValidationResult(req)
	if err != nil {
		t.Fatalf("CommitValidationResult() error: %v", err)
	}
	if txHash != "val-tx-val-001" {
		t.Errorf("expected tx hash val-tx-val-001, got %s", txHash)
	}
	if len(handler.validationReqs) != 1 {
		t.Fatalf("expected 1 validation request, got %d", len(handler.validationReqs))
	}
	if handler.validationReqs[0].ValidationID != "val-001" {
		t.Errorf("expected ValidationID val-001, got %s", handler.validationReqs[0].ValidationID)
	}
}

func TestClientAnchorEvidencePack(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	eid := "ev-001"
	req := EvidenceAnchorRequest{
		EvidenceID:   eid,
		EvidenceType: "dilithium_signature",
		NodeID:       "node-1",
		ValidationID: "val-001",
		Payload:      map[string]interface{}{"hash": "abc123"},
		Signature:    "dilithium_sig_123",
		Algorithm:    "DILITHIUM3",
		PublicKey:    "pk_abc",
	}

	txHash, err := client.AnchorEvidencePack(req)
	if err != nil {
		t.Fatalf("AnchorEvidencePack() error: %v", err)
	}
	if txHash != "ev-tx-ev-001" {
		t.Errorf("expected tx hash ev-tx-ev-001, got %s", txHash)
	}
	if len(handler.evidenceReqs) != 1 {
		t.Fatalf("expected 1 evidence request, got %d", len(handler.evidenceReqs))
	}
	if handler.evidenceReqs[0].EvidenceType != "dilithium_signature" {
		t.Errorf("expected EvidenceType dilithium_signature, got %s", handler.evidenceReqs[0].EvidenceType)
	}
}

func TestClientSubmitDVEURI(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	req := DVEURIRecord{
		DVEID:      "dve-001",
		FullURI:    "https://dve-001.knirv.com",
		WalletAddr: "0xabc123",
		CreatedAt:  1234567890,
		TxHash:     "",
	}

	txHash, err := client.SubmitDVEURI(req)
	if err != nil {
		t.Fatalf("SubmitDVEURI() error: %v", err)
	}
	if txHash != "dve-tx-dve-001" {
		t.Errorf("expected tx hash dve-tx-dve-001, got %s", txHash)
	}
	if len(handler.dveURIRequests) != 1 {
		t.Fatalf("expected 1 DVE URI request, got %d", len(handler.dveURIRequests))
	}
}

func TestClientGetRecord(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	record, err := client.GetRecord("test-record-1")
	if err != nil {
		t.Fatalf("GetRecord() error: %v", err)
	}
	if record == nil {
		t.Fatal("expected non-nil record")
	}
	if record["id"] != "test-record-1" {
		t.Errorf("expected record id test-record-1, got %v", record["id"])
	}
}

func TestClientGetBlockHeight(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	height, err := client.GetBlockHeight()
	if err != nil {
		t.Fatalf("GetBlockHeight() error: %v", err)
	}
	if height != 42 {
		t.Errorf("expected height 42, got %d", height)
	}
}

func TestClientGetBlockHeight_FallbackToBlocks(t *testing.T) {
	// Server that only serves /blocks (not /chain/height)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/chain/height":
			http.Error(w, "not found", http.StatusNotFound)
		case "/blocks":
			blocks := make([]json.RawMessage, 10)
			json.NewEncoder(w).Encode(blocks)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	height, err := client.GetBlockHeight()
	if err != nil {
		t.Fatalf("GetBlockHeight() error: %v", err)
	}
	if height != 9 {
		t.Errorf("expected height 9 (10 blocks - 1), got %d", height)
	}
}

func TestClientGetDVEURI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dve_uri/dve-001" {
			json.NewEncoder(w).Encode(DVEURIRecord{
				DVEID:      "dve-001",
				FullURI:    "https://dve-001.knirv.com",
				WalletAddr: "0xabc",
				CreatedAt:  12345,
				TxHash:     "tx-abc",
			})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	record, err := client.GetDVEURI("dve-001")
	if err != nil {
		t.Fatalf("GetDVEURI() error: %v", err)
	}
	if record.DVEID != "dve-001" {
		t.Errorf("expected DVEID dve-001, got %s", record.DVEID)
	}
}

func TestClientGetDVEURIByWallet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dve_uri/by_wallet/0xabc" {
			json.NewEncoder(w).Encode([]DVEURIRecord{
				{DVEID: "dve-001", FullURI: "https://dve-001.knirv.com", WalletAddr: "0xabc", TxHash: "tx-1"},
				{DVEID: "dve-002", FullURI: "https://dve-002.knirv.com", WalletAddr: "0xabc", TxHash: "tx-2"},
			})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	records, err := client.GetDVEURIByWallet("0xabc")
	if err != nil {
		t.Fatalf("GetDVEURIByWallet() error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
}

func TestClientSubmitTransaction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/send_txn" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"transaction_hash": "tx-raw-001",
			})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	txHash, err := client.SubmitTransaction(map[string]string{"action": "test"})
	if err != nil {
		t.Fatalf("SubmitTransaction() error: %v", err)
	}
	if txHash != "tx-raw-001" {
		t.Errorf("expected tx hash tx-raw-001, got %s", txHash)
	}
}

func TestClientCommitPolicy_FallbackToSubmitTransaction(t *testing.T) {
	// When /policy/commit fails (returns 404), it should fall back to SubmitTransaction
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/policy/commit":
			http.Error(w, "not found", http.StatusNotFound)
		case "/send_txn":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"transaction_hash": "fallback-tx-001",
			})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	txHash, err := client.CommitPolicy(PolicyCommitRequest{Name: "test-policy"})
	if err != nil {
		t.Fatalf("CommitPolicy() fallback error: %v", err)
	}
	if txHash != "fallback-tx-001" {
		t.Errorf("expected fallback tx hash fallback-tx-001, got %s", txHash)
	}
}

func TestClientGetRecord_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)
	_, err := client.GetRecord("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent record")
	}
}

// Test all three commit paths together simulating a real flow
func TestClient_ValidationPipeline(t *testing.T) {
	handler := newMockHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL)

	// Step 1: Commit a policy
	polTx, err := client.CommitPolicy(PolicyCommitRequest{
		Name: "resource-limits", Type: "guardrail", Enabled: true,
	})
	if err != nil {
		t.Fatalf("step 1 policy commit: %v", err)
	}
	if polTx != "pol-tx-resource-limits" {
		t.Errorf("step 1: unexpected tx hash %s", polTx)
	}

	// Step 2: Commit a validation result
	valTx, err := client.CommitValidationResult(CommitValidationResultRequest{
		ValidationID: "val-002", NodeID: "node-1",
	})
	if err != nil {
		t.Fatalf("step 2 validation commit: %v", err)
	}
	if valTx != "val-tx-val-002" {
		t.Errorf("step 2: unexpected tx hash %s", valTx)
	}

	// Step 3: Anchor evidence
	eid := "ev-002"
	evTx, err := client.AnchorEvidencePack(EvidenceAnchorRequest{
		EvidenceID: eid, EvidenceType: "dilithium",
	})
	if err != nil {
		t.Fatalf("step 3 evidence anchor: %v", err)
	}
	if evTx != "ev-tx-ev-002" {
		t.Errorf("step 3: unexpected tx hash %s", evTx)
	}

	// Verify all 3 requests were received by the mock
	if len(handler.policyRequests) != 1 {
		t.Errorf("expected 1 policy request, got %d", len(handler.policyRequests))
	}
	if len(handler.validationReqs) != 1 {
		t.Errorf("expected 1 validation request, got %d", len(handler.validationReqs))
	}
	if len(handler.evidenceReqs) != 1 {
		t.Errorf("expected 1 evidence request, got %d", len(handler.evidenceReqs))
	}
}

func TestClient_RequestTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't respond — let the client timeout
		select {}
	}))
	defer slow.Close()

	// Override the default client timeout by creating client with short timeout
	client := &Client{
		baseURL: slow.URL,
		client:  &http.Client{Timeout: 1}, // 1 nanosecond effectively immediate timeout
	}

	var err error
	_, err = client.CommitPolicy(PolicyCommitRequest{Name: "timeout-test"})
	if err == nil {
		t.Error("expected timeout error")
	}
	fmt.Println(err)
}

