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
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
	"go.uber.org/zap"
)

func TestGenerateWalletRequiresControllerCustody(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/generate_wallet", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
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

func TestLegacyPrivateKeyTransactionIsDisabled(t *testing.T) {
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/send_signed_txn", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
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
	t.Setenv("KNIRV_ENABLE_DEMO", "true")
	oracleInstance := newTestOracle(t)
	router := NewOracleRoutes(oracleInstance, zap.NewNop())
	mux := http.NewServeMux()
	router.RegisterRoutes(mux)

	walletKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate wallet: %v", err)
	}

	body, _ := json.Marshal(faucetRequest{
		Address: walletKey.Address.String(),
		Amount:  500,
	})
	req := httptest.NewRequest(http.MethodPost, "/test/faucet", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	balanceReq := httptest.NewRequest(http.MethodGet, "/balance/"+walletKey.Address.String(), nil)
	balanceRec := httptest.NewRecorder()
	mux.ServeHTTP(balanceRec, balanceReq)
	if balanceRec.Code != http.StatusOK {
		t.Fatalf("balance status = %d, body = %s", balanceRec.Code, balanceRec.Body.String())
	}
	bodyBytes, _ := io.ReadAll(balanceRec.Body)
	if !bytes.Contains(bodyBytes, []byte(`"balance":"500"`)) {
		t.Fatalf("balance body = %s, want 500", string(bodyBytes))
	}
}

func signRegisterForTest(t *testing.T, keyPair *crypto.KeyPair, address, ownerID string) string {
	t.Helper()

	parsedAddress, err := types.AddressFromString(address)
	if err != nil {
		t.Fatalf("parse registration address: %v", err)
	}
	return signEnvelopeForTest(t, keyPair, "wallet-registration", ownerID, []byte(walletRegistrationMessage(parsedAddress, ownerID)))
}

func signTransferForTest(t *testing.T, keyPair *crypto.KeyPair, from, to, amount string, nonce uint64) string {
	t.Helper()

	message := []byte(fmt.Sprintf("transfer:%s:%s:%s:%d", from, to, amount, nonce))
	return signEnvelopeForTest(t, keyPair, "oracle-transfer", fmt.Sprint(nonce), message)
}

func signBurnForTest(t *testing.T, keyPair *crypto.KeyPair, from, amount, reason string, nonce uint64) string {
	t.Helper()

	message := []byte(fmt.Sprintf("burn:%s:%s:%s:%d", from, amount, reason, nonce))
	return signEnvelopeForTest(t, keyPair, "oracle-burn", fmt.Sprint(nonce), message)
}

func signEnvelopeForTest(t *testing.T, keyPair *crypto.KeyPair, purpose, nonce string, payload []byte) string {
	t.Helper()
	now := time.Now().Unix()
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(keyPair.PrivateKey), knirvsigning.MessageEnvelope{
		Domain: "knirv.oracle", Purpose: purpose, ChainID: "knirvoracle-1", Nonce: nonce,
		IssuedAtUnix: now, ExpiresAtUnix: now + 300, Payload: payload,
	})
	if err != nil {
		t.Fatalf("sign %s: %v", purpose, err)
	}
	encoded, err := json.Marshal(signed)
	if err != nil {
		t.Fatalf("encode %s: %v", purpose, err)
	}
	return string(encoded)
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

func TestCanonicalMessageRoundTrip(t *testing.T) {
	privateKey, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	signed, err := knirvsigning.SignMessage(privateKey, knirvsigning.MessageEnvelope{
		Domain: "knirv.oracle", Purpose: "wallet-registration", ChainID: "knirv-1",
		Nonce: "controller-test", IssuedAtUnix: now, ExpiresAtUnix: now + 300,
		Payload: []byte("register:knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e:controller-test"),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if signed.Address != "knirv1w508d6qejxtdg4y5r3zarvary0c5xw7kaqx36e" {
		t.Fatalf("address = %s", signed.Address)
	}
	if err := knirvsigning.VerifyMessage(signed, "knirv.oracle", "wallet-registration", "knirv-1", "controller-test", time.Now()); err != nil {
		t.Fatalf("verify: %v", err)
	}
}
