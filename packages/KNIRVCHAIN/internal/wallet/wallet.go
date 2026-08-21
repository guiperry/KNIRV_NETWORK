package wallet

import (
	"KNIRVCHAIN/internal/utils"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// WalletImpl stores private and public keys
type WalletImpl struct {
	PrivateKey *ecdsa.PrivateKey `json:"-"` // Exclude from direct JSON marshaling
	PublicKey  *ecdsa.PublicKey  `json:"-"` // Exclude from direct JSON marshaling
	Address    string            // Store the address, will be populated by GetAddress()
}

// walletJSON is an auxiliary struct for custom JSON marshaling/unmarshaling of Wallet
type walletJSON struct {
	PrivateKeyDHex string `json:"private_key_d_hex"`
	CurveName      string `json:"curve_name"`
	PublicKeyXHex  string `json:"public_key_x_hex"`
	PublicKeyYHex  string `json:"public_key_y_hex"`
	Address        string `json:"address"`
}

func NewWallet() (*WalletImpl, error) {
	privateKey, err := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	wallet := new(WalletImpl)
	wallet.PrivateKey = privateKey
	wallet.PublicKey = &privateKey.PublicKey
	wallet.Address = wallet.GetAddress() // Populate address

	return wallet, nil
}

func NewWalletFromPrivateKeyHex(privateKeyHex string) *WalletImpl {
	pk := strings.TrimPrefix(privateKeyHex, "0x")
	parsed, err := ethcrypto.HexToECDSA(pk)
	if err != nil {
		return nil
	}

	wallet := new(WalletImpl)
	wallet.PrivateKey = parsed
	wallet.PublicKey = &parsed.PublicKey
	wallet.Address = wallet.GetAddress() // Populate address

	return wallet
}

func (w *WalletImpl) SignTransaction(tx *Transaction) error {
	if w.PrivateKey == nil || tx.Amount == nil || !tx.Timestamp.After(time.Time{}) {
		return fmt.Errorf("private key, amount, and timestamp are required")
	}
	chainID := tx.ChainID
	if chainID == "" {
		chainID = os.Getenv("KNIRV_CHAIN_ID")
	}
	if chainID == "" {
		chainID = "knirv-1"
	}
	fee := "0"
	if tx.Fee != nil {
		fee = tx.Fee.String()
	}
	signed, err := knirvsigning.SignTransaction(privateKeyBytes(w.PrivateKey), knirvsigning.SignRequest{
		Action:  knirvsigning.Action{Action: "transfer", Sender: tx.From, Recipient: tx.To, Amount: tx.Amount.Uint64(), Payload: tx.Data, TimestampUnix: tx.Timestamp.Unix()},
		ChainID: chainID, AccountNumber: tx.AccountNumber, Sequence: tx.Nonce,
		Fee: knirvsigning.Fee{Denom: "unrn", Amount: fee},
	})
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}
	wire := signed.Wire()
	tx.ChainID, tx.BodyBytes, tx.AuthInfoBytes = chainID, wire.BodyBytes, wire.AuthInfoBytes
	tx.PublicKey, tx.Signatures, tx.Signature, tx.Hash = wire.PublicKey, wire.Signatures, wire.Signatures[0], wire.Hash
	return nil
}

func privateKeyBytes(key *ecdsa.PrivateKey) []byte {
	raw := make([]byte, 32)
	key.D.FillBytes(raw)
	return raw
}
func (w *WalletImpl) GetPrivateKeyHex() string {
	return fmt.Sprintf("0x%x", w.PrivateKey.D)
}

func (w *WalletImpl) PublicKeyString() string {
	return w.GetPublicKeyHex()
}

func (w *WalletImpl) GetPublicKeyHex() string {
	if w.PublicKey == nil || w.PublicKey.X == nil || w.PublicKey.Y == nil {
		// It's possible UnmarshalJSON was called with incomplete data,
		// leaving PublicKey nil. In such cases, we can't generate its hex.
		// log.Println("Warning: GetPublicKeyHex called on wallet with nil PublicKey or coordinates.") // Can be noisy
		return ""
	}
	return base64.StdEncoding.EncodeToString(ethcrypto.CompressPubkey(w.PublicKey))
}

