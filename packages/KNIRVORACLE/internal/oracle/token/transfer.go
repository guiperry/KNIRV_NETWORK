package token

import (
	"fmt"
	"math/big"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// TransferRequest represents a token transfer request
type TransferRequest struct {
	From   types.Address `json:"from"`
	To     types.Address `json:"to"`
	Amount *big.Int      `json:"amount"`
}

// TransferReceipt represents a transfer transaction receipt
type TransferReceipt struct {
	From            types.Address `json:"from"`
	To              types.Address `json:"to"`
	Amount          string        `json:"amount"`
	FromNewBalance  string        `json:"from_new_balance"`
	ToNewBalance    string        `json:"to_new_balance"`
	TransactionHash string        `json:"transaction_hash"`
	Signature       string        `json:"signature"`
}

// Transfer transfers tokens from one address to another
func (n *NRN) Transfer(fromPrivateKey string, toAddr types.Address, amount *big.Int) (*TransferReceipt, error) {
	// Validate amount
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	// Get sender key pair and address
	fromKeyPair, err := crypto.PrivateKeyFromHex(fromPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid sender private key: %w", err)
	}
	fromAddr := fromKeyPair.Address

	// Don't allow transfers to zero address
	if toAddr.IsZero() {
		return nil, fmt.Errorf("cannot transfer to zero address")
	}

	// Don't allow self-transfer
	if fromAddr == toAddr {
		return nil, fmt.Errorf("cannot transfer to self")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check sender balance
	fromBalance := n.balances[fromAddr]
	if fromBalance == nil {
		fromBalance = big.NewInt(0)
	}

	if fromBalance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("%w: have %s, need %s",
			types.ErrInsufficientBalance, fromBalance.String(), amount.String())
	}

	// Get recipient balance
	toBalance := n.balances[toAddr]
	if toBalance == nil {
		toBalance = big.NewInt(0)
	}

	// Calculate new balances
	newFromBalance := new(big.Int).Sub(fromBalance, amount)
	newToBalance := new(big.Int).Add(toBalance, amount)

	// Create transaction data
	txData := fmt.Sprintf("transfer:%s:%s:%s", fromAddr.String(), toAddr.String(), amount.String())

	// Sign the transaction
	signature, err := fromKeyPair.Sign([]byte(txData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign transfer transaction: %w", err)
	}

	// Update balances
	n.setBalance(fromAddr, newFromBalance)
	n.setBalance(toAddr, newToBalance)

	// Generate transaction hash
	txHash := crypto.Keccak256HashWithPrefix([]byte(txData))

	return &TransferReceipt{
		From:            fromAddr,
		To:              toAddr,
		Amount:          amount.String(),
		FromNewBalance:  newFromBalance.String(),
		ToNewBalance:    newToBalance.String(),
		TransactionHash: txHash,
		Signature:       fmt.Sprintf("0x%x", signature),
	}, nil
}

// TransferSigned transfers tokens after verifying a client-side signature.
func (n *NRN) TransferSigned(fromAddr, toAddr types.Address, amount *big.Int, nonce uint64, signature []byte) (*TransferReceipt, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}
	if toAddr.IsZero() {
		return nil, fmt.Errorf("cannot transfer to zero address")
	}
	if fromAddr == toAddr {
		return nil, fmt.Errorf("cannot transfer to self")
	}

	txData := fmt.Sprintf("transfer:%s:%s:%s:%d", fromAddr.String(), toAddr.String(), amount.String(), nonce)
	recoveredAddr, err := crypto.RecoverAddress([]byte(txData), signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}
	if recoveredAddr != fromAddr {
		return nil, fmt.Errorf("signature does not match from address")
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	expectedNonce := n.nonces[fromAddr]
	if nonce != expectedNonce {
		return nil, fmt.Errorf("invalid nonce: got %d, want %d", nonce, expectedNonce)
	}

	fromBalance := n.balances[fromAddr]
	if fromBalance == nil {
		fromBalance = big.NewInt(0)
	}
	if fromBalance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("%w: have %s, need %s",
			types.ErrInsufficientBalance, fromBalance.String(), amount.String())
	}

	toBalance := n.balances[toAddr]
	if toBalance == nil {
		toBalance = big.NewInt(0)
	}

	newFromBalance := new(big.Int).Sub(fromBalance, amount)
	newToBalance := new(big.Int).Add(toBalance, amount)
	n.setBalance(fromAddr, newFromBalance)
	n.setBalance(toAddr, newToBalance)
	n.nonces[fromAddr] = expectedNonce + 1

	txHash := crypto.Keccak256HashWithPrefix([]byte(txData))

	return &TransferReceipt{
		From:            fromAddr,
		To:              toAddr,
		Amount:          amount.String(),
		FromNewBalance:  newFromBalance.String(),
		ToNewBalance:    newToBalance.String(),
		TransactionHash: txHash,
		Signature:       fmt.Sprintf("0x%x", signature),
	}, nil
}

// TransferFrom is a convenience method for string addresses
func (n *NRN) TransferFrom(fromPrivateKey string, toAddrStr string, amount *big.Int) (*TransferReceipt, error) {
	toAddr, err := types.AddressFromString(toAddrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	return n.Transfer(fromPrivateKey, toAddr, amount)
}

// TransferBetween transfers tokens between two addresses (internal use)
func (n *NRN) TransferBetween(fromAddr, toAddr types.Address, amount *big.Int) error {
	// Validate amount
	if err := validateAmount(amount); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check sender balance
	fromBalance := n.balances[fromAddr]
	if fromBalance == nil {
		fromBalance = big.NewInt(0)
	}

	if fromBalance.Cmp(amount) < 0 {
		return types.ErrInsufficientBalance
	}

	// Get recipient balance
	toBalance := n.balances[toAddr]
	if toBalance == nil {
		toBalance = big.NewInt(0)
	}

	// Calculate new balances
	newFromBalance := new(big.Int).Sub(fromBalance, amount)
	newToBalance := new(big.Int).Add(toBalance, amount)

	// Update balances
	n.setBalance(fromAddr, newFromBalance)
	n.setBalance(toAddr, newToBalance)

	return nil
}
