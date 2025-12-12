package token

import (
	"fmt"
	"math/big"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/oracle/crypto"
	"github.com/KNIRV/KNIRV_NETWORK/KNIRVORACLE/internal/oracle/types"
)

// BurnRequest represents a token burning request
type BurnRequest struct {
	From   types.Address `json:"from"`
	Amount *big.Int      `json:"amount"`
	Reason string        `json:"reason,omitempty"`
}

// BurnReceipt represents a burn transaction receipt
type BurnReceipt struct {
	From            types.Address `json:"from"`
	Amount          string        `json:"amount"`
	NewBalance      string        `json:"new_balance"`
	NewTotalSupply  string        `json:"new_total_supply"`
	Reason          string        `json:"reason,omitempty"`
	TransactionHash string        `json:"transaction_hash"`
	Signature       string        `json:"signature"`
}

// Burn burns tokens from an address
func (n *NRN) Burn(fromPrivateKey string, amount *big.Int, reason string) (*BurnReceipt, error) {
	// Validate amount
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	// Get burner key pair and address
	fromKeyPair, err := crypto.PrivateKeyFromHex(fromPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	fromAddr := fromKeyPair.Address

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check balance
	fromBalance := n.balances[fromAddr]
	if fromBalance == nil {
		fromBalance = big.NewInt(0)
	}

	if fromBalance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("%w: have %s, need %s",
			types.ErrInsufficientBalance, fromBalance.String(), amount.String())
	}

	// Calculate new balance and total supply
	newBalance := new(big.Int).Sub(fromBalance, amount)
	newTotalSupply := new(big.Int).Sub(n.totalSupply, amount)

	// Create transaction data
	txData := fmt.Sprintf("burn:%s:%s:%s", fromAddr.String(), amount.String(), reason)

	// Sign the transaction
	signature, err := fromKeyPair.Sign([]byte(txData))
	if err != nil {
		return nil, fmt.Errorf("failed to sign burn transaction: %w", err)
	}

	// Update balance and total supply
	n.setBalance(fromAddr, newBalance)
	n.totalSupply = newTotalSupply

	// Generate transaction hash
	txHash := crypto.Keccak256HashWithPrefix([]byte(txData))

	return &BurnReceipt{
		From:            fromAddr,
		Amount:          amount.String(),
		NewBalance:      newBalance.String(),
		NewTotalSupply:  newTotalSupply.String(),
		Reason:          reason,
		TransactionHash: txHash,
		Signature:       fmt.Sprintf("0x%x", signature),
	}, nil
}

// BurnFrom burns tokens from an address (internal use, no signature required)
func (n *NRN) BurnFrom(addr types.Address, amount *big.Int, reason string) error {
	// Validate amount
	if err := validateAmount(amount); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check balance
	balance := n.balances[addr]
	if balance == nil {
		balance = big.NewInt(0)
	}

	if balance.Cmp(amount) < 0 {
		return types.ErrInsufficientBalance
	}

	// Calculate new balance and total supply
	newBalance := new(big.Int).Sub(balance, amount)
	n.totalSupply.Sub(n.totalSupply, amount)

	// Update balance
	n.setBalance(addr, newBalance)

	return nil
}

// BurnForSkillInvocation burns tokens for a skill invocation
// This is a specialized burn function for the KNIRV network's skill system
func (n *NRN) BurnForSkillInvocation(fromPrivateKey, skillID string, amount *big.Int) (*BurnReceipt, error) {
	reason := fmt.Sprintf("skill_invocation:%s", skillID)
	return n.Burn(fromPrivateKey, amount, reason)
}

// BurnForFee burns tokens as a fee payment
func (n *NRN) BurnForFee(fromAddr types.Address, amount *big.Int, feeType string) error {
	reason := fmt.Sprintf("fee:%s", feeType)
	return n.BurnFrom(fromAddr, amount, reason)
}
