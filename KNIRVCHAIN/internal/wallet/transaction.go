package wallet

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Transaction represents a XION network transaction
type Transaction struct {
	ChainID       string
	AccountNumber uint64
	Sequence      uint64
	Fee           Fee
	Msgs          []Message
	Memo          string
	TimeoutHeight uint64
}

// Fee represents transaction fees
type Fee struct {
	Amount []Coin `json:"amount"`
	Gas    uint64 `json:"gas"`
}

// Coin represents a token amount
type Coin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

// Message represents a transaction message
type Message interface {
	Type() string
	GetSignBytes() []byte
}

// MsgSend represents a token transfer message
type MsgSend struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Amount      []Coin `json:"amount"`
}

func (m *MsgSend) Type() string {
	return "/cosmos.bank.v1beta1.MsgSend"
}

func (m *MsgSend) GetSignBytes() []byte {
	bz, _ := json.Marshal(m)
	return bz
}

// NewTransaction creates a new transaction
func NewTransaction(chainID string, accountNumber, sequence uint64) *Transaction {
	return &Transaction{
		ChainID:       chainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		Msgs:          make([]Message, 0),
		Fee: Fee{
			Amount: []Coin{},
			Gas:    200000, // Default gas limit
		},
	}
}

// AddMessage adds a message to the transaction
func (tx *Transaction) AddMessage(msg Message) {
	tx.Msgs = append(tx.Msgs, msg)
}

// SetFee sets the transaction fee
func (tx *Transaction) SetFee(amount uint64, denom string, gas uint64) {
	tx.Fee = Fee{
		Amount: []Coin{{Denom: denom, Amount: fmt.Sprintf("%d", amount)}},
		Gas:    gas,
	}
}

// SetMemo sets the transaction memo
func (tx *Transaction) SetMemo(memo string) {
	tx.Memo = memo
}

// SignDoc represents the document that needs to be signed
type SignDoc struct {
	ChainID       string    `json:"chain_id"`
	AccountNumber uint64    `json:"account_number,string"`
	Sequence      uint64    `json:"sequence,string"`
	Fee           Fee       `json:"fee"`
	Msgs          []Message `json:"msgs"`
	Memo          string    `json:"memo"`
}

// GetSignBytes returns the bytes to be signed
func (tx *Transaction) GetSignBytes() ([]byte, error) {
	signDoc := SignDoc{
		ChainID:       tx.ChainID,
		AccountNumber: tx.AccountNumber,
		Sequence:      tx.Sequence,
		Fee:           tx.Fee,
		Msgs:          tx.Msgs,
		Memo:          tx.Memo,
	}

	// Marshal to canonical JSON
	bz, err := json.Marshal(signDoc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sign doc: %w", err)
	}

	return bz, nil
}

// Signature represents a transaction signature
type Signature struct {
	PubKey    PublicKey `json:"pub_key"`
	Signature string    `json:"signature"`
}

// PublicKey represents a public key
type PublicKey struct {
	Type string `json:"@type"`
	Key  string `json:"key"`
}

// SignedTx represents a signed transaction ready for broadcasting
type SignedTx struct {
	Body       TxBody       `json:"body"`
	AuthInfo   AuthInfo     `json:"auth_info"`
	Signatures []string     `json:"signatures"`
}

// TxBody represents the transaction body
type TxBody struct {
	Messages      []json.RawMessage `json:"messages"`
	Memo          string            `json:"memo"`
	TimeoutHeight string            `json:"timeout_height"`
}

// AuthInfo represents transaction authentication info
type AuthInfo struct {
	SignerInfos []SignerInfo `json:"signer_infos"`
	Fee         Fee          `json:"fee"`
}

// SignerInfo represents signer information
type SignerInfo struct {
	PublicKey PublicKey    `json:"public_key"`
	ModeInfo  ModeInfo     `json:"mode_info"`
	Sequence  string       `json:"sequence"`
}

// ModeInfo represents signing mode
type ModeInfo struct {
	Single Single `json:"single"`
}

// Single represents single signature mode
type Single struct {
	Mode string `json:"mode"`
}

// PrivateKey represents a simple private key for signing
// In production, this should use proper key management (HSM, keyring, etc.)
type PrivateKey struct {
	bytes []byte
}

// NewPrivateKey creates a private key from bytes
func NewPrivateKey(keyBytes []byte) *PrivateKey {
	return &PrivateKey{bytes: keyBytes}
}

