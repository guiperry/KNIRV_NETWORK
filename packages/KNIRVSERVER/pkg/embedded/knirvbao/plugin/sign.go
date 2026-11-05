package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// signerService serializes policy checks, key selection and signing so a
// rotation or policy change cannot race a request. It is not exposed until
// the OpenBao routes and their audit behavior have been tested.
type signerService struct {
	mu sync.Mutex
}

// consumeRate persists the rolling budget before signing. A crash or restart
// therefore cannot reset it. The caller holds the service mutex.
func consumeRate(ctx context.Context, storage logical.Storage, name string, limit int, now time.Time) error {
	if !validName(name) || limit <= 0 {
		return errors.New("signing rate exceeded")
	}
	path := "limits/" + name
	entry, err := storage.Get(ctx, path)
	if err != nil {
		return err
	}
	var timestamps []time.Time
	if entry != nil {
		if err := entry.DecodeJSON(&timestamps); err != nil {
			return errors.New("invalid rate state")
		}
	}
	active := timestamps[:0]
	for _, stamp := range timestamps {
		if stamp.After(now.Add(-time.Minute)) {
			active = append(active, stamp)
		}
	}
	if len(active) >= limit {
		return errors.New("signing rate exceeded")
	}
	active = append(active, now.UTC())
	entry, err = logical.StorageEntryJSON(path, active)
	if err != nil {
		return err
	}
	return storage.Put(ctx, entry)
}

func loadPolicy(ctx context.Context, storage logical.Storage, name string) (Policy, error) {
	if !validName(name) {
		return Policy{}, errors.New("invalid key name")
	}
	entry, err := storage.Get(ctx, "keys/"+name+"/policy")
	if err != nil {
		return Policy{}, err
	}
	if entry == nil {
		return Policy{}, errors.New("signing policy not configured")
	}
	var policy Policy
	if err := entry.DecodeJSON(&policy); err != nil {
		return Policy{}, errors.New("invalid signing policy")
	}
	return policy, nil
}

func (s *signerService) setPolicy(ctx context.Context, storage logical.Storage, name string, policy Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := readMetadata(ctx, storage, name); err != nil {
		return err
	}
	if err := policy.validate(); err != nil {
		return err
	}
	entry, err := logical.StorageEntryJSON("keys/"+name+"/policy", policy)
	if err != nil {
		return err
	}
	return storage.Put(ctx, entry)
}

func (s *signerService) direct(ctx context.Context, storage logical.Storage, name string, req signing.SignRequest, now time.Time) (signing.WireTransaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	policy, err := loadPolicy(ctx, storage, name)
	if err != nil {
		return signing.WireTransaction{}, err
	}
	if err := policy.allowTransaction(req); err != nil {
		return signing.WireTransaction{}, err
	}
	if err := consumeRate(ctx, storage, name, policy.RateLimit, now); err != nil {
		return signing.WireTransaction{}, err
	}
	meta, err := readMetadata(ctx, storage, name)
	if err != nil {
		return signing.WireTransaction{}, err
	}
	key, err := readVersion(ctx, storage, name, meta.Version)
	if err != nil {
		return signing.WireTransaction{}, err
	}
	if key.Type != meta.Type {
		return signing.WireTransaction{}, errors.New("stored key type mismatch")
	}
	tx, err := signing.SignTransaction(key.PrivateKey, req)
	if err != nil {
		return signing.WireTransaction{}, err
	}
	return tx.Wire(), nil
}

func (s *signerService) message(ctx context.Context, storage logical.Storage, name string, msg signing.MessageEnvelope, now time.Time) (signing.SignedMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	policy, err := loadPolicy(ctx, storage, name)
	if err != nil {
		return signing.SignedMessage{}, err
	}
	if err := policy.allowMessage(msg); err != nil {
		return signing.SignedMessage{}, err
	}
	if err := consumeRate(ctx, storage, name, policy.RateLimit, now); err != nil {
		return signing.SignedMessage{}, err
	}
	meta, err := readMetadata(ctx, storage, name)
	if err != nil {
		return signing.SignedMessage{}, err
	}
	key, err := readVersion(ctx, storage, name, meta.Version)
	if err != nil {
		return signing.SignedMessage{}, err
	}
	if key.Type != meta.Type {
		return signing.SignedMessage{}, errors.New("stored key type mismatch")
	}
	return signing.SignMessage(key.PrivateKey, msg)
}
