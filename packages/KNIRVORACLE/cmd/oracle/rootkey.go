package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/knirvcorp/knirvoracle/internal/oracle"
	"go.uber.org/zap"
	"golang.org/x/crypto/scrypt"
)

const (
	scryptN = 32768
	scryptR = 8
	scryptP = 1
	keyLen  = 32
)

type rootKeyContent struct {
	RootPrivateKeyHex  string
	CloudflareAPIToken string
	CloudflareZoneID   string

	// KNIRVORACLE plan-checkout payment processor (root nodes only — see
	// internal/oracle/payment). Field numbers 1, 2, 29-35 in
	// KNIRV_CORP/packages/server/backend_server/internal/proto/root_key.proto
	// (mirrored in KNIRV_NETWORK/shared-proto/knirvserver/v1/root_key.proto).
	StripeSecretKey              string
	StripeWebhookSecret          string
	PaymentProcessorEnabled      bool
	StripeProfessionalPriceID    string
	StripeEnterprisePriceID      string
	StripeInvestorPriceID        string
	PaymentSuccessURL            string
	PaymentCancelURL             string
	PaymentOnboardingCallbackURL string
}

func initOracleWithSecrets(content *rootKeyContent, logger *zap.Logger) (*oracle.Oracle, error) {
	if content == nil {
		return nil, nil
	}

	rootPrivateKey := content.RootPrivateKeyHex
	if rootPrivateKey == "" {
		logger.Warn("Oracle disabled: root.key decrypted but ROOT_PRIVATE_KEY is empty")
		return nil, nil
	}

	// LoadConfigFromEnv is the fallback layer: with the assumption that
	// KNIRVORACLE only ever runs as a KNIRVSERVER subprocess and root.key is
	// the sole source of configuration, this only matters for local
	// dev/testing without a root.key (or a root.key that predates a given
	// field). Whichever of these fields root.key actually carries below
	// takes precedence over whatever env vars set here.
	oracleCfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to load config from env: %w", err)
	}
	if os.Getenv("ORACLE_DATA_DIR") == "" {
		appDataDir, appDataErr := getOSAppDataDir()
		if appDataErr != nil {
			return nil, fmt.Errorf("oracle: failed to determine app data dir: %w", appDataErr)
		}
		oracleCfg.DataDir = filepath.Join(appDataDir, "oracle")
	}
	oracleCfg.OwnerPrivateKey = rootPrivateKey

	// Payment processor config — root.key is authoritative whenever it
	// carries a value; ORACLE_KEY_PASSWORD remains the only variable an
	// operator sets manually to start KNIRVSERVER at all.
	if content.StripeSecretKey != "" {
		oracleCfg.StripeSecretKey = content.StripeSecretKey
	}
	if content.StripeWebhookSecret != "" {
		oracleCfg.StripeWebhookSecret = content.StripeWebhookSecret
	}
	if content.PaymentProcessorEnabled {
		oracleCfg.PaymentProcessorEnabled = true
	}
	if content.StripeProfessionalPriceID != "" {
		oracleCfg.StripeProfessionalPriceID = content.StripeProfessionalPriceID
	}
	if content.StripeEnterprisePriceID != "" {
		oracleCfg.StripeEnterprisePriceID = content.StripeEnterprisePriceID
	}
	if content.StripeInvestorPriceID != "" {
		oracleCfg.StripeInvestorPriceID = content.StripeInvestorPriceID
	}
	if content.PaymentSuccessURL != "" {
		oracleCfg.PaymentSuccessURL = content.PaymentSuccessURL
	}
	if content.PaymentCancelURL != "" {
		oracleCfg.PaymentCancelURL = content.PaymentCancelURL
	}
	if content.PaymentOnboardingCallbackURL != "" {
		oracleCfg.PaymentOnboardingCallbackURL = content.PaymentOnboardingCallbackURL
	}

	if err := oracle.ValidateConfig(oracleCfg); err != nil {
		return nil, fmt.Errorf("oracle: invalid config: %w", err)
	}

	oracleInstance, err := oracle.NewOracle(oracleCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to create instance: %w", err)
	}

	return oracleInstance, nil
}