// Sign signs a message with the private key
// This is a simplified implementation - in production use proper cryptography
func (pk *PrivateKey) Sign(msg []byte) ([]byte, error) {
	// Simple signing: hash(privkey + msg)
	// In production, use secp256k1 or ed25519
	h := sha256.New()
	h.Write(pk.bytes)
	h.Write(msg)
	return h.Sum(nil), nil
}

// GetPublicKey returns the public key
// Simplified - in production derive from private key properly
func (pk *PrivateKey) GetPublicKey() *PublicKey {
	// Simple public key derivation
	h := sha256.New()
	h.Write(pk.bytes)
	pubKeyBytes := h.Sum(nil)

	return &PublicKey{
		Type: "/cosmos.crypto.secp256k1.PubKey",
		Key:  encodeBase64(pubKeyBytes),
	}
}

// SignTransaction signs a transaction with a private key
func SignTransaction(tx *Transaction, privKey *PrivateKey) (*SignedTx, error) {
	// Get sign bytes
	signBytes, err := tx.GetSignBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get sign bytes: %w", err)
	}

	// Sign
	signature, err := privKey.Sign(signBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Build signed transaction
	messages := make([]json.RawMessage, len(tx.Msgs))
	for i, msg := range tx.Msgs {
		msgBytes := msg.GetSignBytes()
		messages[i] = json.RawMessage(msgBytes)
	}

	pubKey := privKey.GetPublicKey()

	signedTx := &SignedTx{
		Body: TxBody{
			Messages:      messages,
			Memo:          tx.Memo,
			TimeoutHeight: fmt.Sprintf("%d", tx.TimeoutHeight),
		},
		AuthInfo: AuthInfo{
			SignerInfos: []SignerInfo{
				{
					PublicKey: *pubKey,
					ModeInfo: ModeInfo{
						Single: Single{Mode: "SIGN_MODE_DIRECT"},
					},
					Sequence: fmt.Sprintf("%d", tx.Sequence),
				},
			},
			Fee: tx.Fee,
		},
		Signatures: []string{encodeBase64(signature)},
	}

	return signedTx, nil
}

// Serialize serializes a signed transaction to bytes for broadcasting
func (stx *SignedTx) Serialize() ([]byte, error) {
	return json.Marshal(stx)
}

// TransactionBuilder helps build transactions
type TransactionBuilder struct {
	tx *Transaction
}

// NewTransactionBuilder creates a new transaction builder
func NewTransactionBuilder(chainID string, accountNumber, sequence uint64) *TransactionBuilder {
	return &TransactionBuilder{
		tx: NewTransaction(chainID, accountNumber, sequence),
	}
}

// WithMemo adds a memo to the transaction
func (tb *TransactionBuilder) WithMemo(memo string) *TransactionBuilder {
	tb.tx.SetMemo(memo)
	return tb
}

// WithFee sets the transaction fee
func (tb *TransactionBuilder) WithFee(amount uint64, denom string, gas uint64) *TransactionBuilder {
	tb.tx.SetFee(amount, denom, gas)
	return tb
}

// WithSend adds a send message
func (tb *TransactionBuilder) WithSend(from, to string, amount uint64, denom string) *TransactionBuilder {
	msg := &MsgSend{
		FromAddress: from,
		ToAddress:   to,
		Amount:      []Coin{{Denom: denom, Amount: fmt.Sprintf("%d", amount)}},
	}
	tb.tx.AddMessage(msg)
	return tb
}

// Build returns the built transaction
func (tb *TransactionBuilder) Build() *Transaction {
	return tb.tx
}

// TransactionHistory tracks transaction history
type TransactionHistory struct {
	Transactions []HistoricalTransaction
}

// HistoricalTransaction represents a past transaction
type HistoricalTransaction struct {
	TxHash    string
	Timestamp time.Time
	Amount    uint64
	Memo      string
	Status    string
}

// AddTransaction adds a transaction to history
func (th *TransactionHistory) AddTransaction(tx HistoricalTransaction) {
	th.Transactions = append(th.Transactions, tx)
}

// GetRecent returns the N most recent transactions
func (th *TransactionHistory) GetRecent(n int) []HistoricalTransaction {
	if n > len(th.Transactions) {
		n = len(th.Transactions)
	}

	// Return last n transactions (most recent)
	start := len(th.Transactions) - n
	return th.Transactions[start:]
}
