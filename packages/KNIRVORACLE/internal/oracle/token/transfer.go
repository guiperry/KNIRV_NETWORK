package token

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
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
	return nil, fmt.Errorf("private-key transfer is disabled; submit a KNIRVCONTROLLER SignedMessage to TransferSigned")
}

// TransferSigned transfers tokens after verifying a client-side signature.
func (n *NRN) TransferSigned(fromAddr, toAddr types.Address, amount *big.Int, nonce uint64, signed knirvsigning.SignedMessage) (*TransferReceipt, error) {
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
	if err := knirvsigning.VerifyMessagePayload(signed, "knirv.oracle", "oracle-transfer", n.chainID, []byte(txData), time.Now()); err != nil {
		return nil, fmt.Errorf("invalid canonical signature: %w", err)
	}
	if signed.Address != fromAddr.String() {
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

	txHashBytes := sha256.Sum256(signed.Envelope)
	txHash := fmt.Sprintf("%X", txHashBytes)
	signedJSON, _ := json.Marshal(signed)

	return &TransferReceipt{
		From:            fromAddr,
		To:              toAddr,
		Amount:          amount.String(),
		FromNewBalance:  newFromBalance.String(),
		ToNewBalance:    newToBalance.String(),
		TransactionHash: txHash,
		Signature:       string(signedJSON),
	}, nil
}

// LockGovernanceDeposit verifies a Controller-approved proposal authorization
// and atomically moves its deposit into the governance escrow account.
func (n *NRN) LockGovernanceDeposit(fromAddr, escrowAddr types.Address, amount *big.Int, nonce uint64, authorizationPayload []byte, signed knirvsigning.SignedMessage) error {
	if err := validateAmount(amount); err != nil {
		return err
	}
	if fromAddr.IsZero() || escrowAddr.IsZero() || fromAddr == escrowAddr {
		return fmt.Errorf("invalid governance deposit account")
	}
	if err := knirvsigning.VerifyMessagePayload(signed, "knirv.oracle", "oracle-governance-proposal", n.chainID, authorizationPayload, time.Now()); err != nil {
		return fmt.Errorf("invalid canonical signature: %w", err)
	}
	if signed.Address != fromAddr.String() {
		return fmt.Errorf("signature does not match proposer address")
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	expectedNonce := n.nonces[fromAddr]
	if nonce != expectedNonce {
		return fmt.Errorf("invalid nonce: got %d, want %d", nonce, expectedNonce)
	}
	fromBalance := n.balances[fromAddr]
	if fromBalance == nil || fromBalance.Cmp(amount) < 0 {
		return fmt.Errorf("%w: have %s, need %s", types.ErrInsufficientBalance, valueOrZero(fromBalance), amount)
	}
	escrowBalance := n.balances[escrowAddr]
	if escrowBalance == nil {
		escrowBalance = big.NewInt(0)
	}
	n.setBalance(fromAddr, new(big.Int).Sub(fromBalance, amount))
	n.setBalance(escrowAddr, new(big.Int).Add(escrowBalance, amount))
	n.nonces[fromAddr] = expectedNonce + 1
	return nil
}

// TransferFrom is a convenience method for string addresses
func (n *NRN) TransferFrom(fromPrivateKey string, toAddrStr string, amount *big.Int) (*TransferReceipt, error) {
	return nil, fmt.Errorf("private-key transfer is disabled; submit a KNIRVCONTROLLER SignedMessage")
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
