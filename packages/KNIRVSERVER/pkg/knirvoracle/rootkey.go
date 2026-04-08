package knirvoracle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func rootKeyCandidates(configuredPath string) []string {
	candidates := make([]string, 0, 8)
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
	add(os.Getenv("KNIRV_ROOT_KEY_PATH"))
	add(os.Getenv("ORACLE_ROOT_KEY_PATH"))

	if configDir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(configDir, "knirv-server", "root.key"))
	}

	if dataDir, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(dataDir, ".local", "share", "knirv-server", "root.key"))
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		add(filepath.Join(exeDir, "root.key"))
		add(filepath.Join(exeDir, "bin", "root.key"))
		add(filepath.Join(exeDir, "..", "bin", "root.key"))
	}

	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, "bin", "root.key"))
		add(filepath.Join(cwd, "..", "bin", "root.key"))
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
	rootKeyPath, err := ResolveRootKeyPath(configuredPath)
	if err != nil {
		return "", err
	}
	if err := ValidateRootKeyFile(rootKeyPath); err != nil {
		return "", err
	}
	return rootKeyPath, nil
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
