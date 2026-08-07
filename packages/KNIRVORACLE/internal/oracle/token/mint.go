package token

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
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

	now := time.Now()
	signed, err := knirvsigning.SignMessage(ethcrypto.FromECDSA(n.ownerKey.PrivateKey), knirvsigning.MessageEnvelope{
		Domain: "knirv.oracle", Purpose: "oracle-mint", ChainID: n.chainID,
		Nonce: fmt.Sprintf("mint:%d", now.UnixNano()), IssuedAtUnix: now.Unix(), ExpiresAtUnix: now.Add(5 * time.Minute).Unix(),
		Payload: []byte(txData),
	})
	if err != nil {
		// Rollback on signing failure
		n.setBalance(toAddr, currentBalance)
		n.totalSupply.Sub(n.totalSupply, amount)
		return nil, fmt.Errorf("failed to sign mint transaction: %w", err)
	}

	txHashBytes := sha256.Sum256(signed.Envelope)
	txHash := fmt.Sprintf("%X", txHashBytes)
	signedJSON, _ := json.Marshal(signed)

	return &MintReceipt{
		To:              toAddr,
		Amount:          amount.String(),
		NewBalance:      newBalance.String(),
		NewTotalSupply:  newTotalSupply.String(),
		TransactionHash: txHash,
		Signature:       string(signedJSON),
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
