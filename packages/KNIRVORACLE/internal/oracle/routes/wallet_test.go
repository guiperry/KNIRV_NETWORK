package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"go.uber.org/zap"
)

func TestGenerateWalletFundsNewWallet(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/generate_wallet", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp walletCreateResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.PrivateKey == "" || resp.PublicKey == "" || resp.Address == "" {
		t.Fatalf("expected generated wallet credentials, got %+v", resp)
	}
	if resp.InitialFunding != "1000" {
		t.Fatalf("initial funding = %q, want 1000", resp.InitialFunding)
	}
	if resp.Balance != "1000" {
		t.Fatalf("balance = %q, want 1000", resp.Balance)
	}
}

func TestSendSignedTransactionTransfersOracleBalances(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	sender := createWalletThroughAPI(t, mux)
	receiver := createWalletThroughAPI(t, mux)

	body, _ := json.Marshal(legacyTransactionRequest{
		FromAddress: sender.Address,
		ToAddress:   receiver.Address,
		Value:       250,
		PrivateKey:  sender.PrivateKey,
		PublicKey:   sender.PublicKey,
	})

	req := httptest.NewRequest(http.MethodPost, "/send_signed_txn", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	balanceReq := httptest.NewRequest(http.MethodGet, "/balance/"+sender.Address, nil)
	balanceRec := httptest.NewRecorder()
	mux.ServeHTTP(balanceRec, balanceReq)
	if balanceRec.Code != http.StatusOK {
		t.Fatalf("sender balance status = %d, body = %s", balanceRec.Code, balanceRec.Body.String())
	}
	bodyBytes, _ := io.ReadAll(balanceRec.Body)
	if !bytes.Contains(bodyBytes, []byte(`"balance":"750"`)) {
		t.Fatalf("sender balance body = %s, want 750", string(bodyBytes))
	}

	receiverBalanceReq := httptest.NewRequest(http.MethodGet, "/balance/"+receiver.Address, nil)
	receiverBalanceRec := httptest.NewRecorder()
	mux.ServeHTTP(receiverBalanceRec, receiverBalanceReq)
	if receiverBalanceRec.Code != http.StatusOK {
		t.Fatalf("receiver balance status = %d, body = %s", receiverBalanceRec.Code, receiverBalanceRec.Body.String())
	}
	receiverBodyBytes, _ := io.ReadAll(receiverBalanceRec.Body)
	if !bytes.Contains(receiverBodyBytes, []byte(`"balance":"1250"`)) {
		t.Fatalf("receiver balance body = %s, want 1250", string(receiverBodyBytes))
	}
}

func TestOracleFaucetFundsWallet(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	walletResp := createWalletThroughAPI(t, mux)

	body, _ := json.Marshal(faucetRequest{
		Address: walletResp.Address,
		Amount:  500,
	})
	req := httptest.NewRequest(http.MethodPost, "/test/faucet", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	balanceReq := httptest.NewRequest(http.MethodGet, "/balance/"+walletResp.Address, nil)
	balanceRec := httptest.NewRecorder()
	mux.ServeHTTP(balanceRec, balanceReq)
	if balanceRec.Code != http.StatusOK {
		t.Fatalf("balance status = %d, body = %s", balanceRec.Code, balanceRec.Body.String())
	}
	bodyBytes, _ := io.ReadAll(balanceRec.Body)
	if !bytes.Contains(bodyBytes, []byte(`"balance":"1500"`)) {
		t.Fatalf("balance body = %s, want 1500", string(bodyBytes))
	}
}

func newTestOracle(t *testing.T) *oracle.Oracle {
	t.Helper()

	ownerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate owner key: %v", err)
	}

	cfg := oracle.DefaultOracleConfig()
	cfg.OwnerPrivateKey = ownerKey.PrivateKeyHex()
	cfg.DataDir = t.TempDir()
	cfg.DBBackend = "memory"
	cfg.WalletServerEnabled = true
	cfg.WalletInitialFund = big.NewInt(1000)

	instance, err := oracle.NewOracle(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("new oracle: %v", err)
	}

	return instance
}

func createWalletThroughAPI(t *testing.T, mux *http.ServeMux) walletCreateResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/generate_wallet", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp walletCreateResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode wallet: %v", err)
	}
	return resp
}
