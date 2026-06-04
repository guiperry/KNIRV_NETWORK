package core

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	defaultNetworkAppName  = "knirv"
	defaultOracleBootstrap = "https://oracle.knirv.network"
	networkWalletFileName  = "wallet-address.json"
)

// NetworkWalletRecord stores the bootstrap wallet address in app data.
type NetworkWalletRecord struct {
	Address string `json:"address"`
}

// NetworkWalletStore persists the bootstrap wallet record in app data.
type NetworkWalletStore struct {
	path   string
	logger *logrus.Logger
}

// NewNetworkWalletStore creates a store rooted in the OS application data dir.
func NewNetworkWalletStore(logger *logrus.Logger) (*NetworkWalletStore, error) {
	appDataDir, err := getAppDataDir(defaultNetworkAppName)
	if err != nil {
		return nil, err
	}

	return &NetworkWalletStore{
		path:   filepath.Join(appDataDir, networkWalletFileName),
		logger: logger,
	}, nil
}

// Path returns the underlying file path for the persisted wallet record.
func (s *NetworkWalletStore) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Load returns the persisted wallet address, if present.
func (s *NetworkWalletStore) Load() (string, bool, error) {
	if s == nil {
		return "", false, fmt.Errorf("network wallet store is nil")
	}

	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to read network wallet file: %w", err)
	}

	var record NetworkWalletRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return "", false, fmt.Errorf("failed to decode network wallet file: %w", err)
	}

	address := strings.TrimSpace(record.Address)
	if address == "" {
		return "", false, nil
	}

	return address, true, nil
}

// Save writes the wallet address to app data.
func (s *NetworkWalletStore) Save(address string) error {
	if s == nil {
		return fmt.Errorf("network wallet store is nil")
	}

	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf("wallet address cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("failed to create wallet app data directory: %w", err)
	}

	payload, err := json.MarshalIndent(NetworkWalletRecord{Address: address}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode network wallet record: %w", err)
	}

	if err := os.WriteFile(s.path, payload, 0o600); err != nil {
		return fmt.Errorf("failed to write network wallet file: %w", err)
	}

	if s.logger != nil {
		s.logger.WithField("path", s.path).Info("Persisted network wallet address")
	}

	return nil
}

// Ensure bootstraps a wallet address from the oracle if one is not already stored.
func (s *NetworkWalletStore) Ensure(ctx context.Context, oracleURL string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("network wallet store is nil")
	}

	if address, ok, err := s.Load(); err != nil {
		return "", err
	} else if ok {
		return address, nil
	}

	oracleURL = strings.TrimRight(strings.TrimSpace(oracleURL), "/")
	if oracleURL == "" {
		oracleURL = defaultOracleBootstrap
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oracleURL+"/generate_wallet", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create oracle wallet request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request wallet from oracle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("oracle wallet request failed with status %d", resp.StatusCode)
	}

	var payload struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to decode oracle wallet response: %w", err)
	}

	address := strings.TrimSpace(payload.Address)
	if address == "" {
		return "", fmt.Errorf("oracle returned an empty wallet address")
	}

	if err := s.Save(address); err != nil {
		return "", err
	}

	return address, nil
}

func getAppDataDir(appName string) (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user config directory: %w", err)
	}

	appDataDir := filepath.Join(baseDir, appName)
	if err := os.MkdirAll(appDataDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create app data directory %s: %w", appDataDir, err)
	}

	return appDataDir, nil
}
