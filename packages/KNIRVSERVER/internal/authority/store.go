package authority

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	sdk "github.com/guiperry/knirv-sdk-go/authority"
)

// CertJSON and RecordJSON are the core's wire shapes (core/src/authority.rs,
// serde with hex byte fields). They are also what the HTTP surface returns, so a
// consumer can hand a record straight to the core or the browser verifier.
type CertJSON struct {
	SessionPub    string `json:"session_pub"`
	Operator      string `json:"operator"`
	Epoch         uint64 `json:"epoch"`
	NotBefore     int64  `json:"not_before"`
	NotAfter      int64  `json:"not_after"`
	Scope         string `json:"scope"`
	Entropy       string `json:"entropy"`
	SigByLongterm string `json:"sig_by_longterm"`
}

type RecordJSON struct {
	Tier                uint8    `json:"tier"`
	Domain              string   `json:"domain"`
	Scope               string   `json:"scope"`
	Operator            string   `json:"operator"`
	Epoch               uint64   `json:"epoch"`
	CanonicalActionHash string   `json:"canonical_action_hash"`
	PrevRecordHash      string   `json:"prev_record_hash"`
	Header              string   `json:"header"`
	Nonce               uint32   `json:"nonce"`
	Hash                string   `json:"hash"`
	TargetBits          uint32   `json:"target_bits"`
	WitnessKind         string   `json:"witness_kind"`
	DeviceClaim         string   `json:"device_claim,omitempty"`
	SessionCert         CertJSON `json:"session_cert"`
	Signature           string   `json:"signature"`
}

// ToJSON converts an SDK record to the wire shape.
func ToJSON(r *sdk.Record) RecordJSON {
	return RecordJSON{
		Tier: uint8(r.Tier), Domain: r.Domain, Scope: r.Scope, Operator: r.Operator, Epoch: r.Epoch,
		CanonicalActionHash: hex.EncodeToString(r.CanonicalActionHash[:]),
		PrevRecordHash:      hex.EncodeToString(r.PrevRecordHash[:]),
		Header:              hex.EncodeToString(r.Header[:]),
		Nonce:               r.Nonce, Hash: hex.EncodeToString(r.Hash[:]),
		TargetBits: r.TargetBits, WitnessKind: r.WitnessKind, DeviceClaim: r.DeviceClaim,
		SessionCert: CertJSON{
			SessionPub: hex.EncodeToString(r.SessionCert.SessionPub), Operator: r.SessionCert.Operator,
			Epoch: r.SessionCert.Epoch, NotBefore: r.SessionCert.NotBefore, NotAfter: r.SessionCert.NotAfter,
			Scope: r.SessionCert.Scope, Entropy: r.SessionCert.Entropy,
			SigByLongterm: hex.EncodeToString(r.SessionCert.Signature),
		},
		Signature: hex.EncodeToString(r.Signature),
	}
}

func decodeFixed(name, text string, out []byte) error {
	raw, err := hex.DecodeString(text)
	if err != nil || len(raw) != len(out) {
		return fmt.Errorf("%s must be %d bytes of hex", name, len(out))
	}
	copy(out, raw)
	return nil
}

// FromJSON converts the wire shape back to an SDK record, refusing malformed hex.
func FromJSON(j RecordJSON) (*sdk.Record, error) {
	r := &sdk.Record{
		Tier: sdk.Tier(j.Tier), Domain: j.Domain, Scope: j.Scope, Operator: j.Operator, Epoch: j.Epoch,
		Nonce: j.Nonce, TargetBits: j.TargetBits, WitnessKind: j.WitnessKind, DeviceClaim: j.DeviceClaim,
	}
	if err := decodeFixed("canonical_action_hash", j.CanonicalActionHash, r.CanonicalActionHash[:]); err != nil {
		return nil, err
	}
	if err := decodeFixed("prev_record_hash", j.PrevRecordHash, r.PrevRecordHash[:]); err != nil {
		return nil, err
	}
	if err := decodeFixed("header", j.Header, r.Header[:]); err != nil {
		return nil, err
	}
	if err := decodeFixed("hash", j.Hash, r.Hash[:]); err != nil {
		return nil, err
	}
	pub, err := hex.DecodeString(j.SessionCert.SessionPub)
	if err != nil {
		return nil, errors.New("session_pub is not hex")
	}
	sigCert, err := hex.DecodeString(j.SessionCert.SigByLongterm)
	if err != nil {
		return nil, errors.New("sig_by_longterm is not hex")
	}
	sig, err := hex.DecodeString(j.Signature)
	if err != nil {
		return nil, errors.New("signature is not hex")
	}
	r.SessionCert = sdk.SessionCert{
		SessionPub: pub, Operator: j.SessionCert.Operator, Epoch: j.SessionCert.Epoch,
		NotBefore: j.SessionCert.NotBefore, NotAfter: j.SessionCert.NotAfter,
		Scope: j.SessionCert.Scope, Entropy: j.SessionCert.Entropy, Signature: sigCert,
	}
	r.Signature = sig
	return r, nil
}

