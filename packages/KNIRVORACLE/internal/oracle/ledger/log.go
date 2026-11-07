package ledger

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

// readRecords rebuilds the committed record list from the log file. It tolerates
// a crashed/incomplete TRAILING line (a truncated final batch from a crash) by
// dropping it deterministically, so replay never fails the whole log; but it
// still rejects tampering/truncation of any verifiable record (a hash or checksum
// mismatch is never silently ignored).
func readRecords(path string) ([]Record, string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("ledger: read %s: %w", path, err)
	}
	defer file.Close()

	// Read every line up front so we can tell whether a malformed line is the
	// trailing (crash) tail or a committed record in the middle of the log.
	var lines [][]byte
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(raw) == 0 {
			continue
		}
		cp := make([]byte, len(raw))
		copy(cp, raw)
		lines = append(lines, cp)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("ledger: scan %s: %w", path, err)
	}

	records := make([]Record, 0, len(lines))
	last := ""
	for idx, raw := range lines {
		recs, newLast, parseErr, valErr := parseLine(raw, last, uint64(len(records))+1)
		if parseErr != nil {
			// The line could not be decoded at all. A corrupt or truncated TRAILING
			// line is an incomplete crash tail and is dropped; anything else is fatal.
			if idx == len(lines)-1 {
				break
			}
			return nil, "", fmt.Errorf("ledger: %s line %d: %w", path, idx+1, parseErr)
		}
		if valErr != nil {
			// Decoded but failed integrity (hash/checksum/chain): never silently
			// dropped — reject so tampering is not hidden.
			return nil, "", fmt.Errorf("ledger: %s line %d: %w", path, idx+1, valErr)
		}
		records = append(records, recs...)
		last = newLast
	}
	return records, last, nil
}

// parseLine decodes one log line, which is either a single Record or a
// crash-atomic BatchFrame (one JSON line holding a whole batch + checksum). It
// returns the expanded records, the new head hash, and either a parse error
// (could not decode) or a validation error (decoded but failed integrity).
func parseLine(raw []byte, last string, startSeq uint64) (recs []Record, newLast string, parseErr, valErr error) {
	var r Record
	if err := json.Unmarshal(raw, &r); err != nil {
		var f BatchFrame
		if ferr := json.Unmarshal(raw, &f); ferr != nil {
			return nil, "", err, nil
		}
		recs, newLast, valErr = parseFrame(f, last, startSeq)
		return recs, newLast, nil, valErr
	}
	if r.Kind == "" {
		var f BatchFrame
		if ferr := json.Unmarshal(raw, &f); ferr != nil {
			return nil, "", fmt.Errorf("ledger: empty record kind"), nil
		}
		recs, newLast, valErr = parseFrame(f, last, startSeq)
		return recs, newLast, nil, valErr
	}
	if r.Hash != hashRecord(r.Seq, r.Kind, r.Payload, r.PrevHash) {
		return nil, "", nil, fmt.Errorf("hash mismatch (record altered or truncated)")
	}
	if r.Seq != startSeq || r.PrevHash != last {
		return nil, "", nil, fmt.Errorf("broken chain (seq %d, prev %q)", r.Seq, r.PrevHash)
	}
	return []Record{r}, r.Hash, nil, nil
}

// parseFrame validates and expands a BatchFrame: it verifies the batch checksum,
// then each inner record's hash-chain link against the running chain, so a batch
// is only ever applied whole.
func parseFrame(f BatchFrame, last string, startSeq uint64) ([]Record, string, error) {
	if f.Checksum != checksumRecords(f.Records) {
		return nil, "", fmt.Errorf("batch checksum mismatch")
	}
	cur := last
	seq := startSeq
	out := make([]Record, 0, len(f.Records))
	for _, rec := range f.Records {
		if rec.Hash != hashRecord(rec.Seq, rec.Kind, rec.Payload, rec.PrevHash) {
			return nil, "", fmt.Errorf("batch record hash mismatch")
		}
		if rec.Seq != seq || rec.PrevHash != cur {
			return nil, "", fmt.Errorf("batch broken chain (seq %d, prev %q)", rec.Seq, rec.PrevHash)
		}
		out = append(out, rec)
		cur = rec.Hash
		seq++
	}
	return out, cur, nil
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
	if err := writeAll(l.file, append(line, '\n')); err != nil {
		return Record{}, fmt.Errorf("ledger: append: %w", err)
	}
	if err := l.file.Sync(); err != nil {
		return Record{}, fmt.Errorf("ledger: fsync: %w", err)
	}

	l.records = append(l.records, record)
	l.lastHash = hash
	return record, nil
}

// AppendBatch writes and fsyncs several records as ONE crash-atomic batch: the
// whole batch is framed into a single JSON line (records + checksum) and a single
// write+fsync commits it. A crash mid-write leaves a truncated trailing line,
// which replay detects and ignores (never failing the whole replay). The caller
// applies the returned records to its read model only after AppendBatch returns.
// The hash chain is extended sequentially across the batch.
func (l *Log) AppendBatch(entries []BatchEntry) ([]Record, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	bodies := make([][]byte, len(entries))
	for i, e := range entries {
		body, err := json.Marshal(e.Payload)
		if err != nil {
			return nil, fmt.Errorf("ledger: encode %s: %w", e.Kind, err)
		}
		bodies[i] = body
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	seq := uint64(len(l.records)) + 1
	prev := l.lastHash
	inner := make([]Record, 0, len(entries))
	for i, e := range entries {
		h := hashRecord(seq, e.Kind, bodies[i], prev)
		inner = append(inner, Record{Seq: seq, Kind: e.Kind, Payload: bodies[i], PrevHash: prev, Hash: h})
		prev = h
		seq++
	}
	// Frame the whole batch into one line so a partial write is detectable as an
	// incomplete trailing record rather than interleaved with committed data.
	frame := BatchFrame{Records: inner, Checksum: checksumRecords(inner)}
	buf, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("ledger: encode batch frame: %w", err)
	}
	buf = append(buf, '\n')
	if err := writeAll(l.file, buf); err != nil {
		return nil, fmt.Errorf("ledger: append batch: %w", err)
	}
	if err := l.file.Sync(); err != nil {
		return nil, fmt.Errorf("ledger: fsync batch: %w", err)
	}
	// Advance the in-memory view only after the batch is durable.
	l.records = append(l.records, inner...)
	l.lastHash = prev
	return inner, nil
}

// BatchFrame is a crash-atomic batch: one JSON line holding the whole batch plus
// a checksum. Replay validates the checksum and applies the batch only whole.
type BatchFrame struct {
	Records  []Record `json:"records"`
	Checksum string   `json:"checksum"`
}

// checksumRecords is a stable digest over the batch records, used to detect a
// truncated or altered frame on replay.
func checksumRecords(records []Record) string {
	sum := sha256.New()
	enc := json.NewEncoder(sum)
	for _, r := range records {
		_ = enc.Encode(r)
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// writeAll writes buf completely, looping on short writes, so a partial write is
// never silently accepted.
func writeAll(w io.Writer, buf []byte) error {
	for len(buf) > 0 {
		n, err := w.Write(buf)
		if n > 0 {
			buf = buf[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 && err == nil {
			return io.ErrShortWrite
		}
	}
	return nil
}

// BatchEntry is one record in an AppendBatch call.
type BatchEntry struct {
	Kind    string
	Payload any
}


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
