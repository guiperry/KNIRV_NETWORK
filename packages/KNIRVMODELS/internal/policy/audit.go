package policy

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry is one audited tool decision or result. Prev and Hash chain the log:
// editing or removing any line breaks every later hash.
type Entry struct {
	Seq        uint64 `json:"seq"`
	Time       string `json:"time"`
	Stage      string `json:"stage"`
	FrameHash  string `json:"frame_hash"`
	Caller     string `json:"caller"`
	Tool       string `json:"tool"`
	ArgsHash   string `json:"args_hash"`
	ResultHash string `json:"result_hash,omitempty"`
	Outcome    string `json:"outcome"`
	Reason     string `json:"reason,omitempty"`
	Prev       string `json:"prev"`
	Hash       string `json:"hash"`
}

func (e Entry) digest() string {
	e.Hash = ""
	b, _ := json.Marshal(e)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Audit is an append-only, hash-chained JSONL file (tool_audit.jsonl).
type Audit struct {
	path string
	mu   sync.Mutex
	seq  uint64
	last string
}

// OpenAudit opens or creates the log, verifying the existing chain first so a
// tampered file is refused rather than extended.
func OpenAudit(path string) (*Audit, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	a := &Audit{path: path}
	n, last, err := VerifyChain(path)
	if err != nil {
		return nil, err
	}
	a.seq, a.last = n, last
	return a, nil
}

// Append writes one entry, filling seq, time and the chain.
func (a *Audit) Append(e Entry) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	e.Seq = a.seq + 1
	e.Time = time.Now().UTC().Format(time.RFC3339Nano)
	e.Prev = a.last
	e.Hash = e.digest()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	a.seq, a.last = e.Seq, e.Hash
	return nil
}

// RecordResult audits the outcome of an executed call.
func (a *Audit) RecordResult(c Call, result []byte, outcome string) error {
	return a.Append(Entry{Stage: c.Stage, FrameHash: c.FrameHash, Caller: c.Caller, Tool: c.Tool,
		ArgsHash: hashBytes(c.Args), ResultHash: hashBytes(result), Outcome: outcome})
}

func hashBytes(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

// VerifyChain reads the log and returns its entry count and last hash. A
// missing file is an empty chain.
func VerifyChain(path string) (uint64, string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return 0, "", nil
	}
	if err != nil {
		return 0, "", err
	}
	defer f.Close()
	var n uint64
	prev := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return n, prev, fmt.Errorf("audit line %d is not valid: %w", n+1, err)
		}
		if e.Seq != n+1 || e.Prev != prev || e.Hash != e.digest() {
			return n, prev, fmt.Errorf("audit chain broken at entry %d", n+1)
		}
		n, prev = e.Seq, e.Hash
	}
	return n, prev, sc.Err()
}

// RecordResultIfSet is RecordResult that tolerates a nil log (tests, or a node
// started without auditing), so callers need no nil check.
func (a *Audit) RecordResultIfSet(c Call, result []byte, outcome string) error {
	if a == nil {
		return nil
	}
	return a.RecordResult(c, result, outcome)
}
