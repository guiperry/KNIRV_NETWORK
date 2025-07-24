package storage

import (
    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/errors"
    "github.com/syndtr/goleveldb/leveldb/opt"
)

type LevelDBStorage struct {
    db *leveldb.DB
}

func NewLevelDBStorage(path string) (*LevelDBStorage, error) {
    db, err := leveldb.OpenFile(path, nil)
    if err != nil {
        return nil, err
    }
    
    return &LevelDBStorage{db: db}, nil
}

func (s *LevelDBStorage) Put(key, value []byte) error {
    return s.db.Put(key, value, nil)
}

func (s *LevelDBStorage) Get(key []byte) ([]byte, error) {
    data, err := s.db.Get(key, nil)
    if err != nil {
        if err == errors.ErrNotFound {
            return nil, ErrNotFound
        }
        return nil, err
    }
    return data, nil
}

func (s *LevelDBStorage) Delete(key []byte) error {
    return s.db.Delete(key, nil)
}

func (s *LevelDBStorage) Has(key []byte) (bool, error) {
    return s.db.Has(key, nil)
}

func (s *LevelDBStorage) Close() error {
    return s.db.Close()
}

func (s *LevelDBStorage) Batch() Batch {
    return &LevelDBBatch{batch: new(leveldb.Batch)}
}

type LevelDBBatch struct {
    batch *leveldb.Batch
}

func (b *LevelDBBatch) Put(key, value []byte) {
    b.batch.Put(key, value)
}

func (b *LevelDBBatch) Delete(key []byte) {
    b.batch.Delete(key)
}

func (b *LevelDBBatch) Write() error {
    return b.batch.Reset()
}