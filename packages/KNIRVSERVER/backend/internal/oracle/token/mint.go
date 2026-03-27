package token

import (
	"fmt"
	"math/big"

	"backend_server/internal/oracle/crypto"
	"backend_server/internal/oracle/types"
)

// MintRequest represents a token minting request
type MintRequest struct {
	To     types.Address `json:"to"`
	Amount *big.Int      `json:"amount"`
}

// MintReceipt represents a minting transaction receipt
type MintReceipt struct {
	To              types.Address `json:"to"`
	Amount          string        `json:"amount"`
	NewBalance      string        `json:"new_balance"`
	NewTotalSupply  string        `json:"new_total_supply"`
	TransactionHash string        `json:"transaction_hash"`
	Signature       string        `json:"signature"`
}

// Mint mints new tokens to an address
// Only the owner can mint tokens
func (n *NRN) Mint(toAddr types.Address, amount *big.Int) (*MintReceipt, error) {
	// Validate amount
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Check if minting would exceed max supply
	newTotalSupply := new(big.Int).Add(n.totalSupply, amount)
	if newTotalSupply.Cmp(n.maxSupply) > 0 {
		return nil, fmt.Errorf("minting %s tokens would exceed max supply of %s",
			amount.String(), n.maxSupply.String())
	}

	// Get current balance
	currentBalance := n.balances[toAddr]
	if currentBalance == nil {
		currentBalance = big.NewInt(0)
	}

	// Add minted amount to balance
	newBalance := new(big.Int).Add(currentBalance, amount)
	n.setBalance(toAddr, newBalance)

	// Update total supply
	n.totalSupply = newTotalSupply

	// Create transaction data
	txData := fmt.Sprintf("mint:%s:%s", toAddr.String(), amount.String())

	// Sign the transaction
	signature, err := n.ownerKey.Sign([]byte(txData))
	if err != nil {
		// Rollback on signing failure
		n.setBalance(toAddr, currentBalance)
		n.totalSupply.Sub(n.totalSupply, amount)
		return nil, fmt.Errorf("failed to sign mint transaction: %w", err)
	}

	// Generate transaction hash
	txHash := crypto.Keccak256HashWithPrefix([]byte(txData))

	return &MintReceipt{
		To:              toAddr,
		Amount:          amount.String(),
		NewBalance:      newBalance.String(),
		NewTotalSupply:  newTotalSupply.String(),
		TransactionHash: txHash,
		Signature:       fmt.Sprintf("0x%x", signature),
	}, nil
}

// MintTo is a convenience method for minting to an address string
func (n *NRN) MintTo(toAddrStr string, amount *big.Int) (*MintReceipt, error) {
	toAddr, err := types.AddressFromString(toAddrStr)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient address: %w", err)
	}

	return n.Mint(toAddr, amount)
}

// MintOwner mints tokens to the owner address
func (n *NRN) MintOwner(amount *big.Int) (*MintReceipt, error) {
	return n.Mint(n.owner, amount)
}
