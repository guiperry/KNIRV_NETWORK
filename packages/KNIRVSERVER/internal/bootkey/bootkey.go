package bootkey

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// Content holds fields extracted from a decrypted boot.key file.
// Field numbers mirror BootnodeKeyFileContentProto in bootnode_key.proto.
type Content struct {
	RegistrationID               string // field 2
	JWTSecret                    string // field 10
	GeminiAPIKey                 string // field 11
	DeepseekAPIKey               string // field 12
	CerebrasAPIKey               string // field 13
	CloudflareAPIToken           string // field 18
	CloudflareZoneID             string // field 19
	CloudflareAccountID          string // field 20
	CloudflareMainnetTunnelToken string // field 21
	DeviceIP                     string // field 22
	DevicePassword               string // field 23
	DeviceUsername               string // field 24
	CloudflareTestnetTunnelToken string // field 28
	MasterWalletKeyHex           string // field 1 — master wallet key; used as the Validation Chain checkpoint signer identity on Bootnodes that have no root.key (see startValidationChain in main.go)
}

// searchPaths builds a deduplicated, prioritized list of filesystem paths for
// the given filename. envKey is checked first (e.g. "KNIRV_BOOT_KEY_PATH"),
// then standard XDG / system / exe-relative directories.
func searchPaths(filename, envKey string) []string {
	list := make([]string, 0, 12)
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
		list = append(list, path)
	}

	// Explicit env override (e.g. KNIRV_BOOT_KEY_PATH or ORACLE_KEY_PATH).
	add(os.Getenv(envKey))

	// KNIRV_CONFIG_DIR override.
	if configDir := strings.TrimSpace(os.Getenv("KNIRV_CONFIG_DIR")); configDir != "" {
		add(filepath.Join(configDir, filename))
	}

	// System-wide config.
	add(filepath.Join("/etc", "knirv-server", filename))

	// XDG user config (~/.config/knirv-server/).
	if configDir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(configDir, "knirv-server", filename))
	}

	// Home directory shortcuts.
	if homeDir, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(homeDir, ".knirv", filename))
		add(filepath.Join(homeDir, ".local", "share", "knirv-server", filename))
	}

	// Beside the running executable (and its bin/ subdirectory).
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		add(filepath.Join(exeDir, filename))
		add(filepath.Join(exeDir, "bin", filename))
	}

	// CWD and CWD/bin (useful during development and for operators running
	// the binary from its install directory).
	if cwd, err := os.Getwd(); err == nil {
		add(filepath.Join(cwd, filename))
		add(filepath.Join(cwd, "bin", filename))
	}

	return list
}

// candidates returns search paths for boot.key.
func candidates(configuredPath string) []string {
	paths := searchPaths("boot.key", "KNIRV_BOOT_KEY_PATH")
	// Prepend any caller-supplied path so it wins.
	if configuredPath = strings.TrimSpace(configuredPath); configuredPath != "" {
		all := make([]string, 0, len(paths)+1)
		all = append(all, configuredPath)
		for _, p := range paths {
			if p != configuredPath {
				all = append(all, p)
			}
		}
		return all
	}
	return paths
}

// rootKeyCandidates returns search paths for root.key.
func rootKeyCandidates() []string {
	paths := searchPaths("root.key", "ORACLE_KEY_PATH")
	list := make([]string, 0, len(paths)+8)
	seen := make(map[string]struct{}, len(paths)+8)
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		list = append(list, path)
	}

	// Keep the explicit operator override first.
	add(os.Getenv("ORACLE_KEY_PATH"))

	// root.key's designated location includes the hidden .key directory. The
	// generic searchPaths helper intentionally remains unchanged for boot.key.
	if configDir := strings.TrimSpace(os.Getenv("KNIRV_CONFIG_DIR")); configDir != "" {
		add(filepath.Join(configDir, ".key", "root.key"))
	}
	if configDir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(configDir, "knirv-server", ".key", "root.key"))
		// Also check without the .key subdirectory — an operator may have
		// placed root.key directly under ~/.config/knirv-server/.
		add(filepath.Join(configDir, "knirv-server", "root.key"))
	}
	add(filepath.Join("/etc", "knirv-server", ".key", "root.key"))

	// os.UserConfigDir() resolves against the *effective* user — when this
	// process runs under sudo (the common case; see main.go's launch
	// instructions), that's root's ~/.config, not the operator's. sudo sets
	// SUDO_USER even with env_reset, so check that user's config dir too,
	// both with and without the .key subdirectory.
	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" {
		add(filepath.Join("/home", sudoUser, ".config", "knirv-server", ".key", "root.key"))
		add(filepath.Join("/home", sudoUser, ".config", "knirv-server", "root.key"))
	}

	// Last resort: scan every home directory's ~/.config/knirv-server/ for
	// root.key, with and without .key — covers root-run deployments where
	// SUDO_USER isn't set (e.g. a systemd unit running as root directly).
	if entries, err := os.ReadDir("/home"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			add(filepath.Join("/home", entry.Name(), ".config", "knirv-server", ".key", "root.key"))
			add(filepath.Join("/home", entry.Name(), ".config", "knirv-server", "root.key"))
		}
	}

	// Retain legacy locations so existing installations continue to start.
	for _, path := range paths {
		add(path)
	}
	return list
}