// StoredRecord is one line of records.jsonl: the record plus local facts the
// record does not authenticate.
type StoredRecord struct {
	Record RecordJSON `json:"record"`
	// IssuedAt is the service's own clock when the record was produced. It is
	// NOT authenticated by the signature (AUTHORITY_SPEC.md section 2.3, open D1) and
	// must not be used for retirement or slashing; it only lets a reader choose
	// an as_of time for re-verification (section 5.2, open D3).
	IssuedAt int64 `json:"issued_at"`
	// Verified and ErrorCode are the observe-mode self-verification verdict at
	// issue time. In observe mode a failure is recorded, never acted on.
	Verified  bool   `json:"verified"`
	ErrorCode string `json:"error_code,omitempty"`
	Mode      string `json:"mode"`
	// BreakGlass is true when the long-term key signed this record directly
	// because no session key was available (Task 2c.1 step 9). It is logged,
	// counted and returned, never silently normal. PROVISIONAL: what makes a
	// record break-glass is open (spec section 10, D4); this is the narrowest,
	// ceremony-independent reading, "signed by the long-term key itself".
	BreakGlass bool `json:"break_glass,omitempty"`
}

// Hash returns the record's own hash (its witness) as hex.
func (s StoredRecord) Hash() string { return s.Record.Hash }

// AnchorState is what the anchorer must remember across restarts, or it would
// re-post from height 1 and the Oracle would refuse the continuity.
type AnchorState struct {
	LastEnd        uint64 `json:"last_end"`
	PrevCheckpoint string `json:"prev_checkpoint"` // hex of the last checkpoint digest
	Registered     bool   `json:"registered"`
}

// ErrCorrupt means records.jsonl is damaged in a way the store will not repair.
var ErrCorrupt = errors.New("authority: record store is corrupt")

// Store is an append-only, fsynced record log with an in-memory index.
type Store struct {
	mu     sync.Mutex
	dir    string
	file   *os.File
	byHash map[string]StoredRecord
	order  []string
	heads  map[string][32]byte // operator -> hash of the newest record
	counts map[string]uint64
}

const (
	recordsFile = "records.jsonl"
	anchorFile  = "anchor.json"
)

// OpenStore opens (creating if needed) dir and loads every record. A torn final
// line, the signature of a crash mid-append, is truncated away and reported via
// the repaired return. Any other damage, or a broken chain link, is ErrCorrupt:
// the caller must degrade and refuse to append rather than extend a broken chain.
func OpenStore(dir string) (st *Store, repaired bool, err error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, false, fmt.Errorf("create authority dir: %w", err)
	}
	path := filepath.Join(dir, recordsFile)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open %s: %w", path, err)
	}
	s := &Store{dir: dir, file: f, byHash: map[string]StoredRecord{}, heads: map[string][32]byte{}, counts: map[string]uint64{}}
	goodEnd, repaired, err := s.load()
	if err != nil {
		f.Close()
		return nil, false, err
	}
	if repaired {
		if err := f.Truncate(goodEnd); err != nil {
			f.Close()
			return nil, false, fmt.Errorf("truncate torn record: %w", err)
		}
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		return nil, false, err
	}
	return s, repaired, nil
}

