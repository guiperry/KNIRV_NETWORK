package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"

	constants "KNIRVCHAIN_GO_Verifyer/constants"
	"KNIRVCHAIN_GO_Verifyer/types"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey `json:"private_key"`
	PublicKey  *ecdsa.PublicKey  `json:"public_key"`
}

func NewWallet() (*Wallet, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	wallet := new(Wallet)
	wallet.PrivateKey = privateKey
	wallet.PublicKey = &privateKey.PublicKey

	return wallet, nil
}

func NewWalletFromPrivateKeyHex(privateKeyHex string) *Wallet {
	pk := privateKeyHex[2:]
	d := new(big.Int)
	d.SetString(pk, 16)

	var npk ecdsa.PrivateKey
	npk.D = d
	npk.PublicKey.Curve = elliptic.P256()
	npk.PublicKey.X, npk.PublicKey.Y = npk.PublicKey.Curve.ScalarBaseMult(d.Bytes())

	wallet := new(Wallet)
	wallet.PrivateKey = &npk
	wallet.PublicKey = &npk.PublicKey

	return wallet
}

func (w *Wallet) GetPrivateKeyHex() string {
	return fmt.Sprintf("0x%x", w.PrivateKey.D)
}

func (w *Wallet) GetPublicKeyHex() string {
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

func (w *Wallet) GetAddress() string {
	hash := sha256.Sum256([]byte(w.GetPublicKeyHex()[2:]))
	hex := fmt.Sprintf("%x", hash[:])
	address := constants.ADDRESS_PREFIX + hex[len(hex)-40:]
	return address
}

func (w *Wallet) GetSignedTxn(unsignedTxn types.Transaction) (*types.Transaction, error) {
	bs, err := json.Marshal(unsignedTxn)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(bs)

	sig, err := ecdsa.SignASN1(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return nil, err
	}
	signedTxn := types.NewTransaction(unsignedTxn.From, unsignedTxn.To, unsignedTxn.Value, unsignedTxn.Data, unsignedTxn.Origin)
	signedTxn.Signature = sig
	signedTxn.PublicKey = w.GetPublicKeyHex()

	return signedTxn, nil
}
