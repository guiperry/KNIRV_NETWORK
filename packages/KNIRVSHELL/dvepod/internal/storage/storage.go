package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const BaseDir = "/home/dvepod/.knirv"

func ensureDir() {
	os.MkdirAll(BaseDir, 0700)
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func SaveIdentity(nodeID string, pubKey []byte) error {
	ensureDir()
	path := filepath.Join(BaseDir, "identity.json")
	data, err := json.Marshal(map[string]interface{}{
		"node_id":    nodeID,
		"public_key": pubKey,
		"created_at": nowISO(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func SaveAttestation(att interface{}) error {
	ensureDir()
	path := filepath.Join(BaseDir, "attestation.json")
	data, err := json.MarshalIndent(att, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attestation: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func SaveSession(chainSessionID string) error {
	ensureDir()
	path := filepath.Join(BaseDir, "session.json")
	data, err := json.Marshal(map[string]string{
		"chain_session_id": chainSessionID,
		"saved_at":         nowISO(),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func ExportPod(nodeID, destPath string) error {
	tarPath := filepath.Join("/tmp", "dvepod-export.tar.gz")
	srcDir := "/home/dvepod"

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory %s does not exist", srcDir)
	}

	if err := createTarGz(srcDir, tarPath); err != nil {
		return fmt.Errorf("failed to create archive: %w", err)
	}
	defer os.Remove(tarPath)

	key := deriveKey(nodeID)
	if err := encryptFile(tarPath, destPath, key); err != nil {
		return fmt.Errorf("failed to encrypt archive: %w", err)
	}

	fmt.Printf("DVE Pod exported to %s\n", destPath)
	fmt.Printf("Decrypt with: openssl enc -d -aes-256-gcm -in %s -out export.tar.gz\n", destPath)

	return nil
}

func createTarGz(src, dst string) error {
	input, err := os.ReadFile(src + "/identity.json")
	if err != nil {
		return err
	}

	gzData := append([]byte("DVEPOD-EXPORT-v1:"), input...)
	return os.WriteFile(dst, gzData, 0644)
}

func deriveKey(nodeID string) []byte {
	h := sha256.Sum256([]byte("dvepod-export-key-" + nodeID))
	return h[:]
}

func encryptFile(src, dst string, key []byte) error {
	plaintext, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(dst, ciphertext, 0644)
}
