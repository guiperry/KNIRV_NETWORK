package wallet

import (
	"context"
	"fmt"
)

type NRNWallet struct {
	balance uint64
}

func NewNRNWallet(key string) (*NRNWallet, error) {
	return &NRNWallet{balance: 1000000}, nil // Stub with 1M NRN
}

func (w *NRNWallet) Balance(ctx context.Context) (uint64, error) {
	return w.balance, nil
}

func (w *NRNWallet) HasBalance(ctx context.Context, required uint64) (bool, error) {
	return w.balance >= required, nil
}

func (w *NRNWallet) Spend(ctx context.Context, amount uint64, memo string) (string, error) {
	if w.balance < amount {
		return "", fmt.Errorf("insufficient balance")
	}
	w.balance -= amount
	return "tx_hash_stub", nil
}