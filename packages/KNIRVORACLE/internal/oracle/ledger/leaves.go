package ledger

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// The leaf-hash list is a sidecar file next to the state log (<log>.leaves). It
// holds one record hash per line, in sequence order, and is written only by the
// log itself. It is deliberately separate from the audit MMR leaf log
// (mmr_leaf_log.json): that file is written by the checkpoint pipeline, and a
// second copy of the ledger's hashes here lets startup check the state log
// against an independent record of what was committed.
//
// The list is verified, never repaired. A missing, short, long or altered list
// is reported by VerifyLeafList and the caller marks the ledger broken.

// LeafListPath is the leaf-hash list that belongs to the log at logPath.
func LeafListPath(logPath string) string { return logPath + ".leaves" }

// openLeafList opens the leaf-hash list for appending, creating it if absent.
func openLeafList(logPath string) (*os.File, error) {
	f, err := os.OpenFile(LeafListPath(logPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("ledger: open leaf list: %w", err)
	}
	return f, nil
}

// appendLeafHashes writes and fsyncs the hashes of records to the leaf-hash list.
// It runs only after the same records are durable in the log.
func (l *Log) appendLeafHashes(records []Record) error {
	if len(records) == 0 {
		return nil
	}
	var b strings.Builder
	for _, r := range records {
		b.WriteString(r.Hash)
		b.WriteByte('\n')
	}
	if err := writeAll(l.leaves, []byte(b.String())); err != nil {
		return fmt.Errorf("ledger: append leaf list: %w", err)
	}
	if err := l.leaves.Sync(); err != nil {
		return fmt.Errorf("ledger: fsync leaf list: %w", err)
	}
	return nil
}

// VerifyLeafList checks the leaf-hash list against the records the log holds. It
// returns nil only when the list has exactly one line per record, in order, and
// each line equals that record's hash. A missing file with records present is a
// mismatch. It never writes or repairs the list.
func (l *Log) VerifyLeafList() error {
	l.mu.Lock()
	records := make([]Record, len(l.records))
	copy(records, l.records)
	l.mu.Unlock()

	f, err := os.Open(LeafListPath(l.path))
	if err != nil {
		if os.IsNotExist(err) && len(records) == 0 {
			return nil
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("ledger: leaf list missing for %d records", len(records))
		}
		return fmt.Errorf("ledger: read leaf list: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 4096), 1024)
	i := 0
	for scanner.Scan() {
		if i >= len(records) {
			return fmt.Errorf("ledger: leaf list has more entries than the %d records", len(records))
		}
		if scanner.Text() != records[i].Hash {
			return fmt.Errorf("ledger: leaf list entry %d does not match record seq %d", i+1, records[i].Seq)
		}
		i++
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ledger: scan leaf list: %w", err)
	}
	if i != len(records) {
		return fmt.Errorf("ledger: leaf list has %d entries for %d records", i, len(records))
	}
	return nil
}
