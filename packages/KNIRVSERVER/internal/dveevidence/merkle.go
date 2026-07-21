package dveevidence

import "strings"

func MerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return sha256Hex(nil)
	}
	level := make([]string, len(hashes))
	for i, hash := range hashes {
		level[i] = normalizeArtifactHash(hash)
	}
	for len(level) > 1 {
		next := make([]string, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			next = append(next, hashPair(left, right))
		}
		level = next
	}
	return level[0]
}

func verifyMerkle(hashes []string, expected string) error {
	got := MerkleRoot(hashes)
	if strings.TrimSpace(expected) == "" {
		return &VerifyError{Code: "merkle", Message: "missing artifact merkle root"}
	}
	if got != expected {
		return &VerifyError{Code: "merkle", Message: "artifact merkle root mismatch"}
	}
	return nil
}
