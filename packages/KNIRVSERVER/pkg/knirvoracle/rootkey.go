package knirvoracle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalRootKeyPath returns the single, canonical location for root.key:
// <user config dir>/knirv-server/.key/root.key. On a root-owned install
// (the installer runs as root inside Docker via sudo), os.UserConfigDir()
// resolves this to /root/.config/knirv-server/.key/root.key. The ".key"
// directory is intentionally a dotfile so it doesn't turn up in a casual
// `ls` of the parent directory; this is a discoverability speed bump, not a
// substitute for tight file permissions (0700 dir / 0600 file, root-owned)
// or the AES encryption already applied to the file's contents.
func CanonicalRootKeyPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "knirv-server", ".key", "root.key"), nil
}

// rootKeyCandidates returns, in priority order, the paths that may hold
// root.key: an explicit operator-configured override (if provided), then
// the single canonical location. There is deliberately no broader fallback
// search (env vars, /etc, exe-relative, cwd-relative) — root.key lives in
// exactly one place unless an operator explicitly configures otherwise.
func rootKeyCandidates(configuredPath string) []string {
	candidates := make([]string, 0, 2)
	seen := make(map[string]struct{})

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		candidates = append(candidates, path)
	}

	add(configuredPath)

	if canonicalPath, err := CanonicalRootKeyPath(); err == nil {
		add(canonicalPath)
	}

	return candidates
}

func ResolveRootKeyPath(configuredPath string) (string, error) {
	for _, candidate := range rootKeyCandidates(configuredPath) {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		return candidate, nil
	}

	return "", fmt.Errorf("root.key not found in any expected location")
}

func ValidateRootKeyFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read root.key: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("root.key is empty")
	}

	hasEncryptedContent, hasSalt, err := validateEncryptedRootKeyEnvelope(data)
	if err != nil {
		return fmt.Errorf("root.key is not a valid encrypted protobuf envelope: %w", err)
	}
	if !hasEncryptedContent {
		return fmt.Errorf("root.key is missing encrypted content")
	}
	if !hasSalt {
		return fmt.Errorf("root.key is missing salt")
	}

	return nil
}

func ResolveAndValidateRootKey(configuredPath string) (string, error) {
	var firstInvalidErr error

	for _, candidate := range rootKeyCandidates(configuredPath) {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if err := ValidateRootKeyFile(candidate); err != nil {
			if firstInvalidErr == nil {
				firstInvalidErr = err
			}
			continue
		}
		return candidate, nil
	}

	if firstInvalidErr != nil {
		return "", firstInvalidErr
	}

	return "", fmt.Errorf("root.key not found in any expected location")
}

func validateEncryptedRootKeyEnvelope(data []byte) (bool, bool, error) {
	var hasEncryptedContent bool
	var hasSalt bool

	for i := 0; i < len(data); {
		key, next, err := readVarint(data, i)
		if err != nil {
			return false, false, err
		}
		i = next

		fieldNum := int(key >> 3)
		wireType := int(key & 0x7)

		switch wireType {
		case 0:
			_, next, err = readVarint(data, i)
			if err != nil {
				return false, false, err
			}
			i = next
		case 1:
			if i+8 > len(data) {
				return false, false, fmt.Errorf("truncated fixed64 field")
			}
			i += 8
		case 2:
			length, next, err := readVarint(data, i)
			if err != nil {
				return false, false, err
			}
			i = next
			end := i + int(length)
			if end > len(data) {
				return false, false, fmt.Errorf("truncated length-delimited field")
			}
			if fieldNum == 1 && length > 0 {
				hasEncryptedContent = true
			}
			if fieldNum == 2 && length > 0 {
				hasSalt = true
			}
			i = end
		case 5:
			if i+4 > len(data) {
				return false, false, fmt.Errorf("truncated fixed32 field")
			}
			i += 4
		default:
			return false, false, fmt.Errorf("unsupported wire type %d", wireType)
		}
	}

	return hasEncryptedContent, hasSalt, nil
}

func readVarint(data []byte, start int) (uint64, int, error) {
	var value uint64
	var shift uint

	for i := start; i < len(data); i++ {
		b := data[i]
		value |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return value, i + 1, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, 0, fmt.Errorf("varint overflow")
		}
	}

	return 0, 0, fmt.Errorf("truncated varint")
}
