package blockchain

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleChainHeadReportsLatestBlock(t *testing.T) {
	bcs := &BlockchainServer{BlockchainPtr: &BlockchainStruct{
		ChainID: "knirv-test",
		Blocks: []*Block{
			{BlockNumber: 1, BlockHash: []byte{0x01}},
			{BlockNumber: 2, BlockHash: []byte{0xab, 0xcd}, Timestamp: 99},
		},
	}}

	rec := httptest.NewRecorder()
	bcs.handleChainHead(rec, httptest.NewRequest(http.MethodGet, "/chain/head", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var head map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &head); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if head["chain_id"] != "knirv-test" || head["block_hash"] != "abcd" || head["height"].(float64) != 2 {
		t.Fatalf("unexpected head: %v", head)
	}
}

func TestHandleChainHeadEmptyChain(t *testing.T) {
	bcs := &BlockchainServer{BlockchainPtr: &BlockchainStruct{ChainID: "knirv-test"}}
	rec := httptest.NewRecorder()
	bcs.handleChainHead(rec, httptest.NewRequest(http.MethodGet, "/chain/head", nil))
	if rec.Code != http.StatusOK || rec.Body.String() == "" {
		t.Fatalf("empty chain: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestHandleAccountBalanceRequiresAddress(t *testing.T) {
	bcs := &BlockchainServer{}
	rec := httptest.NewRecorder()
	bcs.handleAccountBalance(rec, httptest.NewRequest(http.MethodGet, "/accounts/balance", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
