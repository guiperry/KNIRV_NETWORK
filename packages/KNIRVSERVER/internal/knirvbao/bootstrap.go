package knirvbao

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const bootstrapMarker = "knirv-openbao-bootstrap-v1\n"

// recoveryEnvFailPreRename, when non-nil, forces writeRecoveryEnv to fail after the
// temporary credential file is created but before it is renamed into place. Tests use
// it to prove a previously existing recovery file is left untouched on write failure.
var recoveryEnvFailPreRename error

func (m *Manager) initializeChild(parent context.Context, root string, c *child) error {
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	api := newBaoAPI(c.socket)
	defer api.client.CloseIdleConnections()
	var status sealStatus
	if err := pollChild(ctx, c, func() bool {
		var err error
		status, err = api.sealStatus(ctx)
		return err == nil
	}); err != nil {
		return fmt.Errorf("wait for OpenBao socket: %w", err)
	}
	info, err := os.Lstat(c.socket)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 || info.Mode().Perm() != 0600 {
		return fmt.Errorf("OpenBao socket is not owner-only (mode %04o)", info.Mode().Perm())
	}
	c.socketInfo = info

	if !status.Initialized {
		// First boot: initialize, persist the recovery credentials atomically, wait
		// for unseal, then finish bootstrap. If persistence fails the vault is left
		// initialized and recoverable only via operator action.
		initToken, keys, err := api.initialize(ctx)
		if err != nil {
			return err
		}
		if _, err := writeRecoveryEnv(m.config.ConfigDir, root, initToken, keys); err != nil {
			return fmt.Errorf("OpenBao was initialized but the recovery credentials could not be persisted; the vault is left initialized and is recoverable only via operator action: %w", err)
		}
		log.Printf("OpenBao initialized; recovery credentials written to %s — back up the node key file because losing it makes the sealed data unrecoverable", m.recoveryEnvPath())
		if err := waitForUnseal(ctx, api, c); err != nil {
			return err
		}
		return m.runBootstrap(ctx, api, root, initToken)
	}

	if err := waitForUnseal(ctx, api, c); err != nil {
		return err
	}
	return m.recoverOrVerify(ctx, api, root)
}

func waitForUnseal(ctx context.Context, api *baoAPI, c *child) error {
	if err := pollChild(ctx, c, func() bool {
		state, err := api.sealStatus(ctx)
		if err != nil || !state.Initialized || state.Sealed {
			return false
		}
		code, err := api.request(ctx, http.MethodGet, "sys/health", "", nil, nil)
		return err == nil && code == http.StatusOK
	}); err != nil {
		return fmt.Errorf("wait for OpenBao unseal: %w", err)
	}
	return nil
}

// recoverOrVerify resumes an interrupted bootstrap using the persisted credentials
// when the vault is already initialized but the completion marker is absent. A fully
// bootstrapped vault returns immediately. If the credentials are missing or unsafe,
// it returns a precise error instead of silently failing.
func (m *Manager) recoverOrVerify(ctx context.Context, api *baoAPI, root string) error {
	if markerPresent(root) {
		return nil
	}
	token, err := loadRecoveryToken(m.config.ConfigDir, root)
	if err != nil {
		return fmt.Errorf("OpenBao is initialized but its recovery credentials are missing or unsafe; operator recovery is required: %w", err)
	}
	return m.runBootstrap(ctx, api, root, token)
}

func (m *Manager) runBootstrap(ctx context.Context, api *baoAPI, root, token string) error {
	var bootErr error
	for attempt := 0; attempt < 3; attempt++ {
		if bootErr = api.bootstrap(ctx, token, m.beforeFinish); bootErr == nil {
			return writeBootstrapMarker(root)
		}
		if err := waitContext(ctx, time.Second); err != nil {
			return err
		}
	}
	return fmt.Errorf("OpenBao bootstrap incomplete: %w", bootErr)
}

func (m *Manager) recoveryEnvPath() string {
	dir := m.config.ConfigDir
	if dir == "" {
		if root, err := m.config.root(); err == nil {
			dir = root
		}
	}
	return filepath.Join(dir, "openbao.env")
}

func markerPresent(root string) bool {
	data, err := os.ReadFile(bootstrapMarkerPath(root))
	return err == nil && string(data) == bootstrapMarker
}

func bootstrapMarkerPath(root string) string {
	return filepath.Join(root, "bootstrap-complete")
}

// loadRecoveryToken securely reads the persisted root token from openbao.env. It
// refuses symlinks and any non-regular or non-0600 file, so a careless or hostile
// replacement cannot be used to hijack the bootstrap or leak the credential.
func loadRecoveryToken(configDir, root string) (string, error) {
	dir := configDir
	if dir == "" {
		dir = root
	}
	path := filepath.Join(dir, "openbao.env")
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("recovery env not found: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("recovery env is a symlink")
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return "", fmt.Errorf("recovery env is not a private regular file (mode %04o)", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read recovery env: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "OPENBAO_ROOT_TOKEN=") {
			continue
		}
		tok := strings.TrimPrefix(line, "OPENBAO_ROOT_TOKEN=")
		if tok == "" {
			return "", errors.New("recovery env has an empty root token")
		}
		return tok, nil
	}
	return "", errors.New("recovery env has no root token")
}

func pollChild(ctx context.Context, c *child, probe func() bool) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		if probe() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.wait:
			c.exited = true
			return errors.New("OpenBao process exited")
		case <-ticker.C:
		}
	}
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func writeBootstrapMarker(root string) error {
	path := filepath.Join(root, "bootstrap-complete")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(bootstrapMarker); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	dir, err := os.Open(root)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// writeRecoveryEnv records the one-time OpenBao first-boot recovery material so an
// interrupted bootstrap can be finished and later AppRole / recovery work can proceed
// (owner decision OB-R7). It is written atomically to a same-directory temporary file
// created 0600, explicitly chmod'd, synced, and renamed over the target, then the
// directory is synced. The content is never logged, printed, or returned; only the
// path is. An existing symlink or non-regular target is rejected so the write can
// never follow a symlink or clobber an unrelated path.
func writeRecoveryEnv(configDir, root, token string, keys []string) (string, error) {
	dir := configDir
	if dir == "" {
		dir = root
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create OpenBao config dir: %w", err)
	}
	path := filepath.Join(dir, "openbao.env")

	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing to overwrite symlinked OpenBao recovery env %s", path)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("refusing to overwrite non-regular OpenBao recovery env %s", path)
		}
	}

	var b strings.Builder
	b.WriteString("# OpenBao first-boot recovery credentials. DO NOT COMMIT. DO NOT LOG.\n")
	b.WriteString("OPENBAO_ROOT_TOKEN=" + token + "\n")
	for i, key := range keys {
		fmt.Fprintf(&b, "OPENBAO_RECOVERY_KEY_%d=%s\n", i+1, key)
	}
	content := []byte(b.String())

	tmp, err := os.CreateTemp(dir, ".openbao-env-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create OpenBao recovery env temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if tmp != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0600); err != nil {
		return "", fmt.Errorf("chmod OpenBao recovery env temp file: %w", err)
	}
	if _, err := tmp.Write(content); err != nil {
		return "", fmt.Errorf("write OpenBao recovery env temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("sync OpenBao recovery env temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close OpenBao recovery env temp file: %w", err)
	}
	tmp = nil
	if recoveryEnvFailPreRename != nil {
		_ = os.Remove(tmpName)
		return "", recoveryEnvFailPreRename
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", fmt.Errorf("install OpenBao recovery env: %w", err)
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return path, nil
}