// CandidatePaths exposes boot.key search paths for diagnostic error messages.
func CandidatePaths() []string { return candidates("") }

// RootKeyCandidatePaths exposes root.key search paths for diagnostic error messages.
func RootKeyCandidatePaths() []string { return rootKeyCandidates() }

// Exists returns true if boot.key is found in any expected location.
func Exists() bool {
	for _, path := range candidates("") {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// FindRootKey returns the first root.key path found, or empty string.
func FindRootKey() string {
	for _, path := range rootKeyCandidates() {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// RootKeyExists returns true if root.key is found in any expected location.
func RootKeyExists() bool { return FindRootKey() != "" }

// RootKeyCloudflareCreds holds the Cloudflare fields extracted from a decrypted
// root.key file. Field numbers mirror RootKeyFileContentProto in root_key.proto
// (KNIRV_CORP/packages/server/backend_server/internal/proto/root_key.proto) — NOT the
// same field numbers as BootnodeKeyFileContentProto (see Content above); the two
// protos assign cloudflare_account_id / cloudflare_mainnet_tunnel_token to
// different field numbers.
type RootKeyCloudflareCreds struct {
	CloudflareAPIToken           string // field 18
	CloudflareZoneID             string // field 19
	CloudflareAccountID          string // field 26
	CloudflareMainnetTunnelToken string // field 27
	CloudflareOracleTunnelTok    string // field 28
	CloudflareTestnetTunnelToken string // field 38
	RootPrivateKeyHex            string // field 5 — root node private key; used as the Validation Chain checkpoint signer identity on Root nodes (see startValidationChain in main.go)
}

// LoadRootKeyCloudflareCreds finds root.key, decrypts it using the
// ORACLE_KEY_PASSWORD environment variable, and returns its Cloudflare fields.
// Returns nil if root.key is not found or no password is set.
func LoadRootKeyCloudflareCreds() (*RootKeyCloudflareCreds, error) {
	found := FindRootKey()
	if found == "" {
		return nil, fmt.Errorf("root.key not found in any expected location")
	}

	password := os.Getenv("ORACLE_KEY_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("root.key found at %s but ORACLE_KEY_PASSWORD is not set", found)
	}

	fields, err := decryptFields(found, []byte(password))
	if err != nil {
		return nil, err
	}

	return &RootKeyCloudflareCreds{
		CloudflareAPIToken:           fields[18],
		CloudflareZoneID:             fields[19],
		CloudflareAccountID:          fields[26],
		CloudflareMainnetTunnelToken: fields[27],
		CloudflareOracleTunnelTok:    fields[28],
		CloudflareTestnetTunnelToken: fields[38],
		RootPrivateKeyHex:            fields[5],
	}, nil
}

// Load finds boot.key, decrypts it using the BOOT_KEY_PASSWORD environment
// variable, and returns its contents. Returns nil if boot.key is not found
// or no password is set.
func Load() (*Content, error) {
	var found string
	for _, path := range candidates("") {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			found = path
			break
		}
	}
	if found == "" {
		return nil, fmt.Errorf("boot.key not found in any expected location")
	}

	password := os.Getenv("BOOT_KEY_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("boot.key found at %s but BOOT_KEY_PASSWORD is not set", found)
	}

	return decrypt(found, []byte(password))
}

func decryptFields(path string, password []byte) (map[int]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	encContent, salt, err := parseEnvelope(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s envelope: %w", path, err)
	}

	key, err := scrypt.Key(password, salt, 32768, 8, 1, 32)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encContent) < nonceSize {
		return nil, fmt.Errorf("%s: encrypted content too short", path)
	}
	plaintext, err := gcm.Open(nil, encContent[:nonceSize], encContent[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("%s: wrong password or corrupted file", path)
	}

	fields, err := extractStringFields(plaintext)
	if err != nil {
		return nil, fmt.Errorf("parse %s content: %w", path, err)
	}
	return fields, nil
}

// decrypt reads and decrypts a boot.key file produced by boot_key_encryptor.
func decrypt(path string, password []byte) (*Content, error) {
	fields, err := decryptFields(path, password)
	if err != nil {
		return nil, err
	}

	return &Content{
		RegistrationID:               fields[2],
		JWTSecret:                    fields[10],
		GeminiAPIKey:                 fields[11],
		DeepseekAPIKey:               fields[12],
		CerebrasAPIKey:               fields[13],
		CloudflareAPIToken:           fields[18],
		CloudflareZoneID:             fields[19],
		CloudflareAccountID:          fields[20],
		CloudflareMainnetTunnelToken: fields[21],
		DeviceIP:                     fields[22],
		DevicePassword:               fields[23],
		DeviceUsername:               fields[24],
		CloudflareTestnetTunnelToken: fields[28],
		MasterWalletKeyHex:           fields[1],
	}, nil
}

// parseEnvelope extracts encrypted_content (field 1) and salt (field 2)
// from the outer proto3 envelope.
func parseEnvelope(data []byte) (encryptedContent, salt []byte, err error) {
	for i := 0; i < len(data); {
		tag, next, e := readVarint(data, i)
		if e != nil {
			return nil, nil, e
		}
		i = next
		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)
		if wireType != 2 {
			return nil, nil, fmt.Errorf("unexpected wire type %d at field %d in envelope", wireType, fieldNum)
		}
		length, next, e := readVarint(data, i)
		if e != nil {
			return nil, nil, e
		}
		i = next
		end := i + int(length)
		if end > len(data) {
			return nil, nil, fmt.Errorf("truncated field %d in envelope", fieldNum)
		}
		switch fieldNum {
		case 1:
			encryptedContent = make([]byte, length)
			copy(encryptedContent, data[i:end])
		case 2:
			salt = make([]byte, length)
			copy(salt, data[i:end])
		}
		i = end
	}
	if encryptedContent == nil {
		return nil, nil, fmt.Errorf("boot.key envelope missing encrypted_content")
	}
	if salt == nil {
		return nil, nil, fmt.Errorf("boot.key envelope missing salt")
	}
	return encryptedContent, salt, nil
}

// extractStringFields walks decrypted inner proto3 bytes and returns all
// length-delimited string fields indexed by field number.
func extractStringFields(data []byte) (map[int]string, error) {
	result := make(map[int]string)
	for i := 0; i < len(data); {
		tag, next, err := readVarint(data, i)
		if err != nil {
			return nil, err
		}
		i = next
		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case 0: // varint — skip
			_, next, err = readVarint(data, i)
			if err != nil {
				return nil, err
			}
			i = next
		case 1: // 64-bit fixed — skip 8 bytes
			if i+8 > len(data) {
				return nil, fmt.Errorf("truncated fixed64 at field %d", fieldNum)
			}
			i += 8
		case 2: // length-delimited
			length, next, err := readVarint(data, i)
			if err != nil {
				return nil, err
			}
			i = next
			end := i + int(length)
			if end > len(data) {
				return nil, fmt.Errorf("truncated field %d", fieldNum)
			}
			result[fieldNum] = string(data[i:end])
			i = end
		case 5: // 32-bit fixed — skip 4 bytes
			if i+4 > len(data) {
				return nil, fmt.Errorf("truncated fixed32 at field %d", fieldNum)
			}
			i += 4
		default:
			return nil, fmt.Errorf("unsupported wire type %d at field %d", wireType, fieldNum)
		}
	}
	return result, nil
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
