package main

import (
	"errors"
	"math/big"
	"slices"

	"github.com/guiperry/knirv-sdk-go/signing"
)

// Policy is deny-by-default: an empty allowlist grants no signing authority.
type Policy struct {
	TypeURLs  []string      `json:"type_urls"`
	ChainIDs  []string      `json:"chain_ids"`
	Messages  []MessagePair `json:"messages"`
	MaxFee    string        `json:"max_fee"`
	RateLimit int           `json:"rate_limit"`
}

type MessagePair struct {
	Domain  string `json:"domain"`
	Purpose string `json:"purpose"`
}

func (p Policy) validate() error {
	if p.RateLimit <= 0 {
		return errors.New("rate limit must be positive")
	}
	fee, ok := new(big.Int).SetString(p.MaxFee, 10)
	if !ok || fee.Sign() < 0 {
		return errors.New("invalid maximum fee")
	}
	for _, chain := range p.ChainIDs {
		if chain == "" {
			return errors.New("empty chain id")
		}
	}
	for _, url := range p.TypeURLs {
		if url == "" {
			return errors.New("empty type URL")
		}
	}
	for _, pair := range p.Messages {
		if pair.Domain == "" || pair.Purpose == "" {
			return errors.New("empty message domain or purpose")
		}
	}
	return nil
}

func (p Policy) allowTransaction(req signing.SignRequest) error {
	if !slices.Contains(p.TypeURLs, signing.ActionTypeURL) || !slices.Contains(p.ChainIDs, req.ChainID) {
		return errors.New("transaction type or chain is not allowed")
	}
	amount := req.Fee.Amount
	if amount == "" {
		amount = "0"
	}
	fee, ok := new(big.Int).SetString(amount, 10)
	if !ok || fee.Sign() < 0 {
		return errors.New("invalid fee")
	}
	limit, ok := new(big.Int).SetString(p.MaxFee, 10)
	if !ok || limit.Sign() < 0 || fee.Cmp(limit) > 0 {
		return errors.New("fee exceeds policy")
	}
	return nil
}

func (p Policy) allowMessage(msg signing.MessageEnvelope) error {
	if !slices.Contains(p.ChainIDs, msg.ChainID) || !slices.Contains(p.Messages, MessagePair{msg.Domain, msg.Purpose}) {
		return errors.New("message domain, purpose or chain is not allowed")
	}
	return nil
}
