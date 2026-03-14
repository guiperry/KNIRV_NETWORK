package types

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"math/big"
	"time"

	constants "KNIRVROUTER/internal/constants"
)

type Transaction struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Value     uint64 `json:"value"`
	Data      []byte `json:"data"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
	PublicKey string `json:"public_key,omitempty"`
	Signature []byte `json:"signature"`
	Origin    string `json:"origin,omitempty"` // "nrn" for private, "chain" for public
}

func NewTransaction(from, to string, value uint64, data []byte, origin string) *Transaction {
	t := new(Transaction)
	t.From = from
	t.To = to
	t.Value = value
	t.Data = data
	t.Timestamp = time.Now().UnixNano()
	t.Status = constants.PENDING
	t.PublicKey = ""
	t.Signature = []byte{}
	t.Origin = origin // Set origin explicitly ("nrn" for private, "chain" for public)
	return t
}

func (t Transaction) ToJson() string {
	nb, err := json.Marshal(t)
	if err != nil {
		return err.Error()
	}
	return string(nb)
}

func (t Transaction) VerifyTxn() bool {
	if t.Value <= 0 || t.Value > math.MaxUint64 || t.From == t.To {
		return false
	}
	return t.VerifySignature()
}

func (t Transaction) VerifySignature() bool {
	if t.Signature == nil || t.PublicKey == "" {
		return false
	}

	txnCopy := t
	txnCopy.Signature = []byte{}
	txnCopy.PublicKey = ""

	publicKeyEcdsa := GetPublicKeyFromHex(t.PublicKey)
	bs, _ := json.Marshal(txnCopy)
	hash := sha256.Sum256(bs)
	return ecdsa.VerifyASN1(publicKeyEcdsa, hash[:], t.Signature)
}

func (t Transaction) Hash() string {
	bs, _ := json.Marshal(t)
	sum := sha256.Sum256(bs)
	hexRep := hex.EncodeToString(sum[:32])
	return constants.HEX_PREFIX + hexRep
}

func GetPublicKeyFromHex(publicKeyHex string) *ecdsa.PublicKey {
	rpk := publicKeyHex[2:]
	xHex := rpk[:64]
	yHex := rpk[64:]
	x := new(big.Int)
	y := new(big.Int)
	x.SetString(xHex, 16)
	y.SetString(yHex, 16)

	var npk ecdsa.PublicKey
	npk.Curve = elliptic.P256()
	npk.X = x
	npk.Y = y
	return &npk
}

// IsPrivate checks if a transaction is private (nrn://)
func (t Transaction) IsPrivate() bool {
	return t.Origin == constants.ORIGIN_PRIVATE
}

// IsPublic checks if a transaction is public (chain://)
func (t Transaction) IsPublic() bool {
	return t.Origin == constants.ORIGIN_PUBLIC
}
