package token

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	knirvsigning "github.com/KNIRV/KNIRV_NETWORK/KNIRVSDK/go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
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
	return nil, fmt.Errorf("private-key burn is disabled; submit a KNIRVCONTROLLER SignedMessage to BurnSigned")
}

// BurnSigned burns tokens after verifying a client-side signature.
func (n *NRN) BurnSigned(fromAddr types.Address, amount *big.Int, reason string, nonce uint64, signed knirvsigning.SignedMessage) (*BurnReceipt, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	txData := fmt.Sprintf("burn:%s:%s:%s:%d", fromAddr.String(), amount.String(), reason, nonce)
	if err := knirvsigning.VerifyMessagePayload(signed, "knirv.oracle", "oracle-burn", n.chainID, []byte(txData), time.Now()); err != nil {
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

	newBalance := new(big.Int).Sub(fromBalance, amount)
	newTotalSupply := new(big.Int).Sub(n.totalSupply, amount)
	n.setBalance(fromAddr, newBalance)
	n.totalSupply = newTotalSupply
	n.nonces[fromAddr] = expectedNonce + 1

	txHashBytes := sha256.Sum256(signed.Envelope)
	txHash := fmt.Sprintf("%X", txHashBytes)
	signedJSON, _ := json.Marshal(signed)

	return &BurnReceipt{
		From:            fromAddr,
		Amount:          amount.String(),
		NewBalance:      newBalance.String(),
		NewTotalSupply:  newTotalSupply.String(),
		Reason:          reason,
		TransactionHash: txHash,
		Signature:       string(signedJSON),
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

// BurnForSkillInvocation burns tokens as the cost of canonicalizing or
// invoking a skill (internal use, no signature required — same custody
// model as BurnFrom/BurnForFee: the caller, KNIRVORACLE's
// /oracle/v3/skills/burn route, is itself gated by the internal service
// auth token, not a per-user signature).
func (n *NRN) BurnForSkillInvocation(fromAddr types.Address, skillID string, amount *big.Int) (*BurnReceipt, error) {
	if err := validateAmount(amount); err != nil {
		return nil, err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	fromBalance := n.balances[fromAddr]
	if fromBalance == nil {
		fromBalance = big.NewInt(0)
	}
	if fromBalance.Cmp(amount) < 0 {
		return nil, fmt.Errorf("%w: have %s, need %s",
			types.ErrInsufficientBalance, fromBalance.String(), amount.String())
	}

	reason := fmt.Sprintf("skill-invocation:%s", skillID)
	newBalance := new(big.Int).Sub(fromBalance, amount)
	newTotalSupply := new(big.Int).Sub(n.totalSupply, amount)
	n.setBalance(fromAddr, newBalance)
	n.totalSupply = newTotalSupply

	txHashBytes := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s:%d", fromAddr.String(), amount.String(), reason, time.Now().UnixNano())))

	return &BurnReceipt{
		From:            fromAddr,
		Amount:          amount.String(),
		NewBalance:      newBalance.String(),
		NewTotalSupply:  newTotalSupply.String(),
		Reason:          reason,
		TransactionHash: fmt.Sprintf("%X", txHashBytes),
	}, nil
}

// BurnForFee burns tokens as a fee payment
func (n *NRN) BurnForFee(fromAddr types.Address, amount *big.Int, feeType string) error {
	reason := fmt.Sprintf("fee:%s", feeType)
	return n.BurnFrom(fromAddr, amount, reason)
}
