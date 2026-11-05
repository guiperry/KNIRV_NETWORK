package knirvbao

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func privateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("OpenBao directory is not private: %s", path)
	}
	return nil
}

// installBinary never replaces an existing file: content-addressed paths make
// upgrades additive and an unexpected file a fail-closed error.
func installBinary(root, name string, binary []byte) (string, string, error) {
	if len(binary) == 0 || (name != "bao" && name != "openbao-plugin-secrets-knirv") {
		return "", "", errors.New("missing or invalid embedded binary")
	}
	digest := sha256.Sum256(binary)
	hexDigest := hex.EncodeToString(digest[:])
	binDir := filepath.Join(root, "bin")
	if err := privateDir(root); err != nil {
		return "", "", err
	}
	if err := privateDir(binDir); err != nil {
		return "", "", err
	}
	versionDir := filepath.Join(binDir, hexDigest)
	if err := privateDir(versionDir); err != nil {
		return "", "", err
	}
	path := filepath.Join(versionDir, name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if errors.Is(err, os.ErrExist) {
		info, statErr := os.Lstat(path)
		if statErr != nil {
			return "", "", statErr
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0700 {
			return "", "", errors.New("embedded binary path is not a private executable")
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return "", "", openErr
		}
		defer file.Close()
		hasher := sha256.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return "", "", err
		}
		if !bytes.Equal(hasher.Sum(nil), digest[:]) {
			return "", "", errors.New("existing embedded binary digest mismatch")
		}
		return path, hexDigest, nil
	}
	if err != nil {
		return "", "", err
	}
	if _, err := file.Write(binary); err != nil {
		_ = file.Close()
		return "", "", err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return "", "", err
	}
	if err := file.Close(); err != nil {
		return "", "", err
	}
	return path, hexDigest, nil
}
