package routes

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
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

func TestRegisterWalletFundsClientAddressWithoutReturningKeys(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"address":   clientKey.Address.String(),
		"owner_id":  "controller-vault",
		"signature": signRegisterForTest(t, clientKey, clientKey.Address.String(), "controller-vault"),
	})
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/wallet/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if raw["address"] != clientKey.Address.String() {
		t.Fatalf("address = %v, want %s", raw["address"], clientKey.Address.String())
	}
	if raw["initial_funding"] != "1000" || raw["balance"] != "1000" {
		t.Fatalf("funding response = %+v, want 1000 funding and balance", raw)
	}
	if _, ok := raw["private_key"]; ok {
		t.Fatalf("registration response leaked private_key: %+v", raw)
	}
	if _, ok := raw["public_key"]; ok {
		t.Fatalf("registration response leaked public_key: %+v", raw)
	}
}

func TestRegisterWalletRequiresOwnerSignatureAndFundsOnlyOnce(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	clientKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	attackerKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	wrongSignerSig := signRegisterForTest(t, attackerKey, clientKey.Address.String(), "controller-vault")
	postRegisterWallet(t, mux, clientKey.Address.String(), "controller-vault", wrongSignerSig, http.StatusBadRequest)
	if got := oracleInstance.GetNRNToken().GetBalance(clientKey.Address).String(); got != "0" {
		t.Fatalf("balance after bad signature = %s, want 0", got)
	}

	validSig := signRegisterForTest(t, clientKey, clientKey.Address.String(), "controller-vault")
	for i := 0; i < 3; i++ {
		postRegisterWallet(t, mux, clientKey.Address.String(), "controller-vault", validSig, http.StatusCreated)
		if got := oracleInstance.GetNRNToken().GetBalance(clientKey.Address).String(); got != "1000" {
			t.Fatalf("balance after registration %d = %s, want 1000", i+1, got)
		}
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

func TestSignedTransferVerifiesSignatureAndNonce(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	sender, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate sender key: %v", err)
	}
	receiver, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate receiver key: %v", err)
	}
	attacker, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	if _, err := oracleInstance.FundAddress(sender.Address, big.NewInt(1000), "test funding"); err != nil {
		t.Fatalf("fund sender: %v", err)
	}

	nonceReq := httptest.NewRequest(http.MethodGet, "/oracle/v3/account/nonce/"+sender.Address.String(), nil)
	nonceRec := httptest.NewRecorder()
	mux.ServeHTTP(nonceRec, nonceReq)
	if nonceRec.Code != http.StatusOK {
		t.Fatalf("nonce status = %d, body = %s", nonceRec.Code, nonceRec.Body.String())
	}
	if !bytes.Contains(nonceRec.Body.Bytes(), []byte(`"nonce":0`)) {
		t.Fatalf("nonce body = %s, want nonce 0", nonceRec.Body.String())
	}

	validSig := signTransferForTest(t, sender, sender.Address.String(), receiver.Address.String(), "250", 0)
	postSignedTransfer(t, mux, sender.Address.String(), receiver.Address.String(), "250", 0, validSig, http.StatusOK)

	if got := oracleInstance.GetNRNToken().GetBalance(sender.Address).String(); got != "750" {
		t.Fatalf("sender balance = %s, want 750", got)
	}
	if got := oracleInstance.GetNRNToken().GetBalance(receiver.Address).String(); got != "250" {
		t.Fatalf("receiver balance = %s, want 250", got)
	}
	if got := oracleInstance.GetNRNToken().GetNonce(sender.Address); got != 1 {
		t.Fatalf("nonce = %d, want 1", got)
	}

	postSignedTransfer(t, mux, sender.Address.String(), receiver.Address.String(), "250", 0, validSig, http.StatusBadRequest)

	wrongNonceSig := signTransferForTest(t, sender, sender.Address.String(), receiver.Address.String(), "10", 99)
	postSignedTransfer(t, mux, sender.Address.String(), receiver.Address.String(), "10", 99, wrongNonceSig, http.StatusBadRequest)

	wrongSignerSig := signTransferForTest(t, attacker, sender.Address.String(), receiver.Address.String(), "10", 1)
	postSignedTransfer(t, mux, sender.Address.String(), receiver.Address.String(), "10", 1, wrongSignerSig, http.StatusBadRequest)
}

func TestSignedBurnVerifiesSignatureAndNonce(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	burner, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate burner key: %v", err)
	}
	attacker, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate attacker key: %v", err)
	}

	initialSupply := oracleInstance.GetNRNToken().TotalSupply()
	if _, err := oracleInstance.FundAddress(burner.Address, big.NewInt(1000), "test funding"); err != nil {
		t.Fatalf("fund burner: %v", err)
	}

	validSig := signBurnForTest(t, burner, burner.Address.String(), "250", "test burn", 0)
	postSignedBurn(t, mux, burner.Address.String(), "250", "test burn", 0, validSig, http.StatusOK)

	if got := oracleInstance.GetNRNToken().GetBalance(burner.Address).String(); got != "750" {
		t.Fatalf("burner balance = %s, want 750", got)
	}
	wantSupply := new(big.Int).Add(initialSupply, big.NewInt(750)).String()
	if got := oracleInstance.GetNRNToken().TotalSupply().String(); got != wantSupply {
		t.Fatalf("total supply = %s, want %s", got, wantSupply)
	}
	if got := oracleInstance.GetNRNToken().GetNonce(burner.Address); got != 1 {
		t.Fatalf("nonce = %d, want 1", got)
	}

	postSignedBurn(t, mux, burner.Address.String(), "250", "test burn", 0, validSig, http.StatusBadRequest)

	wrongNonceSig := signBurnForTest(t, burner, burner.Address.String(), "10", "test burn", 99)
	postSignedBurn(t, mux, burner.Address.String(), "10", "test burn", 99, wrongNonceSig, http.StatusBadRequest)

	wrongSignerSig := signBurnForTest(t, attacker, burner.Address.String(), "10", "test burn", 1)
	postSignedBurn(t, mux, burner.Address.String(), "10", "test burn", 1, wrongSignerSig, http.StatusBadRequest)
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

