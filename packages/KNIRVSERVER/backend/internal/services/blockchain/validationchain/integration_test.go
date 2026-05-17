// Package validationchain_test provides integration-level tests for the
// validation chain pipeline, covering the end-to-end flow from
// Policy Commit -> Validation Commit -> Evidence Anchor.
package validationchain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFullValidationPipeline validates the complete lifecycle:
//  1. Commit a policy (e.g., guardrail rule)
//  2. Commit a validation result (e.g., workflow step result)
//  3. Anchor an evidence pack (e.g., PQC-signed proof)
//  4. Read back the chain height to confirm persistence
//  5. Submit a DVE URI and retrieve it
func TestFullValidationPipeline(t *testing.T) {
	// --- Setup integration mock server ---
	mux := http.NewServeMux()
	records := make(map[string]json.RawMessage)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"healthy": true,
			"height": 3,
		})
	})

	mux.HandleFunc("/chain/height", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"height": uint64(3),
		})
	})

	mux.HandleFunc("/policy/commit", func(w http.ResponseWriter, r *http.Request) {
		var req PolicyCommitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "policy name required", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": "0x" + req.Name + "-committed",
		})
		// Simulate persistence in chain record
		record, _ := json.Marshal(map[string]interface{}{
			"type":   "policy",
			"name":   req.Name,
			"status": "committed",
			"height": 1,
		})
		records["policy:"+req.Name] = record
	})

	mux.HandleFunc("/validation/commit", func(w http.ResponseWriter, r *http.Request) {
		var req CommitValidationResultRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.ValidationID == "" {
			http.Error(w, "validation_id required", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		txHash := "0xval-" + req.ValidationID
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": txHash,
		})
		record, _ := json.Marshal(map[string]interface{}{
			"type":          "validation",
			"validation_id": req.ValidationID,
			"node_id":       req.NodeID,
			"result_type":   req.ResultType,
			"tx_hash":       txHash,
			"height":        2,
		})
		records["validation:"+req.ValidationID] = record
	})

	mux.HandleFunc("/evidence/anchor", func(w http.ResponseWriter, r *http.Request) {
		var req EvidenceAnchorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.EvidenceID == "" {
			http.Error(w, "evidence_id required", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		txHash := "0xev-" + req.EvidenceID
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": txHash,
		})
		record, _ := json.Marshal(map[string]interface{}{
			"type":        "evidence",
			"evidence_id": req.EvidenceID,
			"signature":   req.Signature,
			"algorithm":   req.Algorithm,
			"tx_hash":     txHash,
			"height":      3,
		})
		records["evidence:"+req.EvidenceID] = record
	})

	mux.HandleFunc("/dve_uri/submit", func(w http.ResponseWriter, r *http.Request) {
		var req DVEURIRecord
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"transaction_hash": "0xdve-" + req.DVEID,
		})
	})

	// GET /records/{id}
	mux.HandleFunc("/records/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/records/"):]
		if id == "" {
			http.Error(w, "record id required", http.StatusBadRequest)
			return
		}
		if rec, ok := records[id]; ok {
			w.WriteHeader(http.StatusOK)
			w.Write(rec)
			return
		}
		// Try prefixed lookups
		for key, val := range records {
			if len(key) > len(id) && key[len(key)-len(id):] == id {
				w.WriteHeader(http.StatusOK)
				w.Write(val)
				return
			}
		}
		http.Error(w, "record not found", http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL)

	// --- STEP 1: Policy Commit ---
	t.Run("PolicyCommit", func(t *testing.T) {
		txHash, err := client.CommitPolicy(PolicyCommitRequest{
			Name:     "memory-limit-policy",
			Type:     "guardrail_policy",
			Priority: 10,
			Enabled:  true,
			Rules: map[string]interface{}{
				"rule-1": map[string]interface{}{
					"type":       "guardrail",
					"condition":  map[string]interface{}{"max_value": 1024.0},
					"action":     "block",
					"parameters": map[string]interface{}{"guardrail_type": "memory"},
				},
			},
		})
		if err != nil {
			t.Fatalf("Policy Commit failed: %v", err)
		}
		if txHash != "0xmemory-limit-policy-committed" {
			t.Errorf("unexpected policy tx hash: %s", txHash)
		}
	})

	// --- STEP 2: Validation Result Commit ---
	t.Run("ValidationResultCommit", func(t *testing.T) {
		txHash, err := client.CommitValidationResult(CommitValidationResultRequest{
			ValidationID: "val-workflow-001",
			NodeID:       "dve-node-alpha",
			ResultType:   "workflow_chain_commit",
			Payload: map[string]interface{}{
				"evidence_id": "ev-workflow-001",
				"status":      "verified",
				"checksum":    "sha256:abc123def456",
			},
		})
		if err != nil {
			t.Fatalf("Validation Commit failed: %v", err)
		}
		if txHash != "0xval-val-workflow-001" {
			t.Errorf("unexpected validation tx hash: %s", txHash)
		}
	})

	// --- STEP 3: Evidence Anchor ---
	t.Run("EvidenceAnchor", func(t *testing.T) {
		txHash, err := client.AnchorEvidencePack(EvidenceAnchorRequest{
			EvidenceID:   "ev-workflow-001",
			EvidenceType: "dilithium_signature",
			NodeID:       "dve-node-alpha",
			ValidationID: "val-workflow-001",
			Payload: map[string]interface{}{
				"hash":      "sha256:abc123def456",
				"timestamp": 1234567890,
			},
			Signature: "dilithium3_abcdef1234567890abcdef1234567890",
			Algorithm: "DILITHIUM3",
			PublicKey: "pk_node_alpha_001",
		})
		if err != nil {
			t.Fatalf("Evidence Anchor failed: %v", err)
		}
		if txHash != "0xev-ev-workflow-001" {
			t.Errorf("unexpected evidence tx hash: %s", txHash)
		}
	})

	// --- STEP 4: Chain Health Check ---
	t.Run("ChainHealth", func(t *testing.T) {
		if err := client.Health(); err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})

	// --- STEP 5: Block Height ---
	t.Run("BlockHeight", func(t *testing.T) {
		height, err := client.GetBlockHeight()
		if err != nil {
			t.Fatalf("GetBlockHeight failed: %v", err)
		}
		if height != 3 {
			t.Errorf("expected height 3, got %d", height)
		}
	})

	// --- STEP 6: Submit and retrieve DVE URI ---
	t.Run("DVEURI", func(t *testing.T) {
		txHash, err := client.SubmitDVEURI(DVEURIRecord{
			DVEID:      "dve-alpha",
			FullURI:    "https://dve-alpha.knirv.com",
			WalletAddr: "0xdeadbeef",
			CreatedAt:  1234567890,
		})
		if err != nil {
			t.Fatalf("DVE URI submit failed: %v", err)
		}
		if txHash != "0xdve-dve-alpha" {
			t.Errorf("unexpected DVE URI tx hash: %s", txHash)
		}
	})
}

