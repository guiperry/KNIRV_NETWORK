package transaction

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// Wallet is an explicitly provisioned unattended service identity. User
// wallets are held and approved by KNIRVCONTROLLER and must not be generated
// inside a headless SDK process.
type Wallet struct {
	address    string
	privateKey []byte
	publicKey  []byte
}

func (w *Wallet) GetAddress() string { return w.address }

func (w *Wallet) GetPublicKeyHex() string { return hex.EncodeToString(w.publicKey) }

// NewWallet is retained only to fail closed for callers of the obsolete API.
func NewWallet() (*Wallet, error) {
	return nil, errors.New("implicit wallet generation is disabled; use KNIRVCONTROLLER for user custody or NewServiceWallet with a registered service key")
}

// NewServiceWallet loads an operator-provisioned secp256k1 key and requires
// its canonical address to match the address registered for that service.
func NewServiceWallet(privateKeyHex, registeredAddress string) (*Wallet, error) {
	raw, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x"))
	if err != nil || len(raw) != 32 {
		return nil, errors.New("service private key must be 32-byte hex")
	}
	key := secp256k1.PrivKeyFromBytes(raw)
	if key.Key.IsZero() {
		return nil, errors.New("service private key cannot be zero")
	}
	pub := key.PubKey().SerializeCompressed()
	address, err := knirvsigning.Address(pub, knirvsigning.DefaultAddressPrefix)
	if err != nil {
		return nil, err
	}
	if registeredAddress == "" || address != registeredAddress {
		return nil, fmt.Errorf("service key address %s does not match registered address %s", address, registeredAddress)
	}
	return &Wallet{address: address, privateKey: append([]byte(nil), raw...), publicKey: pub}, nil
}

// SignTransaction applies KNIRV's canonical Cosmos SIGN_MODE_DIRECT wire
// contract and writes the complete TxRaw components into txn.
func (w *Wallet) SignTransaction(txn *Transaction, chainID string, accountNumber, sequence uint64) error {
	if w == nil || len(w.privateKey) != 32 {
		return errors.New("registered service private key is not available")
	}
	if txn == nil {
		return errors.New("transaction is required")
	}
	if txn.From != "" && txn.From != w.address {
		return errors.New("transaction sender does not match service key")
	}
	amount, err := strconv.ParseUint(defaultDecimal(txn.Value), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid transaction value: %w", err)
	}
	fee, err := strconv.ParseUint(defaultDecimal(txn.Fee), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid transaction fee: %w", err)
	}
	var payload []byte
	if txn.Data != nil {
		payload, err = json.Marshal(txn.Data)
		if err != nil {
			return fmt.Errorf("marshal transaction payload: %w", err)
		}
	}
	signed, err := knirvsigning.SignTransaction(w.privateKey, knirvsigning.SignRequest{
		Action: knirvsigning.Action{
			Action: txn.Type, Sender: w.address, Recipient: txn.To,
			Amount: amount, Payload: payload, TimestampUnix: txn.Timestamp,
		},
		ChainID: chainID, AccountNumber: accountNumber, Sequence: sequence,
		Fee: knirvsigning.Fee{Denom: "uknirv", Amount: strconv.FormatUint(fee, 10)},
	})
	if err != nil {
		return err
	}
	txn.From = w.address
	txn.BodyBytes = base64.StdEncoding.EncodeToString(signed.BodyBytes)
	txn.AuthInfoBytes = base64.StdEncoding.EncodeToString(signed.AuthInfoBytes)
	txn.Signatures = []string{base64.StdEncoding.EncodeToString(signed.Signatures[0])}
	txn.Signature = txn.Signatures[0]
	txn.PublicKey = base64.StdEncoding.EncodeToString(signed.PublicKey)
	txn.ChainID = chainID
	txn.AccountNumber = strconv.FormatUint(accountNumber, 10)
	txn.TransactionHash = strings.ToUpper(hex.EncodeToString(signed.Hash))
	txn.ID = txn.TransactionHash
	return nil
}

func defaultDecimal(value string) string {
	if strings.TrimSpace(value) == "" {
		return "0"
	}
	return value
}

type TransactionType string

const (
	TransactionTypeMCPRegisterCapability TransactionType = "mcp_register_capability"
	TransactionTypeTransfer              TransactionType = "transfer"
	TransactionTypeMCPInvokeCapability   TransactionType = "mcp_invoke_capability"
	TransactionTypeMCPUpdateCapability   TransactionType = "mcp_update_capability"
)

type ResourceType string

const (
	ResourceTypePlugin ResourceType = "plugin"
	ResourceTypeData   ResourceType = "data"
	ResourceTypeCode   ResourceType = "code"
)

type ResourceDescriptor struct {
	ID           string       `json:"id"`
	ResourceType ResourceType `json:"resource_type"`
	ContentHash  string       `json:"content_hash"`
	Schema       struct {
		LocationHints []string `json:"location_hints"`
	} `json:"schema"`
}

func NewMCPTransaction(from, to string, value uint64, data []byte, txType TransactionType, fee uint64, timestamp int64) *Transaction {
	return &Transaction{
		From: from, To: to, Value: strconv.FormatUint(value, 10), Type: string(txType),
		Fee: strconv.FormatUint(fee, 10), Timestamp: timestamp, Version: 1,
		Data: map[string]any{"payload": base64.StdEncoding.EncodeToString(data)},
	}
}