func signRegisterForTest(t *testing.T, keyPair *crypto.KeyPair, address, ownerID string) string {
	t.Helper()

	parsedAddress, err := types.AddressFromString(address)
	if err != nil {
		t.Fatalf("parse registration address: %v", err)
	}
	message := []byte(walletRegistrationMessage(parsedAddress, ownerID))
	signature, err := keyPair.Sign(message)
	if err != nil {
		t.Fatalf("sign registration: %v", err)
	}
	return fmt.Sprintf("0x%x", signature)
}

func signTransferForTest(t *testing.T, keyPair *crypto.KeyPair, from, to, amount string, nonce uint64) string {
	t.Helper()

	message := []byte(fmt.Sprintf("transfer:%s:%s:%s:%d", from, to, amount, nonce))
	signature, err := keyPair.Sign(message)
	if err != nil {
		t.Fatalf("sign transfer: %v", err)
	}
	return fmt.Sprintf("0x%x", signature)
}

func signBurnForTest(t *testing.T, keyPair *crypto.KeyPair, from, amount, reason string, nonce uint64) string {
	t.Helper()

	message := []byte(fmt.Sprintf("burn:%s:%s:%s:%d", from, amount, reason, nonce))
	signature, err := keyPair.Sign(message)
	if err != nil {
		t.Fatalf("sign burn: %v", err)
	}
	return fmt.Sprintf("0x%x", signature)
}

func postRegisterWallet(t *testing.T, mux *http.ServeMux, address, ownerID, signature string, wantStatus int) {
	t.Helper()

	body, _ := json.Marshal(map[string]interface{}{
		"address":   address,
		"owner_id":  ownerID,
		"signature": signature,
	})
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/wallet/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, wantStatus, rec.Body.String())
	}
}

func postSignedTransfer(t *testing.T, mux *http.ServeMux, from, to, amount string, nonce uint64, signature string, wantStatus int) {
	t.Helper()

	body, _ := json.Marshal(map[string]interface{}{
		"from":      from,
		"to":        to,
		"amount":    amount,
		"nonce":     nonce,
		"signature": signature,
	})
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/token/transfer/signed", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, wantStatus, rec.Body.String())
	}
}

func postSignedBurn(t *testing.T, mux *http.ServeMux, from, amount, reason string, nonce uint64, signature string, wantStatus int) {
	t.Helper()

	body, _ := json.Marshal(map[string]interface{}{
		"from":      from,
		"amount":    amount,
		"reason":    reason,
		"nonce":     nonce,
		"signature": signature,
	})
	req := httptest.NewRequest(http.MethodPost, "/oracle/v3/token/burn/signed", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, wantStatus, rec.Body.String())
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

// TestTSGoSignatureRoundTrip verifies that signatures produced by
// knirvwallet-module's signOracleMessage (cosmjs/secp256k1) are
// verifiable by Go's ethcrypto.SigToPub (go-ethereum). This is the
// critical interop gate before wiring Piece 3/4 transfers.
func TestTSGoSignatureRoundTrip(t *testing.T) {
	// Known keypair from the TypeScript spike: private key with only
	// the last byte set to 1. This produces a deterministic secp256k1
	// keypair that both TypeScript and Go can independently derive.
	privateKeyHex := "0000000000000000000000000000000000000000000000000000000000000001"
	message := "register:0xTESTADDR:controller-test"

	// Signature produced by TypeScript's signOracleMessage (cosmjs):
	//   keccak_256(message) → Secp256k1.createSignature → 65-byte fixed length
	tsSignatureHex := "2b78463fda6665f1732713020aa171b6a04f978b0225a84054069fd1b04a7707752d3bb3bba65c7b751e261a95f39a55dca9bbb7ea03caab72fc35e54b3b0f2801"

	expectedAddress := "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"

	sig, err := hex.DecodeString(tsSignatureHex)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	recovered, err := crypto.RecoverAddress([]byte(message), sig)
	if err != nil {
		t.Fatalf("recover address: %v", err)
	}
	if recovered.String() != expectedAddress {
		t.Fatalf("recovered address = %s, want %s", recovered.String(), expectedAddress)
	}

	// Also verify the same keypair in Go produces a compatible address
	kp, err := crypto.PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	if kp.Address.String() != expectedAddress {
		t.Fatalf("go-derived address = %s, want %s", kp.Address.String(), expectedAddress)
	}

	// Verify Go can also sign and recover the same message
	goSig, err := kp.Sign([]byte(message))
	if err != nil {
		t.Fatalf("go sign: %v", err)
	}
	goRecovered, err := crypto.RecoverAddress([]byte(message), goSig)
	if err != nil {
		t.Fatalf("go recover: %v", err)
	}
	if goRecovered.String() != expectedAddress {
		t.Fatalf("go recovered address = %s, want %s", goRecovered.String(), expectedAddress)
	}
}