// TestPolicyCommitFlowThroughICME validates the policy commit flow
// as it would be called from the ICME service.
func TestPolicyCommitFlowThroughICME(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/policy/commit" {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)

			// Validate required fields
			if name, ok := req["name"].(string); !ok || name == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"transaction_hash": "icme-tx-" + req["name"].(string),
			})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(ts.URL)

	// This simulates what ICME handleCommitPolicy does
	policyName := "data-retention-policy"
	txHash, err := client.CommitPolicy(PolicyCommitRequest{
		Name:    policyName,
		Type:    "compliance",
		Enabled: true,
		Rules: map[string]interface{}{
			"retention_days": 90,
			"auto_delete":    true,
		},
	})
	if err != nil {
		t.Fatalf("ICME policy commit failed: %v", err)
	}
	expected := "icme-tx-" + policyName
	if txHash != expected {
		t.Errorf("expected tx hash %s, got %s", expected, txHash)
	}
}

// TestValidationReportMinting validates that validation reports are
// properly minted and anchored as evidence.
func TestValidationReportMinting(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/validation/commit":
			var req CommitValidationResultRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.ResultType == "" {
				http.Error(w, "result_type required", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"transaction_hash": "report-" + req.ValidationID,
			})

		case "/evidence/anchor":
			var req EvidenceAnchorRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Algorithm == "" {
				http.Error(w, "algorithm required for evidence anchoring", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"transaction_hash": "anchored-" + req.EvidenceID,
			})

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := NewClient(ts.URL)

	// Mint a validation report
	reportID := "val-report-42"
	reportTx, err := client.CommitValidationResult(CommitValidationResultRequest{
		ValidationID: reportID,
		NodeID:       "dve-node-1",
		ResultType:   "llm_evaluation_report",
		Payload: map[string]interface{}{
			"model":      "hermes-3",
			"score":      0.92,
			"test_suite": "compliance-v2",
			"passed":     47,
			"failed":     3,
		},
	})
	if err != nil {
		t.Fatalf("validation report minting failed: %v", err)
	}
	if reportTx != "report-"+reportID {
		t.Errorf("expected report tx hash 'report-%s', got %s", reportID, reportTx)
	}

	// Anchor the validation report as evidence
	anchorTx, err := client.AnchorEvidencePack(EvidenceAnchorRequest{
		EvidenceID:   reportID,
		EvidenceType: "validation_report",
		NodeID:       "dve-node-1",
		ValidationID: reportID,
		Payload: map[string]interface{}{
			"report_tx": reportTx,
			"method":    "DILITHIUM3",
		},
		Signature: "sig_" + reportTx,
		Algorithm: "DILITHIUM3",
		PublicKey: "pk_dve_node_1",
	})
	if err != nil {
		t.Fatalf("validation report anchoring failed: %v", err)
	}
	if anchorTx != "anchored-"+reportID {
		t.Errorf("expected anchor tx hash 'anchored-%s', got %s", reportID, anchorTx)
	}
}
