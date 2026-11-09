package database

import (
	"encoding/binary"
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const keyPrefixAccountSequence = "account_sequence:"

// SaveAccountSequence stores the next sequence an account must sign with.
func (db *LevelDB) SaveAccountSequence(address string, sequence uint64) error {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, sequence)
	return db.PutBytes(keyPrefixAccountSequence+address, data)
}

// GetAccountSequence returns the next sequence an account must sign with. An account
// that has never transacted is at 0.
func (db *LevelDB) GetAccountSequence(address string) (uint64, error) {
	data, err := db.Client.Get([]byte(keyPrefixAccountSequence+address), &opt.ReadOptions{})
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get account sequence: %w", err)
	}
	if len(data) != 8 {
		return 0, fmt.Errorf("invalid account sequence data length")
	}
	return binary.BigEndian.Uint64(data), nil
}
