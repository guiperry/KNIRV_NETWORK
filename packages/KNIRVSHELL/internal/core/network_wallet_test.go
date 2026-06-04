package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNetworkWalletStoreSaveLoad(t *testing.T) {
	store := &NetworkWalletStore{
		path:   filepath.Join(t.TempDir(), "wallet-address.json"),
		logger: logrus.New(),
	}

	require.NoError(t, store.Save("0x1234567890abcdef1234567890abcdef12345678"))

	address, ok, err := store.Load()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "0x1234567890abcdef1234567890abcdef12345678", address)
}

func TestNetworkWalletStoreEnsureBootstrapsFromOracle(t *testing.T) {
	requestedPath := make(chan string, 1)
	oracle := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath <- r.URL.Path
		if r.URL.Path != "/generate_wallet" {
			t.Errorf("path = %s, want /generate_wallet", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if err := json.NewEncoder(w).Encode(map[string]string{
			"address": "0xabcdef1234567890abcdef1234567890abcdef12",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer oracle.Close()

	store := &NetworkWalletStore{
		path:   filepath.Join(t.TempDir(), "wallet-address.json"),
		logger: logrus.New(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	address, err := store.Ensure(ctx, oracle.URL)
	require.NoError(t, err)
	require.Equal(t, "0xabcdef1234567890abcdef1234567890abcdef12", address)
	require.Equal(t, "/generate_wallet", <-requestedPath)

	saved, ok, err := store.Load()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, address, saved)
}
