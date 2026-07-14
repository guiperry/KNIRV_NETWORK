package knirvproof

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrTooLarge = errors.New("object too large")
)

var safeIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type FileStore struct {
	root           string
	maxObjectBytes int64
	mu             sync.RWMutex
}

func NewFileStore(root string, maxObjectBytes int64) (*FileStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("proof store root is required")
	}
	if maxObjectBytes <= 0 {
		return nil, fmt.Errorf("maximum object size must be positive")
	}
	store := &FileStore{root: root, maxObjectBytes: maxObjectBytes}
	for _, dir := range []string{"objects/sha256", "proofs", "operations", "indexes/commits", "indexes/objects", "tmp"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			return nil, fmt.Errorf("create proof store %s: %w", dir, err)
		}
	}
	return store, nil
}

func (s *FileStore) ReserveObject(projectID, cid string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	digest, err := digestHex(cid)
	if err != nil {
		return err
	}
	path := filepath.Join(s.root, "indexes", "objects", projectID, digest)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	if file != nil {
		return file.Close()
	}
	return nil
}

func (s *FileStore) ProjectHasObject(projectID, cid string) (bool, error) {
	if err := validateProjectID(projectID); err != nil {
		return false, err
	}
	digest, err := digestHex(cid)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(s.root, "indexes", "objects", projectID, digest))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func validateProjectID(projectID string) error {
	if !safeIDPattern.MatchString(projectID) {
		return fmt.Errorf("invalid project id")
	}
	return nil
}

func (s *FileStore) objectPath(cid string) (string, error) {
	digest, err := digestHex(cid)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.root, "objects", "sha256", digest[:2], digest[2:]), nil
}

func (s *FileStore) HasObject(cid string) (bool, int64, error) {
	path, err := s.objectPath(cid)
	if err != nil {
		return false, 0, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	if !info.Mode().IsRegular() {
		return false, 0, fmt.Errorf("CAS path is not a regular file")
	}
	return true, info.Size(), nil
}

func (s *FileStore) PutObject(ctx context.Context, cid string, source io.Reader) (int64, bool, error) {
	normalized, err := NormalizeSHA256(cid)
	if err != nil {
		return 0, false, err
	}
	destination, err := s.objectPath(normalized)
	if err != nil {
		return 0, false, err
	}
	if exists, size, err := s.HasObject(normalized); err != nil {
		return 0, false, err
	} else if exists {
		return size, true, nil
	}

	tmp, err := os.CreateTemp(filepath.Join(s.root, "tmp"), ".object-*")
	if err != nil {
		return 0, false, err
	}
	tmpPath := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	reader := io.LimitReader(&contextReader{ctx: ctx, reader: source}, s.maxObjectBytes+1)
	written, err := io.Copy(io.MultiWriter(tmp, hasher), reader)
	if err != nil {
		return 0, false, err
	}
	if written > s.maxObjectBytes {
		return 0, false, ErrTooLarge
	}
	actual := cidPrefix + hex.EncodeToString(hasher.Sum(nil))
	if actual != normalized {
		return 0, false, fmt.Errorf("ciphertext cid mismatch: expected %s, got %s", normalized, actual)
	}
	if err := tmp.Sync(); err != nil {
		return 0, false, err
	}
	if err := tmp.Close(); err != nil {
		return 0, false, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return 0, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if info, statErr := os.Stat(destination); statErr == nil {
		return info.Size(), true, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return 0, false, statErr
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		return 0, false, err
	}
	if err := os.Chmod(destination, 0o600); err != nil {
		return 0, false, err
	}
	keep = true
	return written, false, nil
}

func (s *FileStore) OpenObject(cid string) (*os.File, int64, error) {
	path, err := s.objectPath(cid)
	if err != nil {
		return nil, 0, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

func (s *FileStore) proofPath(projectID, proofRoot string) (string, error) {
	if err := validateProjectID(projectID); err != nil {
		return "", err
	}
	digest, err := digestHex(proofRoot)
	if err != nil {
		return "", fmt.Errorf("proof root: %w", err)
	}
	return filepath.Join(s.root, "proofs", projectID, digest+".json"), nil
}

func (s *FileStore) operationPath(operationID string) (string, error) {
	if !safeIDPattern.MatchString(operationID) {
		return "", fmt.Errorf("invalid operation id")
	}
	return filepath.Join(s.root, "operations", operationID+".json"), nil
}

func (s *FileStore) commitIndexPath(projectID, commitDigest string) (string, error) {
	if err := validateProjectID(projectID); err != nil {
		return "", err
	}
	digest, err := digestHex(commitDigest)
	if err != nil {
		return "", fmt.Errorf("commit digest: %w", err)
	}
	return filepath.Join(s.root, "indexes", "commits", projectID, digest+".json"), nil
}

func (s *FileStore) GetProof(projectID, proofRoot string) (*ProofRecord, error) {
	path, err := s.proofPath(projectID, proofRoot)
	if err != nil {
		return nil, err
	}
	var record ProofRecord
	if err := readJSON(path, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *FileStore) SaveProof(record *ProofRecord) error {
	if record == nil {
		return fmt.Errorf("proof record is required")
	}
	path, err := s.proofPath(record.Submission.ProjectID, record.Submission.ProofRoot)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSONAtomic(path, record)
}

func (s *FileStore) GetOperation(operationID string) (*Operation, error) {
	path, err := s.operationPath(operationID)
	if err != nil {
		return nil, err
	}
	var operation Operation
	if err := readJSON(path, &operation); err != nil {
		return nil, err
	}
	return &operation, nil
}

func (s *FileStore) SaveOperation(operation *Operation) error {
	if operation == nil {
		return fmt.Errorf("operation is required")
	}
	path, err := s.operationPath(operation.ID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSONAtomic(path, operation)
}

func (s *FileStore) BindCommit(projectID, commitDigest, proofRoot string) error {
	path, err := s.commitIndexPath(projectID, commitDigest)
	if err != nil {
		return err
	}
	normalizedProof, err := NormalizeSHA256(proofRoot)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var existing struct {
		ProofRoot string `json:"proof_root"`
	}
	if err := readJSON(path, &existing); err == nil {
		if existing.ProofRoot == normalizedProof {
			return nil
		}
		return fmt.Errorf("%w: commit already bound to %s", ErrConflict, existing.ProofRoot)
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return writeJSONAtomic(path, struct {
		ProofRoot string `json:"proof_root"`
	}{ProofRoot: normalizedProof})
}

func (s *FileStore) ProofForCommit(projectID, commitDigest string) (*ProofRecord, error) {
	path, err := s.commitIndexPath(projectID, commitDigest)
	if err != nil {
		return nil, err
	}
	var index struct {
		ProofRoot string `json:"proof_root"`
	}
	if err := readJSON(path, &index); err != nil {
		return nil, err
	}
	return s.GetProof(projectID, index.ProofRoot)
}

func (s *FileStore) ListOperations() ([]Operation, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "operations"))
	if err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var operation Operation
		if err := readJSON(filepath.Join(s.root, "operations", entry.Name()), &operation); err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, nil
}

func newOperationID() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "op_" + hex.EncodeToString(random), nil
}

func readJSON(path string, destination any) error {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".metadata-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(buffer)
	}
}