func (s *Store) load() (goodEnd int64, repaired bool, err error) {
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return 0, false, err
	}
	reader := bufio.NewReaderSize(s.file, 1<<20)
	var offset int64
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && errors.Is(readErr, io.EOF) {
			return offset, repaired, nil
		}
		complete := len(line) > 0 && line[len(line)-1] == '\n'
		var sr StoredRecord
		decodeErr := json.Unmarshal(bytes.TrimSpace(line), &sr)
		if !complete || decodeErr != nil {
			if errors.Is(readErr, io.EOF) { // only the final line may be torn
				return offset, true, nil
			}
			return 0, false, fmt.Errorf("%w: unreadable record at byte %d", ErrCorrupt, offset)
		}
		if err := s.index(sr); err != nil {
			return 0, false, fmt.Errorf("%w: record at byte %d: %v", ErrCorrupt, offset, err)
		}
		offset += int64(len(line))
	}
}

// index checks the record's shape and chain link, then adds it. Callers hold no lock
// during load; Append holds mu.
func (s *Store) index(sr StoredRecord) error {
	rec, err := FromJSON(sr.Record)
	if err != nil {
		return err
	}
	if prev, ok := s.heads[rec.Operator]; ok {
		if rec.PrevRecordHash != prev {
			return fmt.Errorf("prev_record_hash %x does not extend the operator's head %x", rec.PrevRecordHash[:4], prev[:4])
		}
	} else if rec.PrevRecordHash != ([32]byte{}) {
		return errors.New("first record of an operator must have the zero prev_record_hash")
	}
	if _, dup := s.byHash[sr.Record.Hash]; dup {
		return errors.New("duplicate record hash")
	}
	s.byHash[sr.Record.Hash] = sr
	s.order = append(s.order, sr.Record.Hash)
	s.heads[rec.Operator] = rec.Hash
	s.counts[rec.Operator]++
	return nil
}

// Head returns the hash of the operator's newest record, or the zero hash for a
// new chain (the genesis prev_record_hash).
func (s *Store) Head(operator string) [32]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.heads[operator]
}

// Count is the number of records the operator has appended (its chain height).
func (s *Store) Count(operator string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[operator]
}

// Total is the number of records of every operator.
func (s *Store) Total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.order)
}

// Get returns the record with the given hex hash.
func (s *Store) Get(hash string) (StoredRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sr, ok := s.byHash[hash]
	return sr, ok
}

// Append writes and fsyncs one record. It refuses a record that does not extend
// the operator's current head, so the store can never hold a fork or a gap.
func (s *Store) Append(sr StoredRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.index(sr); err != nil {
		return err
	}
	line, err := json.Marshal(sr)
	if err != nil {
		s.unindex(sr)
		return err
	}
	line = append(line, '\n')
	if _, err := s.file.Write(line); err != nil {
		s.unindex(sr)
		return fmt.Errorf("append record: %w", err)
	}
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("sync record: %w", err)
	}
	return nil
}

// unindex undoes index for a record that could not be written.
func (s *Store) unindex(sr StoredRecord) {
	rec, err := FromJSON(sr.Record)
	if err != nil {
		return
	}
	delete(s.byHash, sr.Record.Hash)
	if n := len(s.order); n > 0 && s.order[n-1] == sr.Record.Hash {
		s.order = s.order[:n-1]
	}
	s.counts[rec.Operator]--
	if s.counts[rec.Operator] == 0 {
		delete(s.heads, rec.Operator)
		delete(s.counts, rec.Operator)
		return
	}
	s.heads[rec.Operator] = rec.PrevRecordHash
}

// LoadAnchor reads the anchorer's saved state; a missing file is the zero state.
func (s *Store) LoadAnchor() (AnchorState, error) {
	var st AnchorState
	data, err := os.ReadFile(filepath.Join(s.dir, anchorFile))
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	return st, json.Unmarshal(data, &st)
}

// SaveAnchor writes the anchorer's state atomically (temp file, fsync, rename).
func (s *Store) SaveAnchor(st AnchorState) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dir, anchorFile+".tmp")
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dir, anchorFile))
}

// Dir is the directory holding the store's files.
func (s *Store) Dir() string { return s.dir }

// Close releases the file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.file.Close()
}