func initOracleFromKeyFile(logger *zap.Logger) (*oracle.Oracle, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	keyPath, err := getRootKeyPath()
	if err != nil {
		return nil, fmt.Errorf("oracle: could not resolve root key path: %w", err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logger.Info("Oracle disabled: no root.key found (not a root node)", zap.String("expected_path", keyPath))
		return nil, nil
	}

	var keyPassword []byte
	if envPwd := os.Getenv("ORACLE_KEY_PASSWORD"); envPwd != "" {
		keyPassword = []byte(envPwd)
	} else {
		if !isInteractiveStdin() {
			return nil, fmt.Errorf("oracle: password not provided via ORACLE_KEY_PASSWORD environment variable and stdin is not interactive")
		}
		keyPassword, err = promptForPassword("Enter root key password to start oracle: ")
		if err != nil {
			return nil, fmt.Errorf("oracle: failed to read password: %w", err)
		}
	}

	content, err := loadEncryptedKeyFile(keyPath, keyPassword)
	for i := range keyPassword {
		keyPassword[i] = 0
	}
	if err != nil {
		return nil, fmt.Errorf("oracle: failed to decrypt root.key: %w", err)
	}

	oracleInstance, err := initOracleWithSecrets(content, logger)
	if err != nil {
		return nil, err
	}

	if oracleInstance != nil {
		logger.Info("Oracle initialised from root.key", zap.String("key_path", keyPath))
	}

	return oracleInstance, nil
}

func getRootKeyPath() (string, error) {
	candidates := make([]string, 0, 6)
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

	add(os.Getenv("ORACLE_ROOT_KEY_PATH"))
	add(os.Getenv("KNIRV_ROOT_KEY_PATH"))

	if configDir, err := os.UserConfigDir(); err == nil {
		add(filepath.Join(configDir, "knirv-server", "root.key"))
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(homeDir, ".knirv", "root.key"))
	}

	// os.UserConfigDir() above resolves against the *effective* user — this
	// binary is normally spawned by KNIRVSERVER running under sudo, so that's
	// root's ~/.config, not the operator's. sudo sets SUDO_USER even with
	// env_reset, so check that user's config dir too; if SUDO_USER isn't set
	// (e.g. a systemd unit running as root directly), fall back to scanning
	// every home directory.
	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" {
		add(filepath.Join("/home", sudoUser, ".config", "knirv-server", "root.key"))
	} else if entries, err := os.ReadDir("/home"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				add(filepath.Join("/home", entry.Name(), ".config", "knirv-server", "root.key"))
			}
		}
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		add(filepath.Join(exeDir, "root.key"))
		add(filepath.Join(exeDir, "bin", "root.key"))
	}

	var firstCandidate string
	for i, candidate := range candidates {
		if i == 0 {
			firstCandidate = candidate
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return candidate, nil
	}

	if firstCandidate != "" {
		return firstCandidate, nil
	}

	return "", fmt.Errorf("failed to resolve root.key path")
}

func getOSAppDataDir() (string, error) {
	var appDataDir string

	if explicit := os.Getenv("KNIRV_APP_DATA_DIR"); explicit != "" {
		appDataDir = explicit
	} else if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
		appDataDir = filepath.Join(xdgDataHome, "knirvserver")
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		appDataDir = filepath.Join(homeDir, ".local", "share", "knirvserver")
	} else {
		return "", fmt.Errorf("could not determine application data directory")
	}

	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create application data directory: %w", err)
	}

	return appDataDir, nil
}

func promptForPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := readPasswordNoEcho(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	return []byte(strings.TrimSpace(string(password))), nil
}

func readPasswordNoEcho(fd int) ([]byte, error) {
	state, err := getTermios(fd)
	if err != nil {
		return nil, err
	}

	newState := *state
	newState.Lflag &^= syscall.ECHO
	if err := setTermios(fd, &newState); err != nil {
		return nil, err
	}
	defer func() {
		_ = setTermios(fd, state)
	}()

	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	return []byte(password), nil
}

func getTermios(fd int) (*syscall.Termios, error) {
	termios := &syscall.Termios{}
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return termios, nil
}

func setTermios(fd int, termios *syscall.Termios) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TCSETS), uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

func isInteractiveStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func loadEncryptedKeyFile(keyFilePath string, password []byte) (*rootKeyContent, error) {
	data, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	encryptedContent, salt, err := parseEncryptedRootKeyEnvelope(data)
	if err != nil {
		return nil, err
	}

	key, err := scrypt.Key(password, salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key from password: %w", err)
	}

	decryptedData, err := decrypt(encryptedContent, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key file (incorrect password?): %w", err)
	}

	fields, varints, err := parseProtoFields(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root key content: %w", err)
	}

	rootPrivateKey := string(firstFieldValue(fields[5]))
	if rootPrivateKey == "" {
		return nil, fmt.Errorf("root key content missing root_private_key_hex")
	}

	cloudflareAPIToken := string(firstFieldValue(fields[18]))
	cloudflareZoneID := string(firstFieldValue(fields[19]))

	// Field numbers 1, 2, 29-35 — see root_key.proto in
	// KNIRV_CORP/packages/server/backend_server/internal/proto (mirrored in
	// KNIRV_NETWORK/shared-proto/knirvserver/v1).
	paymentProcessorEnabled := firstVarintValue(varints[29]) != 0

	return &rootKeyContent{
		RootPrivateKeyHex:            rootPrivateKey,
		CloudflareAPIToken:           cloudflareAPIToken,
		CloudflareZoneID:             cloudflareZoneID,
		StripeSecretKey:              string(firstFieldValue(fields[1])),
		StripeWebhookSecret:          string(firstFieldValue(fields[2])),
		PaymentProcessorEnabled:      paymentProcessorEnabled,
		StripeProfessionalPriceID:    string(firstFieldValue(fields[30])),
		StripeEnterprisePriceID:      string(firstFieldValue(fields[31])),
		StripeInvestorPriceID:        string(firstFieldValue(fields[32])),
		PaymentSuccessURL:            string(firstFieldValue(fields[33])),
		PaymentCancelURL:             string(firstFieldValue(fields[34])),
		PaymentOnboardingCallbackURL: string(firstFieldValue(fields[35])),
	}, nil
}

func decrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func parseEncryptedRootKeyEnvelope(data []byte) ([]byte, []byte, error) {
	fields, _, err := parseProtoFields(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse encrypted key file: %w", err)
	}

	encryptedContent := firstFieldValue(fields[1])
	salt := firstFieldValue(fields[2])
	if len(encryptedContent) == 0 {
		return nil, nil, fmt.Errorf("encrypted key file missing encrypted_content")
	}
	if len(salt) == 0 {
		return nil, nil, fmt.Errorf("encrypted key file missing salt")
	}

	return encryptedContent, salt, nil
}

func parseRootPrivateKeyHex(data []byte) (string, error) {
	fields, _, err := parseProtoFields(data)
	if err != nil {
		return "", fmt.Errorf("failed to parse root key content: %w", err)
	}

	rootPrivateKey := string(firstFieldValue(fields[5]))
	if rootPrivateKey == "" {
		return "", fmt.Errorf("root key content missing root_private_key_hex")
	}

	return rootPrivateKey, nil
}

// parseProtoFields walks a proto3 wire-format message without depending on
// generated code (packages/cli and this package are in different Go
// modules/repos from the .proto's canonical home in KNIRV_CORP). Returns
// length-delimited field values (strings/bytes, wire type 2) keyed by field
// number, and varint field values (bools/ints, wire type 0) keyed
// separately — needed for payment_processor_enabled (field 29, bool).
func parseProtoFields(data []byte) (map[int][][]byte, map[int][]uint64, error) {
	fields := make(map[int][][]byte)
	varints := make(map[int][]uint64)

	for i := 0; i < len(data); {
		key, next, err := readVarint(data, i)
		if err != nil {
			return nil, nil, err
		}
		i = next

		fieldNum := int(key >> 3)
		wireType := int(key & 0x7)

		switch wireType {
		case 0:
			value, next, err := readVarint(data, i)
			if err != nil {
				return nil, nil, err
			}
			i = next
			varints[fieldNum] = append(varints[fieldNum], value)
		case 1:
			if i+8 > len(data) {
				return nil, nil, fmt.Errorf("truncated fixed64 field")
			}
			i += 8
		case 2:
			length, next, err := readVarint(data, i)
			if err != nil {
				return nil, nil, err
			}
			i = next
			end := i + int(length)
			if end > len(data) {
				return nil, nil, fmt.Errorf("truncated length-delimited field")
			}
			value := append([]byte(nil), data[i:end]...)
			fields[fieldNum] = append(fields[fieldNum], value)
			i = end
		case 5:
			if i+4 > len(data) {
				return nil, nil, fmt.Errorf("truncated fixed32 field")
			}
			i += 4
		default:
			return nil, nil, fmt.Errorf("unsupported wire type %d", wireType)
		}
	}

	return fields, varints, nil
}

func firstVarintValue(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}

func firstFieldValue(values [][]byte) []byte {
	if len(values) == 0 {
		return nil
	}
	return values[0]
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
