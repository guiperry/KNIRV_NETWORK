package evidence

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Store is the content-addressed snapshot store: a normalized document is saved
// once under its own sha256, and a span is a (snapshot, start, end) range of it.
// Anything that cites a span can be checked against the bytes it names.
type Store struct{ dir string }

// OpenStore creates the directory if needed.
func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(d Digest) string {
	h := d.Hex()
	return filepath.Join(s.dir, h[:2], h+".txt")
}

// Put stores normalized text and returns its id. Putting the same bytes again is
// a no-op that returns the same id.
func (s *Store) Put(normalized []byte) (Digest, error) {
	d := SnapshotID(normalized)
	p := s.path(d)
	if _, err := os.Stat(p); err == nil {
		return d, nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return d, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".put-*")
	if err != nil {
		return d, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(normalized); err != nil {
		tmp.Close()
		return d, err
	}
	if err := tmp.Close(); err != nil {
		return d, err
	}
	return d, os.Rename(tmp.Name(), p)
}

// Has reports whether a snapshot is stored.
func (s *Store) Has(d Digest) bool {
	_, err := os.Stat(s.path(d))
	return err == nil
}

// ReadSpan returns bytes [start,end) of a snapshot.
func (s *Store) ReadSpan(d Digest, start, end int) (string, error) {
	if start < 0 || end < start {
		return "", fmt.Errorf("evidence: bad span %d-%d", start, end)
	}
	f, err := os.Open(s.path(d))
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, end-start)
	if _, err := f.ReadAt(buf, int64(start)); err != nil && err != io.EOF {
		return "", err
	}
	return string(buf), nil
}

// Verify re-hashes a stored snapshot and checks it still matches its name.
func (s *Store) Verify(d Digest) error {
	b, err := os.ReadFile(s.path(d))
	if err != nil {
		return err
	}
	if SnapshotID(b) != d {
		return fmt.Errorf("evidence: snapshot %s is corrupt", d)
	}
	return nil
}

// ParseDigest reads "sha256:<hex>" or bare hex.
func ParseDigest(s string) (Digest, error) {
	if len(s) > 7 && s[:7] == "sha256:" {
		s = s[7:]
	}
	var d Digest
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return d, fmt.Errorf("evidence: %q is not a sha256 digest", s)
	}
	copy(d[:], b)
	return d, nil
}

// VerifyStore re-hashes every snapshot under dir against its file name and returns
// the number checked and the ones that no longer match.
func VerifyStore(dir string) (checked int, corrupt []string, err error) {
	err = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".txt") {
			return nil
		}
		name := strings.TrimSuffix(filepath.Base(p), ".txt")
		want, perr := ParseDigest(name)
		if perr != nil {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		checked++
		if SnapshotID(b) != want {
			corrupt = append(corrupt, name)
		}
		return nil
	})
	return
}
