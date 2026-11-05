package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/guiperry/knirv-sdk-go/secrets"
	knirvsigning "github.com/guiperry/knirv-sdk-go/signing"
	"github.com/knirvcorp/knirvoracle/internal/oracle"
	oraclecrypto "github.com/knirvcorp/knirvoracle/internal/oracle/crypto"
	"go.uber.org/zap"
)

type rootKeyContent struct {
	RootPrivateKeyHex  string
	ReplicationSigner  knirvsigning.RemoteSigner
	CloudflareAPIToken string
	CloudflareZoneID   string

	// KNIRVORACLE plan-checkout payment processor (root nodes only — see
	// internal/oracle/payment). Field numbers 1, 2, 29-36 in
	// KNIRV_CORP/packages/server/backend_server/internal/proto/root_key.proto
	// (mirrored in KNIRV_NETWORK/shared-proto/knirvserver/v1/root_key.proto).
	StripeSecretKey              string
	StripeWebhookSecret          string
	StripeThinWebhookSecret      string
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
	oracleCfg.ReplicationSigner = content.ReplicationSigner

	// Payment processor config — root.key is authoritative whenever it
	// carries a value; ORACLE_KEY_PASSWORD remains the only variable an
	// operator sets manually to start KNIRVSERVER at all.
	if content.StripeSecretKey != "" {
		oracleCfg.StripeSecretKey = content.StripeSecretKey
	}
	if content.StripeWebhookSecret != "" {
		oracleCfg.StripeWebhookSecret = content.StripeWebhookSecret
	}
	if content.StripeThinWebhookSecret != "" {
		oracleCfg.StripeThinWebhookSecret = content.StripeThinWebhookSecret
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
	// FileProvider supplies the root-owner signer when the existing environment
	// password handoff is available. Interactive unlock remains on the legacy
	// path until Provider supports a non-environment password handoff.
	content.ReplicationSigner = replicationFileSigner(content, logger)

	oracleInstance, err := initOracleWithSecrets(content, logger)
	if err != nil {
		return nil, err
	}

	if oracleInstance != nil {
		logger.Info("Oracle initialised from root.key", zap.String("key_path", keyPath))
	}

	return oracleInstance, nil
}

func replicationFileSigner(content *rootKeyContent, logger *zap.Logger) knirvsigning.RemoteSigner {
	if os.Getenv("ORACLE_KEY_PASSWORD") != "" {
		signer, signerErr := secrets.NewFileProvider().Signer(context.Background(), "root-owner")
		if signerErr != nil {
			logger.Warn("replication signer unavailable from key-file provider; using existing unlocked identity", zap.Error(signerErr))
		} else {
			existing, keyErr := hex.DecodeString(strings.TrimPrefix(content.RootPrivateKeyHex, "0x"))
			var identity knirvsigning.RemoteSigner
			if keyErr == nil {
				identity, keyErr = knirvsigning.NewLocalSigner(existing)
			}
			for i := range existing {
				existing[i] = 0
			}
			if keyErr != nil || signer.Address() != identity.Address() {
				logger.Warn("replication provider identity differs from the unlocked root key; using existing unlocked identity")
			} else {
				return signer
			}
		}
	}
	return nil
}

// initOracleForMode activates the Oracle according to ORACLE_MODE, which
// KNIRVSERVER renders from noderole.Resolve. Active is the historical path
// (root.key required); standby needs only a boot.key; off starts nothing.
func initOracleForMode(logger *zap.Logger) (*oracle.Oracle, error) {
	mode, err := oracle.NormalizeMode(os.Getenv("ORACLE_MODE"))
	if err != nil {
		return nil, err
	}
	switch mode {
	case oracle.ModeOff:
		logger.Info("Oracle mode is off for this node — not activating")
		return nil, nil
	case oracle.ModeStandby:
		return initStandbyOracle(logger)
	default:
		return initOracleFromKeyFile(logger)
	}
}

// initStandbyOracle builds the read-only mirror. No engine that
// signs, votes, gossips or disburses is started; the HTTP guard refuses every
// route outside the standby allowlist.
//
// The token constructor requires an owner address. On a bootnode that is
// boot.key's master wallet key. If boot.key is absent or locked the standby
// must still start (it is a failover ledger, not an authority), so it uses an
// in-memory identity that is never persisted and never used to sign anything.
func initStandbyOracle(logger *zap.Logger) (*oracle.Oracle, error) {
	cfg, err := oracle.LoadConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("standby: load config: %w", err)
	}
	cfg.Mode = oracle.ModeStandby

	boot, bootErr := secrets.NewFileProvider().Boot(context.Background())
	if bootErr == nil && boot != nil && boot.MasterWalletKeyHex != "" {
		cfg.OwnerPrivateKey = boot.MasterWalletKeyHex
		logger.Info("Standby Oracle identity: bootnode-master from boot.key")
	} else {
		key, keyErr := oraclecrypto.GenerateKeyPair()
		if keyErr != nil {
			return nil, fmt.Errorf("standby: create in-memory identity: %w", keyErr)
		}
		cfg.OwnerPrivateKey = key.PrivateKeyHex()
		logger.Warn("Standby Oracle started without a usable boot.key; using an in-memory identity that never signs",
			zap.Error(bootErr))
	}

	instance, err := oracle.NewOracle(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("standby: %w", err)
	}
	logger.Info("Oracle initialised in standby mode", zap.String("data_dir", cfg.DataDir))
	return instance, nil
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
	fields, err := secrets.DecryptFields(keyFilePath, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt key file (incorrect password?): %w", err)
	}

	rootPrivateKey := fields.String(5)
	if rootPrivateKey == "" {
		return nil, fmt.Errorf("root key content missing root_private_key_hex")
	}

	return &rootKeyContent{
		RootPrivateKeyHex:            rootPrivateKey,
		CloudflareAPIToken:           fields.String(18),
		CloudflareZoneID:             fields.String(19),
		StripeSecretKey:              fields.String(1),
		StripeWebhookSecret:          fields.String(2),
		PaymentProcessorEnabled:      fields.Bool(29),
		StripeProfessionalPriceID:    fields.String(30),
		StripeEnterprisePriceID:      fields.String(31),
		StripeInvestorPriceID:        fields.String(32),
		PaymentSuccessURL:            fields.String(33),
		PaymentCancelURL:             fields.String(34),
		PaymentOnboardingCallbackURL: fields.String(35),
		StripeThinWebhookSecret:      fields.String(36),
	}, nil
}