func (w *WalletImpl) GetAddress() string {
	// If the address field is already populated (e.g. from UnmarshalJSON with incomplete keys but valid address),
	// and it looks like a valid address, prefer it.
	if w.Address != "" && strings.HasPrefix(w.Address, utils.ADDRESS_PREFIX) { // Basic check
		return w.Address
	}

	// If we need to generate it from keys, ensure keys are present.
	if w.PublicKey == nil {
		return w.Address
	}
	publicKey := ethcrypto.CompressPubkey(w.PublicKey)
	if len(publicKey) == 0 {
		// log.Println("Warning: GetPublicKeyHex returned empty in GetAddress. Cannot generate address from keys. Returning w.Address.")
		return w.Address // Fallback to w.Address if key-based generation fails (w.Address might be empty)
	}

	address, err := knirvsigning.Address(publicKey, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		return w.Address
	}
	return address
}

func (w *WalletImpl) GetSignedTxn(unsignedTxn Transaction) (*Transaction, error) {
	signedTxn := unsignedTxn
	if err := w.SignTransaction(&signedTxn); err != nil {
		return nil, err
	}
	return &signedTxn, nil
}

// MarshalJSON implements custom JSON marshaling for Wallet
func (w *WalletImpl) MarshalJSON() ([]byte, error) {
	if w.PrivateKey == nil || w.PublicKey == nil {
		// Handle cases where keys might not be fully initialized,
		// though NewWallet and NewWalletFromPrivateKeyHex should ensure they are.
		// Still, good to be safe.
		return json.Marshal(walletJSON{Address: w.Address})
	}
	return json.Marshal(&walletJSON{
		PrivateKeyDHex: hex.EncodeToString(w.PrivateKey.D.Bytes()),
		CurveName:      w.PrivateKey.Curve.Params().Name,
		PublicKeyXHex:  hex.EncodeToString(w.PublicKey.X.Bytes()),
		PublicKeyYHex:  hex.EncodeToString(w.PublicKey.Y.Bytes()),
		Address:        w.Address, // w.Address should be populated by NewWallet or GetAddress
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Wallet
func (w *WalletImpl) UnmarshalJSON(data []byte) error {
	var aux walletJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal wallet data into auxiliary struct: %w. Data: %s", err, string(data))
	}

	if aux.PrivateKeyDHex == "" || aux.CurveName == "" || aux.PublicKeyXHex == "" || aux.PublicKeyYHex == "" {
		// This might be a wallet loaded from an older format or one that only had an address.
		// If aux.Address is present, use it. Otherwise, this wallet will be incomplete.
		w.Address = aux.Address
		log.Printf("Wallet UnmarshalJSON: Incomplete key data in JSON for address (if any): %s. Wallet may be unusable for signing.", aux.Address)
		return nil // Or return an error if a full wallet is always expected
	}

	privDBytes, err := hex.DecodeString(aux.PrivateKeyDHex)
	if err != nil {
		return fmt.Errorf("failed to decode private key D from hex: %w", err)
	}
	pubXBytes, err := hex.DecodeString(aux.PublicKeyXHex)
	if err != nil {
		return fmt.Errorf("failed to decode public key X from hex: %w", err)
	}
	pubYBytes, err := hex.DecodeString(aux.PublicKeyYHex)
	if err != nil {
		return fmt.Errorf("failed to decode public key Y from hex: %w", err)
	}

	curve := ethcrypto.S256()
	if aux.CurveName != curve.Params().Name {
		return fmt.Errorf("unsupported elliptic curve: %s", aux.CurveName)
	}

	priv := new(ecdsa.PrivateKey)
	priv.D = new(big.Int).SetBytes(privDBytes)
	priv.PublicKey.Curve = curve
	priv.PublicKey.X = new(big.Int).SetBytes(pubXBytes)
	priv.PublicKey.Y = new(big.Int).SetBytes(pubYBytes)

	// Sanity check: ensure D generates the provided X, Y on the curve.
	// This also correctly sets PublicKey.X and PublicKey.Y if they were derived from D.
	// Important: The stored X,Y should ideally match what D produces on the curve.
	// If not, it implies an inconsistency in how the wallet was saved or if D was changed.
	// For robustness, we can recalculate X,Y from D.
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(priv.D.Bytes())

	w.PrivateKey = priv
	w.PublicKey = &priv.PublicKey
	w.Address = w.GetAddress() // Recalculate address based on the reconstructed keys

	if aux.Address != "" && aux.Address != w.Address {
		log.Printf("Wallet UnmarshalJSON: Warning - Loaded address '%s' does not match address '%s' recalculated from public key. Using recalculated address.", aux.Address, w.Address)
	}
	return nil
}
