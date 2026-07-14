package knirvproof

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const cidPrefix = "sha256:"

func NormalizeSHA256(value string) (string, error) {
	hexValue := strings.ToLower(strings.TrimSpace(value))
	hexValue = strings.TrimPrefix(hexValue, cidPrefix)
	if len(hexValue) != sha256.Size*2 {
		return "", fmt.Errorf("expected a SHA-256 digest")
	}
	if _, err := hex.DecodeString(hexValue); err != nil {
		return "", fmt.Errorf("invalid SHA-256 digest: %w", err)
	}
	return cidPrefix + hexValue, nil
}

func digestHex(value string) (string, error) {
	normalized, err := NormalizeSHA256(value)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(normalized, cidPrefix), nil
}

func HashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return cidPrefix + hex.EncodeToString(sum[:])
}

// StorageRoot is a deterministic Merkle root over ordered ciphertext object
// descriptors. A leaf is SHA-256("<normalized-cid>\n<size>"). Odd levels
// duplicate their final node. Object order is part of the proof descriptor.
func StorageRoot(objects []ObjectRef) (string, error) {
	if len(objects) == 0 {
		return "", fmt.Errorf("at least one ciphertext object is required")
	}
	level := make([][sha256.Size]byte, 0, len(objects))
	seen := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		cid, err := NormalizeSHA256(object.CID)
		if err != nil {
			return "", fmt.Errorf("object cid: %w", err)
		}
		if object.Size < 0 {
			return "", fmt.Errorf("object %s has negative size", cid)
		}
		if _, exists := seen[cid]; exists {
			return "", fmt.Errorf("duplicate object %s", cid)
		}
		seen[cid] = struct{}{}
		level = append(level, sha256.Sum256([]byte(cid+"\n"+strconv.FormatInt(object.Size, 10))))
	}
	for len(level) > 1 {
		next := make([][sha256.Size]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			right := level[i]
			if i+1 < len(level) {
				right = level[i+1]
			}
			pair := make([]byte, 0, sha256.Size*2)
			pair = append(pair, level[i][:]...)
			pair = append(pair, right[:]...)
			next = append(next, sha256.Sum256(pair))
		}
		level = next
	}
	return cidPrefix + hex.EncodeToString(level[0][:]), nil
}
