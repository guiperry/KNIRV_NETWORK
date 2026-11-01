package ledger

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Record is one log entry. Hash chains records: changing or removing an earlier
// record invalidates every later hash, which is what makes the log auditable.
type Record struct {
	Seq      uint64          `json:"seq"`
	Kind     string          `json:"kind"`
	Payload  json.RawMessage `json:"payload"`
	PrevHash string          `json:"prev_hash"`
	Hash     string          `json:"hash"`
}

// Log is an append-only JSONL file. One process owns it; appends are serialized
// and fsync'd before Append returns, so a record the caller has acknowledged
// cannot be missing after a crash.
type Log struct {
	path     string
	mu       sync.Mutex
	file     *os.File
	records  []Record
	lastHash string
}

// Open reads an existing log (if any) and opens it for appending. A malformed
// line is an error, never silently skipped: replaying a truncated log would
// produce different state without saying so.
func Open(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("ledger: create dir: %w", err)
	}

	records, lastHash, err := readRecords(path)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("ledger: open %s: %w", path, err)
	}
	return &Log{path: path, file: file, records: records, lastHash: lastHash}, nil
}

func readRecords(path string) ([]Record, string, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("ledger: read %s: %w", path, err)
	}
	defer file.Close()

	var records []Record
	last := ""
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, "", fmt.Errorf("ledger: %s line %d: %w", path, line, err)
		}
		want := hashRecord(r.Seq, r.Kind, r.Payload, r.PrevHash)
		if r.Hash != want {
			return nil, "", fmt.Errorf("ledger: %s line %d: hash mismatch (record altered or truncated)", path, line)
		}
		if r.Seq != uint64(len(records))+1 || r.PrevHash != last {
			return nil, "", fmt.Errorf("ledger: %s line %d: broken chain (seq %d, prev %q)", path, line, r.Seq, r.PrevHash)
		}
		records = append(records, r)
		last = r.Hash
	}
	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("ledger: scan %s: %w", path, err)
	}
	return records, last, nil
}

// Append writes and fsyncs one record. The caller applies it to the read model
// only after Append returns.
func (l *Log) Append(kind string, payload any) (Record, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return Record{}, fmt.Errorf("ledger: encode %s: %w", kind, err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	seq := uint64(len(l.records)) + 1
	hash := hashRecord(seq, kind, body, l.lastHash)
	record := Record{Seq: seq, Kind: kind, Payload: body, PrevHash: l.lastHash, Hash: hash}

	line, err := json.Marshal(record)
	if err != nil {
		return Record{}, fmt.Errorf("ledger: encode record: %w", err)
	}
	if _, err := l.file.Write(append(line, '\n')); err != nil {
		return Record{}, fmt.Errorf("ledger: append: %w", err)
	}
	if err := l.file.Sync(); err != nil {
		return Record{}, fmt.Errorf("ledger: fsync: %w", err)
	}

	l.records = append(l.records, record)
	l.lastHash = hash
	return record, nil
}

// ValidateRecord verifies the sequence and hash-chain fields of a record
// received from a replica source. Keeping this beside Append means a follower
// does not get a second, subtly different implementation of the log format.
func ValidateRecord(record Record, previous string, expectedSeq uint64) error {
	if record.Seq != expectedSeq {
		return fmt.Errorf("ledger: unexpected sequence %d (want %d)", record.Seq, expectedSeq)
	}
	if record.PrevHash != previous {
		return fmt.Errorf("ledger: broken previous hash at sequence %d", record.Seq)
	}
	if record.Hash != hashRecord(record.Seq, record.Kind, record.Payload, record.PrevHash) {
		return fmt.Errorf("ledger: hash mismatch at sequence %d", record.Seq)
	}
	return nil
}

// Records returns a copy of the log.
func (l *Log) Records() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Record, len(l.records))
	copy(out, l.records)
	return out
}

// RecordsRange returns records with Seq in [from, from+limit). Seq is
// 1-based, so from=0 starts at the first record.
func (l *Log) RecordsRange(from uint64, limit int) []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	if limit <= 0 || limit > 1024 {
		limit = 1024
	}
	out := make([]Record, 0, limit)
	for _, record := range l.records {
		if record.Seq > from && len(out) < limit {
			out = append(out, record)
		}
	}
	return out
}

// Len is the number of committed records.
func (l *Log) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.records)
}

// LastHash is the head of the hash chain ("" when empty).
func (l *Log) LastHash() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.lastHash
}

// Path is the log file location.
func (l *Log) Path() string { return l.path }

// Close releases the file handle.
func (l *Log) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func hashRecord(seq uint64, kind string, payload json.RawMessage, prev string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d\x00%s\x00", seq, kind)
	h.Write(payload)
	h.Write([]byte{0})
	h.Write([]byte(prev))
	return hex.EncodeToString(h.Sum(nil))
}
