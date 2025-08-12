package types

import (
	"encoding/json"
	"fmt"
	"sync"
)

type State struct {
	mu       sync.RWMutex
	Accounts map[string]*Account `json:"accounts"`
	Height   uint64              `json:"height"`
}

type Account struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

func NewState() *State {
	return &State{
		Accounts: make(map[string]*Account),
		Height:   0,
	}
}

func (s *State) GetAccount(address string) *Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getAccountUnsafe(address)
}

// getAccountUnsafe retrieves an account without acquiring locks (assumes caller holds appropriate lock)
func (s *State) getAccountUnsafe(address string) *Account {
	if account, exists := s.Accounts[address]; exists {
		return account
	}

	return &Account{
		Address: address,
		Balance: 0,
		Nonce:   0,
	}
}

func (s *State) SetAccount(account *Account) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Accounts[account.Address] = account
}

func (s *State) Transfer(from, to string, amount uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fromAccount := s.getAccountUnsafe(from)
	toAccount := s.getAccountUnsafe(to)

	if fromAccount.Balance < amount {
		return ErrInsufficientBalance
	}

	fromAccount.Balance -= amount
	toAccount.Balance += amount

	s.Accounts[from] = fromAccount
	s.Accounts[to] = toAccount

	return nil
}

func (s *State) Serialize() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return json.Marshal(s)
}

var (
	ErrInsufficientBalance = fmt.Errorf("insufficient balance")
)
