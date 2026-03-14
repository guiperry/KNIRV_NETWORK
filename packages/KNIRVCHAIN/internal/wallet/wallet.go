package wallet

import (
	"KNIRVCHAIN/internal/utils"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
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
	CurveName      string `json:"curve_name"` // e.g., "P-256"
	PublicKeyXHex  string `json:"public_key_x_hex"`
	PublicKeyYHex  string `json:"public_key_y_hex"`
	Address        string `json:"address"`
}

func NewWallet() (*WalletImpl, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
	pk := privateKeyHex[2:]
	d := new(big.Int)
	d.SetString(pk, 16)

	var npk ecdsa.PrivateKey
	npk.D = d
	npk.PublicKey.Curve = elliptic.P256()
	npk.PublicKey.X, npk.PublicKey.Y = npk.PublicKey.Curve.ScalarBaseMult(d.Bytes())

	wallet := new(WalletImpl)
	wallet.PrivateKey = &npk
	wallet.PublicKey = &npk.PublicKey
	wallet.Address = wallet.GetAddress() // Populate address

	return wallet
}

func (w *WalletImpl) SignTransaction(tx *Transaction) error {
	// Create a simple hash of the transaction data for signing
	// This is a simplified version - in production you'd want proper serialization
	txData := fmt.Sprintf("%s:%s:%s:%s", tx.From, tx.To, tx.Amount.String(), string(tx.Data))
	bs := []byte(txData)

	// Hash the canonical bytes
	hash := sha256.Sum256(bs)

	// Sign the hash
	sig, err := ecdsa.SignASN1(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Update transaction with signature (convert to hex string)
	tx.Signature = fmt.Sprintf("%x", sig)

	// Set the transaction hash
	tx.Hash = fmt.Sprintf("%x", hash)

	return nil
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
	// Use compressed public key format (02/03 prefix + X coordinate)
	prefix := byte(0x02)
	if w.PublicKey.Y.Bit(0) != 0 {
		prefix = 0x03
	}
	compressed := append([]byte{prefix}, w.PublicKey.X.Bytes()...)
	// Add checksum (first 4 bytes of double SHA256)
	hash := sha256.Sum256(compressed)
	hash = sha256.Sum256(hash[:])
	checksum := hash[:4]

	// Base58 encode with checksum
	full := append(compressed, checksum...)
	return "0x" + base58Encode(full)
}

func base58Encode(input []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	var result []byte
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)
		result = append(result, alphabet[mod.Int64()])
	}

	// Reverse bytes
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

func base58Decode(input string) ([]byte, error) {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	result := big.NewInt(0)
	for _, char := range input {
		alphaIndex := -1
		for i, a := range alphabet {
			if rune(a) == char {
				alphaIndex = i
				break
			}
		}
		if alphaIndex == -1 {
			return nil, fmt.Errorf("invalid character '%c' in base58 string", char)
		}
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(alphaIndex)))
	}

	decoded := result.Bytes()

	// Validate checksum
	if len(decoded) < 4 {
		return nil, fmt.Errorf("invalid base58 string (too short for checksum)")
	}
	payload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]

	hash := sha256.Sum256(payload)
	hash = sha256.Sum256(hash[:])

	if !bytes.Equal(checksum, hash[:4]) {
		return nil, fmt.Errorf("invalid base58 checksum")
	}

	return payload, nil
}

func (w *WalletImpl) GetAddress() string {
	// If the address field is already populated (e.g. from UnmarshalJSON with incomplete keys but valid address),
	// and it looks like a valid address, prefer it.
	if w.Address != "" && strings.HasPrefix(w.Address, utils.ADDRESS_PREFIX) { // Basic check
		return w.Address
	}

	// If we need to generate it from keys, ensure keys are present.
	publicKeyHex := w.GetPublicKeyHex() // This will call the revised GetPublicKeyHex
	if publicKeyHex == "" {
		// log.Println("Warning: GetPublicKeyHex returned empty in GetAddress. Cannot generate address from keys. Returning w.Address.")
		return w.Address // Fallback to w.Address if key-based generation fails (w.Address might be empty)
	}

	// Ensure publicKeyHex starts with "0x" and has content after it.
	if !strings.HasPrefix(publicKeyHex, "0x") || len(publicKeyHex) <= 2 {
		// log.Printf("Warning: GetPublicKeyHex returned invalid hex '%s' in GetAddress. Cannot generate address. Returning w.Address.", publicKeyHex)
		return w.Address // Fallback to w.Address
	}

	hash := sha256.Sum256([]byte(publicKeyHex[2:])) // Slice off "0x"
	hexStr := fmt.Sprintf("%x", hash[:])
	address := utils.ADDRESS_PREFIX + hexStr[len(hexStr)-40:]
	return address
}

func (w *WalletImpl) GetSignedTxn(unsignedTxn Transaction) (*Transaction, error) {
	// Create a simple hash of the transaction data for signing
	txData := fmt.Sprintf("%s:%s:%s:%s", unsignedTxn.From, unsignedTxn.To, unsignedTxn.Amount.String(), string(unsignedTxn.Data))
	bs := []byte(txData)

	hash := sha256.Sum256(bs)

	sig, err := ecdsa.SignASN1(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return nil, err
	}

	// Create a copy of the transaction with signature
	signedTxn := &Transaction{
		From:      unsignedTxn.From,
		To:        unsignedTxn.To,
		Amount:    unsignedTxn.Amount,
		Data:      unsignedTxn.Data,
		Timestamp: unsignedTxn.Timestamp,
		Fee:       unsignedTxn.Fee,
		Nonce:     unsignedTxn.Nonce,
		Signature: fmt.Sprintf("%x", sig),
		Hash:      fmt.Sprintf("%x", hash),
	}

	return signedTxn, nil
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

	var curve elliptic.Curve
	switch aux.CurveName {
	case "P-256": // elliptic.P256().Params().Name returns "P-256"
		curve = elliptic.P256()
	// Add other curves if you plan to support them, e.g., P-224, P-384, P-521
	default:
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
