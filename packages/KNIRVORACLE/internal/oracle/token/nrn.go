package token

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"github.com/knirvcorp/knirvoracle/internal/oracle/types"
)

// NRN represents the KNIRV Network Token
type NRN struct {
	name        string
	symbol      string
	decimals    uint8
	totalSupply *big.Int
	maxSupply   *big.Int
	balances    map[types.Address]*big.Int
	nonces      map[types.Address]uint64
	registered  map[types.Address]bool
	owner       types.Address
	ownerKey    *crypto.KeyPair
	mu          sync.RWMutex
}

// TokenInfo contains basic token information
type TokenInfo struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	TotalSupply string `json:"total_supply"`
	MaxSupply   string `json:"max_supply"`
	Owner       string `json:"owner"`
}

// NewNRN creates a new NRN token instance
func NewNRN(name, symbol string, initialSupply, maxSupply *big.Int, ownerPrivateKey string) (*NRN, error) {
	// Validate parameters
	if initialSupply.Cmp(maxSupply) > 0 {
		return nil, fmt.Errorf("initial supply cannot exceed max supply")
	}

	if initialSupply.Sign() < 0 || maxSupply.Sign() < 0 {
		return nil, fmt.Errorf("supply values must be non-negative")
	}

	// Load owner key pair
	keyPair, err := crypto.PrivateKeyFromHex(ownerPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load owner key: %w", err)
	}

	// Initialize NRN token
	nrn := &NRN{
		name:        name,
		symbol:      symbol,
		decimals:    18, // Standard ERC20 decimals
		totalSupply: new(big.Int).Set(initialSupply),
		maxSupply:   new(big.Int).Set(maxSupply),
		balances:    make(map[types.Address]*big.Int),
		nonces:      make(map[types.Address]uint64),
		registered:  make(map[types.Address]bool),
		owner:       keyPair.Address,
		ownerKey:    keyPair,
	}

	// Mint initial supply to owner
	nrn.balances[keyPair.Address] = new(big.Int).Set(initialSupply)

	return nrn, nil
}

// Info returns basic token information
func (n *NRN) Info() TokenInfo {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return TokenInfo{
		Name:        n.name,
		Symbol:      n.symbol,
		Decimals:    n.decimals,
		TotalSupply: n.totalSupply.String(),
		MaxSupply:   n.maxSupply.String(),
		Owner:       n.owner.String(),
	}
}

// Name returns the token name
func (n *NRN) Name() string {
	return n.name
}

// Symbol returns the token symbol
func (n *NRN) Symbol() string {
	return n.symbol
}

// Decimals returns the number of decimals
func (n *NRN) Decimals() uint8 {
	return n.decimals
}

// TotalSupply returns the current total supply
func (n *NRN) TotalSupply() *big.Int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return new(big.Int).Set(n.totalSupply)
}

// MaxSupply returns the maximum supply
func (n *NRN) MaxSupply() *big.Int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return new(big.Int).Set(n.maxSupply)
}

// Owner returns the owner address
func (n *NRN) Owner() types.Address {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.owner
}

// GetBalance returns the balance of an address
func (n *NRN) GetBalance(addr types.Address) *big.Int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	balance, exists := n.balances[addr]
	if !exists {
		return big.NewInt(0)
	}
	return new(big.Int).Set(balance)
}

// GetBalanceString returns the balance as a string
func (n *NRN) GetBalanceString(addr types.Address) string {
	return n.GetBalance(addr).String()
}

// GetNonce returns the next nonce expected for a signed request from an address.
func (n *NRN) GetNonce(addr types.Address) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.nonces[addr]
}

// RegisterAddress marks an address as registered and returns true only the first time.
func (n *NRN) RegisterAddress(addr types.Address) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.registered[addr] {
		return false
	}
	n.registered[addr] = true
	return true
}

// setBalance sets the balance for an address (internal use only, must hold lock)
func (n *NRN) setBalance(addr types.Address, amount *big.Int) {
	if amount.Sign() == 0 {
		delete(n.balances, addr)
	} else {
		n.balances[addr] = new(big.Int).Set(amount)
	}
}

// validateAmount checks if an amount is valid
func validateAmount(amount *big.Int) error {
	if amount == nil {
		return types.ErrInvalidAmount
	}
	if amount.Sign() < 0 {
		return fmt.Errorf("%w: negative amount", types.ErrInvalidAmount)
	}
	if amount.Sign() == 0 {
		return fmt.Errorf("%w: zero amount", types.ErrInvalidAmount)
	}
	return nil
}
