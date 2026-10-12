package connector

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteRecords writes Stage 0's stable interchange stream. The file uses
// newline-delimited JSON so it can be resumed/tailed cheaply; the mapper also
// accepts this representation when a Parquet runtime is unavailable.
func WriteRecords(path string, records []RawRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	seen := map[string]struct{}{}
	for _, raw := range records {
		raw.Text = strings.TrimSpace(raw.Text)
		if len(raw.Text) < 32 || len(raw.Text) > 16384 {
			continue
		}
		hash := sha256.Sum256([]byte(raw.Text))
		key := hex.EncodeToString(hash[:])
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		b, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, string(b)); err != nil {
			return err
		}
	}
	return nil
}
